package main

import (
	"flag"

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

	logger.Info("NetMap starting")

	configPath := flag.String(
		"config",
		app.ConfigPath("client.json"),
		"config file path",
	)

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error(
			"load config failed: %v",
			err,
		)
		return
	}

	logger.Info(
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
	logger.Info("starting client server")

	server := client.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {
		startError <- server.Start()
	}()

	select {
	case <-server.Started():
		logger.Info("client server started")

		tray.SetServerRunning(true)

		tray.Start("client", func() {
			logger.Info("stopping NetMap")

			server.Stop()

			logger.Info("NetMap stopped")
		})

	case err := <-startError:
		logger.Error(
			"start client server failed: %v",
			err,
		)

		tray.Start("client", func() {
			logger.Info("stopping NetMap")
		})
	}
}

// startRelay 启动 Relay。
func startRelay(cfg *config.Config) {
	logger.Info("starting relay server")

	server := relay.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {
		startError <- server.Start()
	}()

	select {
	case <-server.Started():
		logger.Info("relay server started")

		tray.SetServerRunning(true)

		tray.Start("relay", func() {
			logger.Info("stopping NetMap")

			server.Stop()

			logger.Info("NetMap stopped")
		})

	case err := <-startError:
		logger.Error(
			"start relay server failed: %v",
			err,
		)

		tray.Start("relay", func() {
			logger.Info("stopping NetMap")
		})
	}
}
