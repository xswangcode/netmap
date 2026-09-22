package logger

import (
	"fmt"
	"log"
	"os"

	"netmap/internal/app"
)

var logFile *os.File

// Init 初始化 NetMap 日志。
//
// 日志固定保存到：
//
// netmap.exe
// logs/
//
//	netmap.log
func Init() error {
	if err := app.EnsureDir(app.LogDir()); err != nil {
		return fmt.Errorf(
			"create log directory failed: %w",
			err,
		)
	}

	logPath := app.LogPath("netmap.log")

	file, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf(
			"open log file failed: %w",
			err,
		)
	}

	logFile = file

	log.SetFlags(
		log.Ldate |
			log.Ltime |
			log.Lmicroseconds,
	)

	log.SetOutput(logFile)

	log.Printf(
		"logger initialized path=%s",
		logPath,
	)

	return nil
}

// Close 关闭日志文件。
func Close() {
	if logFile == nil {
		return
	}

	_ = logFile.Close()
	logFile = nil
}
