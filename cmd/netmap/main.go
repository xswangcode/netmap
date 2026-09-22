package main

import (
	"flag"
	"log"

	"netmap/internal/client"
	"netmap/internal/config"
	"netmap/internal/relay"
)

func main() {
	configPath := flag.String("config", "", "NetMap configuration file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("config file is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	switch cfg.Role {
	case "client":
		server := client.NewServer(cfg)

		if err := server.Start(); err != nil {
			log.Fatal(err)
		}

	case "relay":
		server := relay.NewServer(cfg)

		if err := server.Start(); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatalf("unsupported role: %s", cfg.Role)
	}
}
