package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"115localnatmanager/api"
	"115localnatmanager/config"
	"115localnatmanager/handler"
	"115localnatmanager/service"
)

func main() {
	dataDir := flag.String("data", getDefaultDataDir(), "Data directory path")
	port := flag.String("port", "", "Server port (overrides config)")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting 115 Local NAT Manager...")
	log.Printf("Data directory: %s", *dataDir)

	cfg := config.NewManager(*dataDir)
	logger := service.NewLogger(1000)

	if *port != "" {
		cfg.UpdateConfig(config.ConfigUpdateRequest{Port: *port})
	}

	client := api.NewClient(cfg)
	tokenMgr := service.NewTokenManager(client, cfg)
	monitor := service.NewDownloadMonitor(client, cfg, logger)

	tokenMgr.Start()

	appConfig := cfg.GetConfig()
	if appConfig.MonitorEnabled {
		monitor.Start()
	}

	router := handler.NewRouter(client, cfg, monitor, logger)

	addr := ":" + appConfig.Port
	log.Printf("Server listening on %s", addr)
	log.Printf("Download directory: %s", appConfig.DownloadDir)
	logger.Info("Server started on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
		logger.Error("Server failed: %v", err)
		os.Exit(1)
	}
}

func getDefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".115manager"
	}
	return filepath.Join(home, ".115manager")
}
