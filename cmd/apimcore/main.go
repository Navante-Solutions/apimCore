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

const (
	DefaultConfigPath       = "config.yaml"
	HotReloadInterval       = 5 * time.Second
	MetricsTickerInterval   = 2 * time.Second
	TuiTrafficBatchBuffer   = 100
	TuiTrafficBatchSize     = 50
	TuiTrafficBatchInterval = 40 * time.Millisecond
	MetricsHistorySince     = -1 * time.Hour
	CPUPercentSampleDur     = 100 * time.Millisecond
	DefaultDBPath           = "data/apimcore.db"
	NodeIDEnv               = "APIM_NODE_ID"
	ClusterNodesEnv         = "APIM_CLUSTER_NODES"
	DefaultNodeID           = "local"
	DefaultClusterNodes     = "1"
)

//go:embed all:web/devportal
var devportalFS embed.FS

var processStartTime = time.Now()

type appFlags struct {
	configPath string
	hotReload  bool
	useTUI        bool
	useDB         bool
	useFileLog    string
	useFileLogAll string
}

func parseFlags() appFlags {
	flag.Usage = printHelp
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printHelp()
			os.Exit(0)
		}
	}
	var f appFlags
	flag.StringVar(&f.configPath, "f", "", "Path to config file (default: config.yaml or APIM_CONFIG)")
	flag.StringVar(&f.configPath, "config", "", "Path to config file (same as -f)")
	flag.BoolVar(&f.hotReload, "hot-reload", false, "Watch config file and reload on change")
	flag.BoolVar(&f.useTUI, "tui", false, "Enable interactive TUI monitor")
	flag.BoolVar(&f.useDB, "use-db", false, "Persist security events to SQLite at ./data/apimcore.db")
	flag.StringVar(&f.useFileLog, "use-file-log", "", "Persist security events (BLOCKED/RATE_LIMIT only) to JSONL at PATH (ignored if -use-db)")
	flag.StringVar(&f.useFileLogAll, "file-log-all", "", "Persist ALL traffic to JSONL at PATH (for debugging/perf tests)")
	flag.Parse()
	if f.configPath == "" {
		f.configPath = os.Getenv("APIM_CONFIG")
	}
	if f.configPath == "" {
		f.configPath = DefaultConfigPath
	}
	return f
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `ApimCore - API gateway with config-driven products, subscriptions, and security.

USAGE
  apimcore [OPTIONS]

OPTIONS
  -f, -config PATH     Config file path. Default: config.yaml, or APIM_CONFIG env.
  -tui                 Start the interactive TUI (traffic, geo, config, system).
  -hot-reload          Watch config file and reload on change. Without this, use [R] in TUI or restart to apply changes.
  -use-db               Persist BLOCKED/RATE_LIMIT events to SQLite at ./data/apimcore.db (creates dir if needed).
  -use-file-log PATH    Persist only BLOCKED/RATE_LIMIT to JSONL at PATH. Ignored if -use-db is set.
  -file-log-all PATH    Persist ALL traffic (every request) to JSONL at PATH. Use for debugging or perf tests.
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

func loadConfig(path string) (cfg *config.Config, noConfigFile bool) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		log.Printf("config file not found: %s (using defaults; gateway will run with no products)", path)
		return config.Default(), true
	}
	var err error
	cfg, err = config.Load(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	return cfg, false
}

func setupPersistence(useDB bool, useFileLog string) (securitylog.Logger, string) {
	if useDB {
		if err := os.MkdirAll(filepath.Dir(DefaultDBPath), 0755); err != nil {
			log.Printf("could not create data dir for -use-db: %v", err)
			return nil, ""
		}
		l, err := securitylog.New("sqlite:" + DefaultDBPath)
		if err != nil {
			log.Printf("security log: %v (events will not be persisted)", err)
			return nil, ""
		}
		return l, "sqlite:" + DefaultDBPath
	}
	path := useFileLog
	if path == "" {
		path = os.Getenv("APIM_FILE_LOG")
	}
	if path == "" {
		return nil, ""
	}
	l, err := securitylog.New(path)
	if err != nil {
		log.Printf("security log: %v (events will not be persisted)", err)
		return nil, ""
	}
	return l, path
}

func setupMuxes(st *store.Store, gw *gateway.Gateway, reg *prometheus.Registry) (gatewayMux, serverMux *http.ServeMux) {
	gatewayMux = http.NewServeMux()
	gatewayMux.Handle("/", gw)

	serverMux = http.NewServeMux()
	serverMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	serverMux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	serverMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	admin.New(st, "/api/admin", gw).Register(serverMux)
	devportal.New(st, "/devportal").Register(serverMux)
	dpFS, _ := fs.Sub(devportalFS, "web/devportal")
	serverMux.Handle("/devportal/", http.StripPrefix("/devportal", http.FileServer(http.FS(dpFS))))
	return gatewayMux, serverMux
}

func runGateway(listen string, mux *http.ServeMux) {
	log.Printf("apimcore gateway listening on %s", listen)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("gateway: %v", err)
	}
}

func runManagementServer(listen string, mux *http.ServeMux) {
	log.Printf("apimcore server listening on %s (admin, devportal, metrics)", listen)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

type tuiWriter struct{ p *tea.Program }

func (tw *tuiWriter) Write(p []byte) (n int, err error) {
	if tw.p != nil {
		tw.p.Send(tui.LogMsg(string(p)))
	}
	return len(p), nil
}

func runTUI(opts struct {
	cfg          *config.Config
	st           *store.Store
	gw           *gateway.Gateway
	hb           *hub.Broadcaster
	m            *meter.Meter
	configPath   string
	noConfigFile bool
	hotReload    bool
	trafficChan  chan []hub.TrafficEvent
	serverMux    *http.ServeMux
	serverListen string
}) {
	nodeID := os.Getenv(NodeIDEnv)
	if nodeID == "" {
		nodeID, _ = os.Hostname()
	}
	if nodeID == "" {
		nodeID = DefaultNodeID
	}
	clusterNodes := os.Getenv(ClusterNodesEnv)
	if clusterNodes == "" {
		clusterNodes = DefaultClusterNodes
	}

	onReload := func() bool {
		newCfg, err := config.Load(opts.configPath)
		if err != nil {
			return false
		}
		opts.gw.UpdateConfig(newCfg)
		opts.st.PopulateFromConfig(newCfg)
		return true
	}
	model := tui.NewModel(onReload, opts.st, opts.gw, opts.hb, opts.configPath, nodeID, clusterNodes, processStartTime, opts.noConfigFile, opts.hotReload)
	p := tea.NewProgram(model, tea.WithAltScreen())
	log.SetOutput(&tuiWriter{p: p})

	go func() {
		ticker := time.NewTicker(MetricsTickerInterval)
		since := time.Now().Add(MetricsHistorySince)
		for range ticker.C {
			statsTotal, _, _ := opts.m.StatsSince(since)
			avgLat := opts.m.AvgLatencySince(since)
			blocked, rateLimited := opts.gw.Stats()
			cpuPct := 0.0
			if percents, err := cpu.Percent(CPUPercentSampleDur, false); err == nil && len(percents) > 0 {
				cpuPct = percents[0] / 100.0
			}
			memUsedMB := uint64(0)
			memTotalMB := uint64(0)
			if v, err := mem.VirtualMemory(); err == nil {
				memUsedMB = v.Used / (1024 * 1024)
				memTotalMB = v.Total / (1024 * 1024)
			}
			p.Send(tui.MetricsUpdateMsg{TotalRequests: statsTotal, AvgLatency: avgLat})
			p.Send(hub.SystemStats{
				TotalRequests: statsTotal, AvgLatency: avgLat,
				RateLimited: rateLimited, Blocked: blocked,
				Uptime: time.Since(processStartTime), CPUUsage: cpuPct,
				MemoryUsageMB: memUsedMB, MemoryTotalMB: memTotalMB,
			})
		}
	}()

	go func() {
		for batch := range opts.trafficChan {
			if len(batch) > 0 {
				p.Send(tui.TrafficBatchMsg(batch))
			}
		}
	}()

	go runManagementServer(opts.serverListen, opts.serverMux)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
		os.Exit(1)
	}
}

func main() {
	flags := parseFlags()

	cfg, noConfigFile := loadConfig(flags.configPath)
	st := store.NewStore()
	st.PopulateFromConfig(cfg)

	reg := prometheus.NewRegistry()
	m := meter.New(st, reg)
	hb := hub.NewBroadcaster()
	gw := gateway.New(cfg, st, m, hb)

	secLog, secLogPath := setupPersistence(flags.useDB, flags.useFileLog)
	if secLog != nil {
		defer secLog.Close()
		log.Printf("security events logged to %s", secLogPath)
	}

	var allTrafficLog securitylog.Logger
	if flags.useFileLogAll != "" {
		var err error
		allTrafficLog, err = securitylog.NewFileLoggerAll(flags.useFileLogAll)
		if err != nil {
			log.Printf("file-log-all: %v", err)
		} else {
			defer allTrafficLog.Close()
			log.Printf("all traffic logged to %s", flags.useFileLogAll)
		}
	}

	var tuiTrafficChan chan []hub.TrafficEvent
	if flags.useTUI {
		tuiTrafficChan = make(chan []hub.TrafficEvent, TuiTrafficBatchBuffer)
	}
	go func() {
		var batch []hub.TrafficEvent
		ticker := time.NewTicker(TuiTrafficBatchInterval)
		for {
			select {
			case ev, ok := <-hb.TrafficChan():
				if !ok {
					if tuiTrafficChan != nil && len(batch) > 0 {
						tuiTrafficChan <- append([]hub.TrafficEvent(nil), batch...)
					}
					return
				}
				if secLog != nil && (ev.Action == "BLOCKED" || ev.Action == "RATE_LIMIT") {
					secLog.Append(ev)
				}
				if allTrafficLog != nil {
					allTrafficLog.Append(ev)
				}
				if tuiTrafficChan != nil {
					batch = append(batch, ev)
					if len(batch) >= TuiTrafficBatchSize {
						tuiTrafficChan <- append([]hub.TrafficEvent(nil), batch...)
						batch = nil
					}
				}
			case <-ticker.C:
				if tuiTrafficChan != nil && len(batch) > 0 {
					tuiTrafficChan <- append([]hub.TrafficEvent(nil), batch...)
					batch = nil
				}
			}
		}
	}()

	if flags.hotReload {
		go func() {
			lastMod := time.Now()
			for {
				time.Sleep(HotReloadInterval)
				info, err := os.Stat(flags.configPath)
				if err != nil {
					continue
				}
				if info.ModTime().After(lastMod) {
					log.Printf("config file changed, reloading...")
					newCfg, err := config.Load(flags.configPath)
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

	gatewayMux, serverMux := setupMuxes(st, gw, reg)
	go runGateway(cfg.Gateway.Listen, gatewayMux)

	if flags.useTUI {
		runTUI(struct {
			cfg          *config.Config
			st           *store.Store
			gw           *gateway.Gateway
			hb           *hub.Broadcaster
			m            *meter.Meter
			configPath   string
			noConfigFile bool
			hotReload    bool
			trafficChan  chan []hub.TrafficEvent
			serverMux    *http.ServeMux
			serverListen string
		}{
			cfg: cfg, st: st, gw: gw, hb: hb, m: m,
			configPath: flags.configPath, noConfigFile: noConfigFile, hotReload: flags.hotReload,
			trafficChan: tuiTrafficChan, serverMux: serverMux, serverListen: cfg.Server.Listen,
		})
	} else {
		runManagementServer(cfg.Server.Listen, serverMux)
	}
}
