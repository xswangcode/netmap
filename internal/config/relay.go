package config

import (
	"encoding/json"
	"fmt"
)

// RelayConfig Relay 服务端配置。
//
// Relay 负责：
// 1. 接收 Client 请求
// 2. 校验目标白名单
// 3. 连接真实目标
// 4. TCP 转发
type RelayConfig struct {
	Role string `json:"role"`

	// Relay监听地址
	Listen ListenConfig `json:"listen"`

	// 目标连接配置
	Target TargetConfig `json:"target"`

	// 日志配置
	Log LogConfig `json:"log"`
}

// TargetConfig Relay目标配置。
type TargetConfig struct {
	ConnectTimeoutSeconds int `json:"connectTimeoutSeconds"`

	AllowedTargets []string `json:"allowedTargets"`
}

// GetRole 实现 RoleConfig。
func (r *RelayConfig) GetRole() string {

	return r.Role
}

// LoadRelay 加载Relay配置。
func LoadRelay(
	data []byte,
) (*RelayConfig, error) {

	var cfg RelayConfig

	if err := json.Unmarshal(
		data,
		&cfg,
	); err != nil {

		return nil, fmt.Errorf(
			"parse relay config failed: %w",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {

		return nil, err
	}

	return &cfg, nil
}

// Validate 校验Relay配置。
func (r *RelayConfig) Validate() error {

	if r.Role != RoleRelay {

		return fmt.Errorf(
			"invalid relay role: %s",
			r.Role,
		)
	}

	if err := ValidateListen(
		r.Listen,
	); err != nil {

		return err
	}

	if r.Log.Level == "" {
		r.Log.Level = "INFO"
	}
	if err := validateLog(r.Log); err != nil {
		return err
	}

	if r.Target.ConnectTimeoutSeconds <= 0 {

		return fmt.Errorf(
			"target connect timeout must be greater than 0",
		)
	}

	if len(r.Target.AllowedTargets) == 0 {

		return fmt.Errorf(
			"allowedTargets cannot be empty",
		)
	}

	return nil
}
