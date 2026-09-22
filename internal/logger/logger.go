package logger

import (
	"fmt"
	"log"
	"os"
	"sync"

	"netmap/internal/app"
)

// Level 日志级别。
//
// INFO：程序启动、组件初始化、核心运行状态。
// DEBUG：连接、断开、发送、接收等调试信息。
// ERROR：错误信息，不受日志级别限制。
type Level int

const (
	// INFO 重要运行日志。
	INFO Level = iota

	// DEBUG 调试日志。
	DEBUG
)

const (
	// 单个日志文件最大大小：5 MB。
	maxFileSize int64 = 5 * 1024 * 1024

	// 最大保留日志文件数量。
	maxBackupFiles = 7
)

var (
	logFile *os.File

	// 默认使用 INFO。
	// 实际等级由配置文件通过 SetLevel 修改。
	level = INFO
)

// rotatingWriter 日志轮转 Writer。
type rotatingWriter struct {
	mu sync.Mutex

	filePath string
	file     *os.File
	maxSize  int64
	maxFiles int
}

// Write 写入日志。
//
// 当当前日志文件达到最大大小时，先进行轮转，再写入新日志。
func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return 0, fmt.Errorf("log file is not initialized")
	}

	// 获取当前文件大小。
	info, err := w.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat log file failed: %w", err)
	}

	// 当前文件 + 本次写入内容超过限制，则进行轮转。
	if info.Size()+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	return w.file.Write(p)
}

// rotate 执行日志文件轮转。
//
// 当前：
//
//	netmap.log
//
// 轮转后：
//
//	netmap.log     <- 新文件
//	netmap.log.1
//	netmap.log.2
//	...
//	netmap.log.7
func (w *rotatingWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log file failed: %w", err)
		}

		w.file = nil
	}

	// 删除最旧的日志文件。
	oldest := fmt.Sprintf("%s.%d", w.filePath, w.maxFiles)
	_ = os.Remove(oldest)

	// 从后往前移动日志文件。
	for i := w.maxFiles - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", w.filePath, i)
		newPath := fmt.Sprintf("%s.%d", w.filePath, i+1)

		if _, err := os.Stat(oldPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return fmt.Errorf(
				"stat rotated log file failed: %w",
				err,
			)
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf(
				"rename rotated log file failed: %w",
				err,
			)
		}
	}

	// 当前日志文件改名为 .1。
	backupPath := fmt.Sprintf("%s.1", w.filePath)

	if _, err := os.Stat(w.filePath); err == nil {
		if err := os.Rename(w.filePath, backupPath); err != nil {
			return fmt.Errorf(
				"rename current log file failed: %w",
				err,
			)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf(
			"stat current log file failed: %w",
			err,
		)
	}

	// 创建新的当前日志文件。
	file, err := os.OpenFile(
		w.filePath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf(
			"create new log file failed: %w",
			err,
		)
	}

	w.file = file
	logFile = file

	return nil
}

// Init 初始化 NetMap 日志。
//
// 日志固定保存到：
//
// netmap.exe
// logs/
//
//	netmap.log
//	netmap.log.1
//	netmap.log.2
//	...
//	netmap.log.7
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

	writer := &rotatingWriter{
		filePath: logPath,
		file:     file,
		maxSize:  maxFileSize,
		maxFiles: maxBackupFiles,
	}

	log.SetFlags(
		log.Ldate |
			log.Ltime |
			log.Lmicroseconds,
	)

	log.SetOutput(writer)

	Info(
		"logger initialized path=%s level=%s maxFileSize=%dMB maxBackupFiles=%d",
		logPath,
		LevelName(level),
		maxFileSize/(1024*1024),
		maxBackupFiles,
	)

	return nil
}

// SetLevel 设置日志级别。
//
// 建议由配置文件统一控制，不在业务代码中修改。
func SetLevel(newLevel Level) {
	level = newLevel
}

// Debug 输出 DEBUG 日志。
//
// 用于：
//   - TCP/UDP 连接
//   - Socket 连接建立/断开
//   - 数据发送/接收
//   - 请求转发
//   - IP 映射过程
//   - 高频运行细节
func Debug(format string, args ...interface{}) {
	if level < DEBUG {
		return
	}

	log.Printf("[DEBUG] "+format, args...)
}

// Info 输出 INFO 日志。
//
// 用于：
//   - 程序启动
//   - 组件初始化
//   - 配置加载
//   - 服务启动/停止
//   - 核心功能启用
func Info(format string, args ...interface{}) {
	if level < INFO {
		return
	}

	log.Printf("[INFO] "+format, args...)
}

// Error 输出 ERROR 日志。
//
// ERROR 不受当前日志等级影响，始终输出。
func Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// LevelName 返回日志级别名称。
func LevelName(l Level) string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// Close 关闭日志文件。
func Close() {
	if logFile == nil {
		return
	}

	_ = logFile.Close()
	logFile = nil
}
