package config

import (
	"encoding/json"
	"fmt"
	"os"

	"netmap/internal/rule"
)

// Config NetMap 配置。
type Config struct {
	Role   string       `json:"role"`
	Listen ListenConfig `json:"listen"`
	Relay  RelayConfig  `json:"relay"`
	Target TargetConfig `json:"target"`
	Rules  []rule.Rule  `json:"rules"`
	Log    LogConfig    `json:"log"`
}

// LogConfig 日志配置。
type LogConfig struct {
	Level string `json:"level"`
}

// ListenConfig 服务监听配置。
type ListenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// RelayConfig Relay 连接配置。
type RelayConfig struct {
	Host                  string `json:"host"`
	Port                  int    `json:"port"`
	ConnectTimeoutSeconds int    `json:"connectTimeoutSeconds"`
}

// TargetConfig 目标服务配置。
type TargetConfig struct {
	ConnectTimeoutSeconds int      `json:"connectTimeoutSeconds"`
	AllowedTargets        []string `json:"allowedTargets"`
}

// Load 从 JSON 配置文件加载 NetMap 配置。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read config file failed: %w",
			err,
		)
	}

	var config Config

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"parse config file failed: %w",
			err,
		)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 校验配置。
func (c *Config) Validate() error {
	if c.Role != "client" && c.Role != "relay" {
		return fmt.Errorf(
			"invalid role: %s",
			c.Role,
		)
	}

	if c.Listen.Host == "" {
		return fmt.Errorf(
			"listen host is required",
		)
	}

	if c.Listen.Port <= 0 ||
		c.Listen.Port > 65535 {
		return fmt.Errorf(
			"invalid listen port: %d",
			c.Listen.Port,
		)
	}

	// 未配置日志等级时使用 INFO。
	if c.Log.Level == "" {
		c.Log.Level = "INFO"
	}

	if c.Log.Level != "INFO" &&
		c.Log.Level != "DEBUG" {
		return fmt.Errorf(
			"invalid log level: %s, expected INFO or DEBUG",
			c.Log.Level,
		)
	}

	if c.Role == "client" {
		if c.Relay.Host == "" {
			return fmt.Errorf(
				"relay host is required",
			)
		}

		if c.Relay.Port <= 0 ||
			c.Relay.Port > 65535 {
			return fmt.Errorf(
				"invalid relay port: %d",
				c.Relay.Port,
			)
		}

		if c.Relay.ConnectTimeoutSeconds <= 0 {
			return fmt.Errorf(
				"relay connect timeout must be greater than 0",
			)
		}

		if err := rule.ValidateRules(c.Rules); err != nil {
			return err
		}
	}

	if c.Role == "relay" {
		if c.Target.ConnectTimeoutSeconds <= 0 {
			return fmt.Errorf(
				"target connect timeout must be greater than 0",
			)
		}

		if len(c.Target.AllowedTargets) == 0 {
			return fmt.Errorf(
				"allowedTargets cannot be empty",
			)
		}
	}

	return nil
}
