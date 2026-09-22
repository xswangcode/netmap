package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir 返回 NetMap.exe 所在目录。
func Dir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}

	return filepath.Dir(exePath)
}

// ConfigPath 返回配置文件路径。
func ConfigPath(fileName string) string {
	return filepath.Join(
		Dir(),
		"configs",
		fileName,
	)
}

// LogDir 返回日志目录。
func LogDir() string {
	return filepath.Join(
		Dir(),
		"logs",
	)
}

// LogPath 返回日志文件路径。
func LogPath(fileName string) string {
	return filepath.Join(
		LogDir(),
		fileName,
	)
}

// AssetPath 返回资源文件路径。
func AssetPath(fileName string) string {
	return filepath.Join(
		Dir(),
		"assets",
		fileName,
	)
}

// EnsureDir 确保目录存在。
func EnsureDir(path string) error {
	if path == "" {
		return fmt.Errorf("directory path is empty")
	}

	return os.MkdirAll(path, 0755)
}
