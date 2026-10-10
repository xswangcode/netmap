package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	RoleClient = "client"
	RoleRelay  = "relay"
)

// RoleConfig 所有配置公共接口。
type RoleConfig interface {
	GetRole() string
	Validate() error
}

// ListenConfig 通用监听配置。
type ListenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// LogConfig 日志配置。
type LogConfig struct {
	Level string `json:"level"`
}

func validateLog(log LogConfig) error {
	if log.Level != "" &&
		log.Level != "INFO" &&
		log.Level != "DEBUG" {
		return fmt.Errorf(
			"invalid log level: %s, expected INFO or DEBUG",
			log.Level,
		)
	}

	return nil
}

// LoadRole 根据配置中的 role 加载对应配置。
func LoadRole(path string) (RoleConfig, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read config failed: %w",
			err,
		)
	}

	var header struct {
		Role string `json:"role"`
	}

	if err := json.Unmarshal(
		data,
		&header,
	); err != nil {

		return nil, fmt.Errorf(
			"parse config role failed: %w",
			err,
		)
	}

	switch header.Role {

	case RoleClient:

		return LoadClient(data)

	case RoleRelay:

		return LoadRelay(data)

	default:

		return nil, fmt.Errorf(
			"invalid role: %s",
			header.Role,
		)
	}
}

// ValidateListen 校验监听地址。
func ValidateListen(
	listen ListenConfig,
) error {

	if listen.Host == "" {

		return fmt.Errorf(
			"listen host is required",
		)
	}

	if listen.Port <= 0 ||
		listen.Port > 65535 {

		return fmt.Errorf(
			"invalid listen port: %d",
			listen.Port,
		)
	}

	return nil
}
