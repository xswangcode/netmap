package tray

import (
	"os"
	"os/exec"

	"github.com/getlantern/systray"

	"netmap/internal/app"
	"netmap/internal/logger"
)

var (
	serverRole   string
	serverStatus *systray.MenuItem
)

// Start 启动 Windows 系统托盘。
//
// role 表示当前 NetMap 运行角色：client / relay。
// stopFunc 用于通知业务服务停止。
//
// 日志级别约定：
//   - logger.Info ：托盘启停、菜单点击等用户交互事件
//   - logger.Debug：服务状态变化等生命周期细节
//   - logger.Error：图标读取失败、打开文件/目录失败等
func Start(role string, stopFunc func()) {
	serverRole = role

	logger.Info(
		"tray: starting system tray: role=%s",
		roleName(),
	)

	systray.Run(
		func() {
			onReady(stopFunc)
		},
		onExit,
	)
}

// SetServerRunning 更新服务运行状态。
func SetServerRunning(running bool) {
	if serverStatus == nil {
		logger.Debug(
			"tray: set server running ignored, tray not ready",
		)

		return
	}

	name := roleName()

	if running {
		logger.Debug(
			"tray: server status changed: role=%s status=running",
			name,
		)

		serverStatus.SetTitle(
			"● " + name + " 运行中",
		)
		serverStatus.SetTooltip(
			name + " 正常运行",
		)
		return
	}

	logger.Debug(
		"tray: server status changed: role=%s status=stopped",
		name,
	)

	serverStatus.SetTitle(
		"● " + name + " 已停止",
	)
	serverStatus.SetTooltip(
		name + " 已停止",
	)
}

// SetServerFailed 更新服务启动失败状态。
func SetServerFailed() {
	if serverStatus == nil {
		logger.Debug(
			"tray: set server failed ignored, tray not ready",
		)

		return
	}

	name := roleName()

	logger.Error(
		"tray: server failed: role=%s",
		name,
	)

	serverStatus.SetTitle(
		"● " + name + " 启动失败",
	)
	serverStatus.SetTooltip(
		name + " 启动失败，请查看日志",
	)
}

// roleName 返回当前角色名称。
func roleName() string {
	if serverRole == "relay" {
		return "Relay"
	}

	return "Client"
}

// onReady 初始化托盘菜单。
func onReady(stopFunc func()) {
	logger.Debug(
		"tray: initializing tray menu: role=%s",
		roleName(),
	)

	setIcon()

	systray.SetTitle("NetMap")
	systray.SetTooltip("NetMap")

	serverStatus = systray.AddMenuItem(
		"● "+roleName()+" 启动中",
		roleName()+" 当前运行状态",
	)
	serverStatus.Disable()

	systray.AddSeparator()

	openConfig := systray.AddMenuItem(
		"打开配置",
		"打开当前配置文件",
	)

	openLog := systray.AddMenuItem(
		"查看日志",
		"打开 NetMap 日志目录",
	)

	systray.AddSeparator()

	exit := systray.AddMenuItem(
		"退出",
		"退出 NetMap",
	)

	logger.Info(
		"tray: ready: role=%s",
		roleName(),
	)

	go func() {
		for {
			select {
			case <-openConfig.ClickedCh:
				// 用户交互事件。
				logger.Info(
					"tray: menu clicked: action=open_config",
				)

				openConfigFile()

			case <-openLog.ClickedCh:
				logger.Info(
					"tray: menu clicked: action=open_log",
				)

				openLogDirectory()

			case <-exit.ClickedCh:
				logger.Info(
					"tray: menu clicked: action=exit",
				)

				if stopFunc != nil {
					stopFunc()
				}

				SetServerRunning(false)

				systray.Quit()
				return
			}
		}
	}()
}

// setIcon 设置 Windows 托盘图标。
func setIcon() {
	iconPath := app.AssetPath("netmap.ico")

	logger.Debug(
		"tray: loading icon: path=%s",
		iconPath,
	)

	data, err := os.ReadFile(iconPath)
	if err != nil {
		// 图标缺失不应阻止托盘运行，
		// 但需要记录，便于排查"托盘无图标"问题。
		logger.Error(
			"tray: read icon failed: path=%s error=%v",
			iconPath,
			err,
		)

		return
	}

	systray.SetIcon(data)
}

// onExit 托盘退出时执行。
func onExit() {
	logger.Info(
		"tray: exited: role=%s",
		roleName(),
	)
}

// openConfigFile 打开当前配置文件。
func openConfigFile() {
	fileName := "client.json"

	if serverRole == "relay" {
		fileName = "relay.json"
	}

	path := app.ConfigPath(fileName)

	logger.Debug(
		"tray: opening config file: path=%s",
		path,
	)

	if err := exec.Command(
		"cmd",
		"/c",
		"start",
		"",
		path,
	).Start(); err != nil {
		logger.Error(
			"tray: open config file failed: path=%s error=%v",
			path,
			err,
		)
	}
}

// openLogDirectory 打开 NetMap 日志目录。
func openLogDirectory() {
	path := app.LogDir()

	logger.Debug(
		"tray: opening log directory: path=%s",
		path,
	)

	if err := exec.Command(
		"explorer",
		path,
	).Start(); err != nil {
		logger.Error(
			"tray: open log directory failed: path=%s error=%v",
			path,
			err,
		)
	}
}
