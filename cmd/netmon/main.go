package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"netmon/internal/monitor"
	"netmon/internal/server"
	"netmon/internal/store"
	"netmon/web"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
var version = "vX.Y.Z"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("netmon", version)
			os.Exit(0)
		case "--self-test", "-test", "--test", "test":
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			results := monitor.RunSelfTest(ctx)
			monitor.PrintSelfTestReport(results)
			for _, cat := range results {
				for _, t := range cat.Tests {
					if t.Status == monitor.TestFail {
						os.Exit(1)
					}
				}
			}
			os.Exit(0)
		}
	}

	portFlag := flag.String("port", "", "Server HTTP port (default 8080 or PORT env var)")
	selfTestFlag := flag.Bool("self-test", false, "Run diagnostic self-test suite and exit")
	flag.Parse()

	if *selfTestFlag {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		results := monitor.RunSelfTest(ctx)
		monitor.PrintSelfTestReport(results)
		for _, cat := range results {
			for _, t := range cat.Tests {
				if t.Status == monitor.TestFail {
					os.Exit(1)
				}
			}
		}
		os.Exit(0)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	s, err := store.New("netmon.db")
	if err != nil {
		log.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	// Priority: DB saved config > config.json > hardcoded defaults.
	cfg, err := monitor.LoadConfigFile("config.json")
	if err != nil {
		log.Error("config file error, using defaults", "error", err)
		cfg = monitor.DefaultConfig()
	} else {
		log.Info("loaded config from config.json")
	}
	if cs, err := s.GetConfig(); err == nil {
		cfg = monitor.ConfigFromStore(cs)
		log.Info("loaded config from database")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon := monitor.New(cfg, s, log)
	mon.Start(ctx)

	h := server.NewHandler(ctx, s, mon, version)

	distFS, err := fs.Sub(web.FS, "dist")
	if err != nil {
		log.Error("failed to sub web FS", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/data", h.GetData)
	mux.HandleFunc("GET /api/latest", h.GetData)
	mux.HandleFunc("GET /api/config", h.GetConfig)
	mux.HandleFunc("POST /api/config", h.SaveConfig)
	mux.HandleFunc("POST /api/speedtest/run", h.TriggerSpeedTest)
	mux.HandleFunc("GET /api/version", h.GetVersion)
	mux.HandleFunc("POST /api/update", h.TriggerUpdate)
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// Resolve port: flag > env > default 8080
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8081"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	srv := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Trap termination signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("starting server", "addr", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped with error", "error", err)
		}
	}()

	<-sigChan
	log.Info("shutdown signal received, closing server...")

	// Cancel monitor workers
	cancel()

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)

	mon.Wait()
	log.Info("netmon exited cleanly")
}
