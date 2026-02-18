package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/admin"
	"github.com/navantesolutions/apimcore/internal/devportal"
	"github.com/navantesolutions/apimcore/internal/gateway"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/store"
	"github.com/navantesolutions/apimcore/internal/tui"
)

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

func main() {
	useTUI := flag.Bool("tui", false, "Enable interactive TUI monitor")
	flag.Parse()

	configPath := os.Getenv("APIM_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st := store.NewStore()
	st.PopulateFromConfig(cfg)

	reg := prometheus.NewRegistry()
	m := meter.New(st, reg)
	gw := gateway.New(cfg, st, m)

	// Watch for config changes
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
		log.Printf("apim gateway listening on %s", cfg.Gateway.Listen)
		if err := http.ListenAndServe(cfg.Gateway.Listen, gatewayMux); err != nil {
			log.Fatalf("gateway: %v", err)
		}
	}()

	if *useTUI {
		var p *tea.Program
		onReload := func() {
			newCfg, err := config.Load(configPath)
			if err == nil {
				st.PopulateFromConfig(newCfg)
				gw.UpdateConfig(newCfg)
			}
		}

		tuiModel := tui.NewModel(onReload)
		p = tea.NewProgram(tuiModel, tea.WithAltScreen())
		log.SetOutput(&tuiWriter{p: p})

		// Metrics updater
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			for range ticker.C {
				statsTotal, _, _ := m.StatsSince(time.Now().Add(-1 * time.Hour))
				p.Send(tui.MetricsUpdateMsg{
					TotalRequests: statsTotal,
					AvgLatency:    0, // TODO: calculate from mtr
				})
			}
		}()

		// Traffic stream
		go func() {
			for pkt := range gw.TrafficChan {
				p.Send(tui.TrafficPacket{
					Timestamp: pkt.Timestamp,
					Method:    pkt.Method,
					Path:      pkt.Path,
					Backend:   pkt.Backend,
					Status:    pkt.Status,
					Latency:   pkt.Latency,
					TenantID:  pkt.TenantID,
				})
			}
		}()

		go func() {
			log.Printf("apim server listening on %s (admin, devportal, metrics)", cfg.Server.Listen)
			if err := http.ListenAndServe(cfg.Server.Listen, serverMux); err != nil {
				log.Fatalf("server: %v", err)
			}
		}()

		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v", err)
			os.Exit(1)
		}
	} else {
		log.Printf("apim server listening on %s (admin, devportal, metrics)", cfg.Server.Listen)
		if err := http.ListenAndServe(cfg.Server.Listen, serverMux); err != nil {
			log.Fatalf("server: %v", err)
		}
	}
}
