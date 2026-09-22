package rule

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"netmap/internal/logger"
)

// Rule 代理规则。
//
// Target 支持：
// 1. IP
// 2. IP:Port
// 3. 域名
// 4. 域名:Port
//
// 例如：
//
// 10.82.51.231
// 10.82.51.232:8080
// example.com
// api.example.com:443
type Rule struct {
	Target string `json:"target"`
	Proxy  bool   `json:"proxy"`
}

// Matcher 代理规则匹配器。
type Matcher struct {
	rules []Rule
}

// NewMatcher 创建规则匹配器。
func NewMatcher(rules []Rule) *Matcher {
	logger.Debug(
		"rule: matcher created: rules=%d",
		len(rules),
	)

	return &Matcher{
		rules: rules,
	}
}

// ShouldProxy 判断目标是否需要通过 Relay。
//
// 返回 true：通过 Relay。
// 返回 false：本机直连。
func (m *Matcher) ShouldProxy(
	host string,
	port uint16,
) bool {
	for _, rule := range m.rules {
		if matchTarget(
			rule.Target,
			host,
			port,
		) {
			logger.Debug(
				"rule: matched: target=%s port=%d rule=%s proxy=%t",
				host,
				port,
				rule.Target,
				rule.Proxy,
			)

			return rule.Proxy
		}
	}

	// 没有匹配到规则时，本机直连。
	logger.Debug(
		"rule: no match, direct: target=%s port=%d",
		host,
		port,
	)

	return false
}

// matchTarget 判断规则目标是否匹配实际目标。
func matchTarget(
	ruleTarget string,
	host string,
	port uint16,
) bool {
	ruleTarget = strings.TrimSpace(ruleTarget)
	host = strings.TrimSpace(host)

	if ruleTarget == "" || host == "" {
		return false
	}

	// IP。
	if ip := net.ParseIP(ruleTarget); ip != nil {
		return matchHost(
			ip.String(),
			host,
		)
	}

	// IP:Port 或 Domain:Port。
	ruleHost, rulePort, err := net.SplitHostPort(
		ruleTarget,
	)
	if err == nil {
		portNumber, err := strconv.Atoi(rulePort)
		if err != nil ||
			portNumber <= 0 ||
			portNumber > 65535 {
			return false
		}

		if uint16(portNumber) != port {
			return false
		}

		return matchHost(
			ruleHost,
			host,
		)
	}

	// Domain。
	return matchHost(
		ruleTarget,
		host,
	)
}

// matchHost 比较两个 Host。
//
// IP 按 IP 地址比较。
// 域名忽略大小写。
func matchHost(
	ruleHost string,
	requestHost string,
) bool {
	ruleHost = strings.TrimSuffix(
		strings.TrimSpace(ruleHost),
		".",
	)

	requestHost = strings.TrimSuffix(
		strings.TrimSpace(requestHost),
		".",
	)

	ruleIP := net.ParseIP(ruleHost)
	requestIP := net.ParseIP(requestHost)

	// 两边都是 IP。
	if ruleIP != nil && requestIP != nil {
		return ruleIP.Equal(requestIP)
	}

	// 两边都是域名。
	return strings.EqualFold(
		ruleHost,
		requestHost,
	)
}

// ValidateRules 校验代理规则。
func ValidateRules(rules []Rule) error {
	logger.Debug(
		"rule: validating rules: count=%d",
		len(rules),
	)

	for _, rule := range rules {
		if strings.TrimSpace(rule.Target) == "" {
			logger.Error(
				"rule: validation failed: target is empty",
			)

			return fmt.Errorf(
				"rule target cannot be empty",
			)
		}

		// 校验规则格式是否支持。
		//
		// 支持：
		// IP
		// IP:Port
		// Domain
		// Domain:Port
		target := strings.TrimSpace(rule.Target)

		// 纯 IP。
		if ip := net.ParseIP(target); ip != nil {
			logger.Debug(
				"rule: validated ip rule: target=%s proxy=%t",
				target,
				rule.Proxy,
			)

			continue
		}

		// IP:Port 或 Domain:Port。
		if host, port, err := net.SplitHostPort(target); err == nil {
			if strings.TrimSpace(host) == "" {
				logger.Error(
					"rule: validation failed: host is empty: target=%s",
					target,
				)

				return fmt.Errorf(
					"rule target host cannot be empty: %s",
					target,
				)
			}

			portNumber, err := strconv.Atoi(port)
			if err != nil ||
				portNumber <= 0 ||
				portNumber > 65535 {
				logger.Error(
					"rule: validation failed: invalid port: target=%s port=%s",
					target,
					port,
				)

				return fmt.Errorf(
					"invalid rule target port: %s",
					target,
				)
			}

			logger.Debug(
				"rule: validated host:port rule: target=%s proxy=%t",
				target,
				rule.Proxy,
			)

			continue
		}

		// Domain。
		if strings.TrimSpace(target) != "" {
			logger.Debug(
				"rule: validated domain rule: target=%s proxy=%t",
				target,
				rule.Proxy,
			)

			continue
		}

		logger.Error(
			"rule: validation failed: invalid target: %s",
			target,
		)

		return fmt.Errorf(
			"invalid rule target: %s",
			target,
		)
	}

	logger.Debug(
		"rule: validation passed: count=%d",
		len(rules),
	)

	return nil
}
