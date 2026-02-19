package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/admin"
	"github.com/navantesolutions/apimcore/internal/devportal"
	"github.com/navantesolutions/apimcore/internal/gateway"
	"github.com/navantesolutions/apimcore/internal/hub"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/securitylog"
	"github.com/navantesolutions/apimcore/internal/store"
	"github.com/navantesolutions/apimcore/internal/tui"
)

func printHelp() {
	fmt.Fprintf(os.Stderr, `ApimCore - API gateway with config-driven products, subscriptions, and security.

USAGE
  apimcore [OPTIONS]

OPTIONS
  -f, -config PATH     Config file path. Default: config.yaml, or APIM_CONFIG env.
  -tui                 Start the interactive TUI (traffic, geo, config, system).
  -hot-reload          Watch config file and reload on change. Without this, use [R] in TUI or restart to apply changes.
  -use-db               Persist BLOCKED/RATE_LIMIT events to SQLite at ./data/apimcore.db (creates dir if needed).
  -use-file-log PATH    Persist BLOCKED/RATE_LIMIT events to a JSONL file at PATH. Ignored if -use-db is set.
  -h, -help             Show this help and exit.

ENVIRONMENT
  APIM_CONFIG          Config file path when -f is not set.
  APIM_FILE_LOG        Path to JSONL file when -use-file-log is not set (same as -use-file-log).
  APIM_GATEWAY_LISTEN   Override gateway.listen from config.
  APIM_SERVER_LISTEN    Override server.listen from config.

EXAMPLES
  apimcore
  apimcore -f ./config/prod.yaml
  apimcore --tui
  apimcore -f config.yaml -tui -hot-reload
  apimcore -use-db
  apimcore -use-db --tui
  apimcore -use-file-log=apim-security.jsonl --tui

DOCUMENTATION
  See docs/ for configuration, deployment, and architecture.
`)
}

type tuiWriter struct {
	p *tea.Program
}

func (tw *tuiWriter) Write(p []byte) (n int, err error) {
	if tw.p != nil {
		tw.p.Send(tui.LogMsg(string(p)))
	}
	return len(p), nil
}

//go:embed all:web/devportal
var devportalFS embed.FS

var processStartTime = time.Now()

func main() {
	flag.Usage = printHelp
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printHelp()
			os.Exit(0)
		}
	}
	var configPath string
	flag.StringVar(&configPath, "f", "", "Path to config file (default: config.yaml or APIM_CONFIG)")
	flag.StringVar(&configPath, "config", "", "Path to config file (same as -f)")
	hotReload := flag.Bool("hot-reload", false, "Watch config file and reload on change (use [R] in TUI for manual reload)")
	useTUI := flag.Bool("tui", false, "Enable interactive TUI monitor")
	useDB := flag.Bool("use-db", false, "Persist security events to SQLite at ./data/apimcore.db")
	useFileLog := flag.String("use-file-log", "", "Persist security events to JSONL file at PATH (ignored if -use-db)")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("APIM_CONFIG")
	}
	if configPath == "" {
		configPath = "config.yaml"
	}

	var cfg *config.Config
	noConfigFile := false
	if _, err := os.Stat(configPath); err != nil && os.IsNotExist(err) {
		log.Printf("config file not found: %s (using defaults; gateway will run with no products)", configPath)
		cfg = config.Default()
		noConfigFile = true
	} else {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
	}

	st := store.NewStore()
	st.PopulateFromConfig(cfg)

	reg := prometheus.NewRegistry()
	m := meter.New(st, reg)
	hb := hub.NewBroadcaster()
	gw := gateway.New(cfg, st, m, hb)

	securityLogPath := ""
	if *useDB {
		dbPath := "data/apimcore.db"
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			log.Printf("could not create data dir for -use-db: %v", err)
			securityLogPath = ""
		} else {
			securityLogPath = "sqlite:" + dbPath
		}
	} else {
		securityLogPath = *useFileLog
		if securityLogPath == "" {
			securityLogPath = os.Getenv("APIM_FILE_LOG")
		}
	}
	secLog, err := securitylog.New(securityLogPath)
	if err != nil {
		log.Printf("security log: %v (events will not be persisted)", err)
	} else if secLog != nil {
		defer secLog.Close()
		log.Printf("security events logged to %s", securityLogPath)
	}

	var tuiTrafficChan chan hub.TrafficEvent
	if *useTUI {
		tuiTrafficChan = make(chan hub.TrafficEvent, 100)
	}
	go func() {
		for ev := range hb.TrafficChan() {
			if secLog != nil && (ev.Action == "BLOCKED" || ev.Action == "RATE_LIMIT") {
				secLog.Append(ev)
			}
			if tuiTrafficChan != nil {
				select {
				case tuiTrafficChan <- ev:
				default:
				}
			}
		}
	}()

	if *hotReload {
		go func() {
			lastMod := time.Now()
			for {
				time.Sleep(5 * time.Second)
				info, err := os.Stat(configPath)
				if err != nil {
					continue
				}
				if info.ModTime().After(lastMod) {
					log.Printf("config file changed, reloading...")
					newCfg, err := config.Load(configPath)
					if err != nil {
						log.Printf("failed to reload config: %v", err)
						continue
					}
					st.PopulateFromConfig(newCfg)
					gw.UpdateConfig(newCfg)
					lastMod = info.ModTime()
				}
			}
		}()
	}
	gatewayMux := http.NewServeMux()
	gatewayMux.Handle("/", gw)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	serverMux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	serverMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	adminHandler := admin.New(st, "/api/admin")
	adminHandler.Register(serverMux)
	devPortalHandler := devportal.New(st, "/devportal")
	devPortalHandler.Register(serverMux)
	dpFS, _ := fs.Sub(devportalFS, "web/devportal")
	serverMux.Handle("/devportal/", http.StripPrefix("/devportal", http.FileServer(http.FS(dpFS))))

	go func() {
		log.Printf("apimcore gateway listening on %s", cfg.Gateway.Listen)
		if err := http.ListenAndServe(cfg.Gateway.Listen, gatewayMux); err != nil {
			log.Fatalf("gateway: %v", err)
		}
	}()

	if *useTUI {
		nodeID := os.Getenv("APIM_NODE_ID")
		if nodeID == "" {
			nodeID, _ = os.Hostname()
		}
		if nodeID == "" {
			nodeID = "local"
		}
		clusterNodes := os.Getenv("APIM_CLUSTER_NODES")
		if clusterNodes == "" {
			clusterNodes = "1"
		}

		var p *tea.Program
		tuiModel := tui.NewModel(func() bool {
			newCfg, err := config.Load(configPath)
			if err != nil {
				return false
			}
			gw.UpdateConfig(newCfg)
			st.PopulateFromConfig(newCfg)
			return true
		}, st, gw, hb, configPath, nodeID, clusterNodes, processStartTime, noConfigFile, *hotReload)

		p = tea.NewProgram(tuiModel, tea.WithAltScreen())
		log.SetOutput(&tuiWriter{p: p})

		go func() {
			ticker := time.NewTicker(2 * time.Second)
			since := time.Now().Add(-1 * time.Hour)
			for range ticker.C {
				statsTotal, _, _ := m.StatsSince(since)
				avgLat := m.AvgLatencySince(since)
				blocked, rateLimited := gw.Stats()

				cpuPct := 0.0
				if percents, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(percents) > 0 {
					cpuPct = percents[0] / 100.0
				}
				memUsedMB := uint64(0)
				memTotalMB := uint64(0)
				if v, err := mem.VirtualMemory(); err == nil {
					memUsedMB = v.Used / (1024 * 1024)
					memTotalMB = v.Total / (1024 * 1024)
				}

				p.Send(tui.MetricsUpdateMsg{
					TotalRequests: statsTotal,
					AvgLatency:    avgLat,
				})
				p.Send(hub.SystemStats{
					TotalRequests: statsTotal,
					AvgLatency:    avgLat,
					RateLimited:   rateLimited,
					Blocked:       blocked,
					Uptime:        time.Since(processStartTime),
					CPUUsage:      cpuPct,
					MemoryUsageMB: memUsedMB,
					MemoryTotalMB: memTotalMB,
				})
			}
		}()

		go func() {
			for pkt := range tuiTrafficChan {
				p.Send(pkt)
			}
		}()

		go func() {
			log.Printf("apimcore server listening on %s (admin, devportal, metrics)", cfg.Server.Listen)
			if err := http.ListenAndServe(cfg.Server.Listen, serverMux); err != nil {
				log.Fatalf("server: %v", err)
			}
		}()

		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v", err)
			os.Exit(1)
		}
	} else {
		log.Printf("apimcore server listening on %s (admin, devportal, metrics)", cfg.Server.Listen)
		if err := http.ListenAndServe(cfg.Server.Listen, serverMux); err != nil {
			log.Fatalf("server: %v", err)
		}
	}
}
