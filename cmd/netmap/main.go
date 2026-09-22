package main

import (
	"flag"
	"log"

	"netmap/internal/app"
	"netmap/internal/client"
	"netmap/internal/config"
	"netmap/internal/logger"
	"netmap/internal/relay"
	"netmap/internal/singleinstance"
	"netmap/internal/tray"
)

func main() {
	instance, alreadyRunning, err := singleinstance.Acquire(
		"NetMap.SingleInstance",
	)
	if err != nil {
		return
	}

	if alreadyRunning {
		return
	}

	defer instance.Release()

	if err := logger.Init(); err != nil {
		return
	}

	defer logger.Close()

	log.Println("NetMap starting")

	configPath := flag.String(
		"config",
		app.ConfigPath("client.json"),
		"config file path",
	)

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf(
			"load config failed: %v",
			err,
		)
		return
	}

	log.Printf(
		"config loaded role=%s",
		cfg.Role,
	)

	if cfg.Role == "relay" {
		startRelay(cfg)
		return
	}

	startClient(cfg)
}

// startClient 启动 Client。
func startClient(cfg *config.Config) {
	log.Println("starting client server")

	server := client.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {
		startError <- server.Start()
	}()

	select {
	case <-server.Started():
		log.Println("client server started")

		tray.SetServerRunning(true)

		tray.Start("client", func() {
			log.Println("stopping NetMap")

			server.Stop()

			log.Println("NetMap stopped")
		})

	case err := <-startError:
		log.Printf(
			"start client failed: %v",
			err,
		)

		tray.Start("client", func() {
			log.Println("stopping NetMap")
		})
	}
}

// startRelay 启动 Relay。
func startRelay(cfg *config.Config) {
	log.Println("starting relay server")

	server := relay.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {
		startError <- server.Start()
	}()

	select {
	case <-server.Started():
		log.Println("relay server started")

		tray.SetServerRunning(true)

		tray.Start("relay", func() {
			log.Println("stopping NetMap")

			server.Stop()

			log.Println("NetMap stopped")
		})

	case err := <-startError:
		log.Printf(
			"start relay failed: %v",
			err,
		)

		tray.Start("relay", func() {
			log.Println("stopping NetMap")
		})
	}
}
