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
	stopServer   func()
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
	stopServer = stopFunc

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

// Restart stops the active service and closes the tray event loop.
func Restart() {
	logger.Info("tray: restarting NetMap")
	if stopServer != nil {
		stopServer()
	}
	SetServerRunning(false)
	systray.Quit()
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

	openAdmin := systray.AddMenuItem(
		"打开管理页面",
		"打开本地 NetMap 管理页面",
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
			case <-openAdmin.ClickedCh:
				logger.Info(
					"tray: menu clicked: action=open_admin",
				)

				openAdminPage()

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

	logger.Info(
		"tray: loading icon: path=%s",
		iconPath,
	)

	data, err := os.ReadFile(iconPath)
	if err != nil {
		logger.Error(
			"tray: read icon failed: path=%s error=%v",
			iconPath,
			err,
		)

		return
	}

	logger.Info(
		"tray: icon loaded: path=%s size=%d",
		iconPath,
		len(data),
	)

	// ICO 文件至少需要包含基础文件头。
	if len(data) < 22 {
		logger.Error(
			"tray: icon file is invalid or too small: path=%s size=%d",
			iconPath,
			len(data),
		)

		return
	}

	// ICO 文件头：
	// 0-1：保留字段，必须为 0
	// 2-3：类型，ICO 为 1
	if data[0] != 0 ||
		data[1] != 0 ||
		data[2] != 1 ||
		data[3] != 0 {
		logger.Error(
			"tray: invalid ico header: path=%s",
			iconPath,
		)

		return
	}

	systray.SetIcon(data)

	logger.Info(
		"tray: icon set requested: path=%s",
		iconPath,
	)
}

// onExit 托盘退出时执行。
func onExit() {
	logger.Info(
		"tray: exited: role=%s",
		roleName(),
	)
}

// openAdminPage 在默认浏览器中打开本地管理页面。
func openAdminPage() {
	logger.Debug(
		"tray: opening admin page: url=%s",
		app.AdminURL,
	)

	if err := exec.Command(
		"cmd",
		"/c",
		"start",
		"",
		app.AdminURL,
	).Start(); err != nil {
		logger.Error(
			"tray: open admin page failed: url=%s error=%v",
			app.AdminURL,
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
