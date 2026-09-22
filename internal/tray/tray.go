package tray

import (
	"os"
	"os/exec"

	"github.com/getlantern/systray"

	"netmap/internal/app"
)

var (
	serverRole   string
	serverStatus *systray.MenuItem
)

// Start 启动 Windows 系统托盘。
//
// role 表示当前 NetMap 运行角色：client / relay。
// stopFunc 用于通知业务服务停止。
func Start(role string, stopFunc func()) {
	serverRole = role

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
		return
	}

	name := roleName()

	if running {
		serverStatus.SetTitle(
			"● " + name + " 运行中",
		)
		serverStatus.SetTooltip(
			name + " 正常运行",
		)
		return
	}

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
		return
	}

	name := roleName()

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

	go func() {
		for {
			select {
			case <-openConfig.ClickedCh:
				openConfigFile()

			case <-openLog.ClickedCh:
				openLogDirectory()

			case <-exit.ClickedCh:
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

	data, err := os.ReadFile(iconPath)
	if err != nil {
		return
	}

	systray.SetIcon(data)
}

// onExit 托盘退出时执行。
func onExit() {
}

// openConfigFile 打开当前配置文件。
func openConfigFile() {
	fileName := "client.json"

	if serverRole == "relay" {
		fileName = "relay.json"
	}

	path := app.ConfigPath(fileName)

	_ = exec.Command(
		"cmd",
		"/c",
		"start",
		"",
		path,
	).Start()
}

// openLogDirectory 打开 NetMap 日志目录。
func openLogDirectory() {
	path := app.LogDir()

	_ = exec.Command(
		"explorer",
		path,
	).Start()
}
