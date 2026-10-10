package config

import (
	"encoding/json"
	"fmt"

	"netmap/internal/rule"
)

// ClientConfig Client 配置。
//
// Client 负责：
// 1. 接收本地代理请求
// 2. 根据规则判断是否经过 Relay
// 3. 连接 Relay
// 4. TCP 转发
// 5. 提供固定端口 Mapping
type ClientConfig struct {
	Role string `json:"role"`

	// Client监听地址
	Listen ListenConfig `json:"listen"`

	// Relay连接配置
	Relay ClientRelayConfig `json:"relay"`

	// 代理规则
	Rules []RuleConfig `json:"rules"`

	// TCP端口映射
	Mappings []MappingConfig `json:"mappings"`

	// 日志配置
	Log LogConfig `json:"log"`
}

// ClientRelayConfig Client连接Relay配置。
type ClientRelayConfig struct {
	Host string `json:"host"`

	Port int `json:"port"`

	ConnectTimeoutSeconds int `json:"connectTimeoutSeconds"`
}

// RuleConfig 代理规则。
type RuleConfig struct {
	Target string `json:"target"`

	Proxy bool `json:"proxy"`
}

// MappingConfig TCP映射配置。
type MappingConfig struct {
	Name string `json:"name"`

	Enabled bool `json:"enabled"`

	Listen ListenConfig `json:"listen"`

	Target MappingTargetConfig `json:"target"`
}

// MappingTargetConfig Mapping目标。
type MappingTargetConfig struct {
	Host string `json:"host"`

	Port int `json:"port"`
}

// GetRole 实现 RoleConfig。
func (c *ClientConfig) GetRole() string {

	return c.Role
}

// LoadClient 加载Client配置。
func LoadClient(
	data []byte,
) (*ClientConfig, error) {

	var cfg ClientConfig

	if err := json.Unmarshal(
		data,
		&cfg,
	); err != nil {

		return nil, fmt.Errorf(
			"parse client config failed: %w",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {

		return nil, err
	}

	return &cfg, nil
}

// Validate 校验Client配置。
func (c *ClientConfig) Validate() error {

	if c.Role != RoleClient {

		return fmt.Errorf(
			"invalid client role: %s",
			c.Role,
		)
	}

	if err := ValidateListen(
		c.Listen,
	); err != nil {

		return err
	}

	if c.Log.Level == "" {
		c.Log.Level = "INFO"
	}
	if err := validateLog(c.Log); err != nil {
		return err
	}

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

	rules := make([]rule.Rule, len(c.Rules))
	for i, configuredRule := range c.Rules {
		rules[i] = rule.Rule{
			Target: configuredRule.Target,
			Proxy:  configuredRule.Proxy,
		}
	}
	if err := rule.ValidateRules(rules); err != nil {
		return err
	}

	return validateMappings(c.Listen, c.Mappings)
}

func validateMappings(
	mainListen ListenConfig,
	mappings []MappingConfig,
) error {
	usedNames := make(map[string]struct{})
	usedListenAddresses := make(map[string]struct{})
	usedListenAddresses[fmt.Sprintf(
		"%s:%d",
		mainListen.Host,
		mainListen.Port,
	)] = struct{}{}

	for _, mapping := range mappings {
		if mapping.Name == "" {
			return fmt.Errorf("mapping name is required")
		}

		if _, exists := usedNames[mapping.Name]; exists {
			return fmt.Errorf("duplicate mapping name: %s", mapping.Name)
		}
		usedNames[mapping.Name] = struct{}{}

		if mapping.Listen.Host == "" {
			return fmt.Errorf(
				"mapping listen host is required: %s",
				mapping.Name,
			)
		}

		if mapping.Listen.Port <= 0 ||
			mapping.Listen.Port > 65535 {
			return fmt.Errorf(
				"invalid mapping listen port: %s:%d",
				mapping.Name,
				mapping.Listen.Port,
			)
		}

		listenAddress := fmt.Sprintf(
			"%s:%d",
			mapping.Listen.Host,
			mapping.Listen.Port,
		)
		if _, exists := usedListenAddresses[listenAddress]; exists {
			return fmt.Errorf(
				"duplicate mapping listen address: %s",
				listenAddress,
			)
		}
		usedListenAddresses[listenAddress] = struct{}{}

		if mapping.Target.Host == "" {
			return fmt.Errorf(
				"mapping target host is required: %s",
				mapping.Name,
			)
		}

		if mapping.Target.Port <= 0 ||
			mapping.Target.Port > 65535 {
			return fmt.Errorf(
				"invalid mapping target port: %s:%d",
				mapping.Name,
				mapping.Target.Port,
			)
		}
	}

	return nil
}
