package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/navantesolutions/apimcore/config"
	"github.com/navantesolutions/apimcore/internal/admin"
	"github.com/navantesolutions/apimcore/internal/devportal"
	"github.com/navantesolutions/apimcore/internal/gateway"
	"github.com/navantesolutions/apimcore/internal/meter"
	"github.com/navantesolutions/apimcore/internal/store"
)

//go:embed all:web/devportal
var devportalFS embed.FS

func main() {
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

	log.Printf("apim server listening on %s (admin, devportal, metrics)", cfg.Server.Listen)
	if err := http.ListenAndServe(cfg.Server.Listen, serverMux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
