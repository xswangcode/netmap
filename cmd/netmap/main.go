package main

import (
	"flag"

	"netmap/internal/admin"
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
		"NetMap.SingleInstance.debug",
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

	cfgRole, err := config.LoadRole(*configPath)
	if err != nil {
		logger.Error(
			"load config failed: %v",
			err,
		)

		return
	}

	logger.Info(
		"config loaded role=%s",
		cfgRole.GetRole(),
	)

	// 初始化 Web 管理配置缓存。
	store, err := admin.NewConfigStore(
		*configPath,
		cfgRole,
	)
	if err != nil {
		logger.Error(
			"initialize admin config store failed: %v",
			err,
		)

		return
	}

	adminServer := admin.NewServer(store)

	go func() {

		if err := adminServer.Start(); err != nil {

			logger.Error(
				"admin web server failed: %v",
				err,
			)
		}

	}()

	switch cfg := cfgRole.(type) {

	case *config.RelayConfig:

		startRelay(cfg)

	case *config.ClientConfig:

		startClient(cfg)

	default:

		logger.Error(
			"unsupported config type",
		)
	}
}

// startClient 启动 Client。
func startClient(
	cfg *config.ClientConfig,
) {

	logger.Info(
		"starting client server",
	)

	server := client.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {

		startError <- server.Start()

	}()

	select {

	case <-server.Started():

		logger.Info(
			"client server started",
		)

		tray.SetServerRunning(true)

		tray.Start(
			"client",
			func() {

				logger.Info(
					"stopping NetMap",
				)

				server.Stop()

				logger.Info(
					"NetMap stopped",
				)
			},
		)

	case err := <-startError:

		logger.Error(
			"start client server failed: %v",
			err,
		)

		tray.Start(
			"client",
			func() {

				logger.Info(
					"stopping NetMap",
				)
			},
		)
	}
}

// startRelay 启动 Relay。
func startRelay(
	cfg *config.RelayConfig,
) {

	logger.Info(
		"starting relay server",
	)

	server := relay.NewServer(cfg)

	startError := make(chan error, 1)

	go func() {

		startError <- server.Start()

	}()

	select {

	case <-server.Started():

		logger.Info(
			"relay server started",
		)

		tray.SetServerRunning(true)

		tray.Start(
			"relay",
			func() {

				logger.Info(
					"stopping NetMap",
				)

				server.Stop()

				logger.Info(
					"NetMap stopped",
				)
			},
		)

	case err := <-startError:

		logger.Error(
			"start relay server failed: %v",
			err,
		)

		tray.Start(
			"relay",
			func() {

				logger.Info(
					"stopping NetMap",
				)
			},
		)
	}
}
