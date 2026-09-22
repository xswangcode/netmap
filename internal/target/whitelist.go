package target

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Whitelist 目标地址白名单。
//
// 支持四种配置：
//
//  1. IP
//     例如：10.82.51.231
//     表示该 IP 的所有端口都允许。
//
//  2. IP:Port
//     例如：10.82.51.232:8080
//     表示只允许指定 IP 的指定端口。
//
//  3. Domain
//     例如：bastion.voyah.cn
//     表示该域名的所有端口都允许。
//
//  4. Domain:Port
//     例如：bastion.voyah.cn:443
//     表示只允许指定域名的指定端口。
//
// 如果同一个 Host 同时配置了 Host 和 Host:Port，
// 则 Host 规则优先，表示允许该 Host 的所有端口。
type Whitelist struct {
	// allowedHosts 保存允许全部端口的 Host。
	//
	// 例如：
	//
	// 10.82.51.231
	// bastion.voyah.cn
	allowedHosts map[string]struct{}

	// allowedTargets 保存允许指定端口的目标地址。
	//
	// 例如：
	//
	// 10.82.51.232:8080
	// bastion.voyah.cn:443
	allowedTargets map[string]struct{}
}

// NewWhitelist 创建目标白名单。
func NewWhitelist(targets []string) (*Whitelist, error) {
	allowedHosts := make(map[string]struct{})
	allowedTargets := make(map[string]struct{})

	for _, value := range targets {
		targetValue := strings.TrimSpace(value)

		if targetValue == "" {
			continue
		}

		// 纯 IP。
		if ip := net.ParseIP(targetValue); ip != nil {
			allowedHosts[normalizeIP(ip)] = struct{}{}
			continue
		}

		// 尝试解析 Host:Port。
		targetHost, targetPort, splitErr := net.SplitHostPort(
			targetValue,
		)

		if splitErr == nil {
			targetHost = strings.TrimSpace(targetHost)

			if targetHost == "" {
				return nil, fmt.Errorf(
					"target host cannot be empty: %q",
					targetValue,
				)
			}

			// 校验端口。
			portNumber, err := strconv.Atoi(targetPort)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid target port %q: %w",
					targetPort,
					err,
				)
			}

			if portNumber <= 0 || portNumber > 65535 {
				return nil, fmt.Errorf(
					"target port out of range: %d",
					portNumber,
				)
			}

			// Host 是 IP。
			if ip := net.ParseIP(targetHost); ip != nil {
				targetHost = normalizeIP(ip)
			} else {
				// Host 是域名。
				targetHost = normalizeDomain(targetHost)
			}

			normalizedTarget := net.JoinHostPort(
				targetHost,
				strconv.Itoa(portNumber),
			)

			allowedTargets[normalizedTarget] = struct{}{}

			continue
		}

		// 如果包含冒号，但是又不是合法的 Host:Port，
		// 则认为配置格式错误。
		//
		// 普通域名本身不会包含冒号。
		if strings.Contains(targetValue, ":") {
			return nil, fmt.Errorf(
				"invalid target %q, expected IP, IP:Port, domain or domain:Port",
				targetValue,
			)
		}

		// 普通域名。
		//
		// 例如：
		//
		// bastion.voyah.cn
		// example.com
		//
		// 这里不进行 DNS 解析。
		// Relay 真正连接目标时，
		// net.DialTimeout 会负责解析域名。
		domain := normalizeDomain(targetValue)

		if domain == "" {
			return nil, fmt.Errorf(
				"target host cannot be empty: %q",
				targetValue,
			)
		}

		allowedHosts[domain] = struct{}{}
	}

	if len(allowedHosts) == 0 &&
		len(allowedTargets) == 0 {
		return nil, fmt.Errorf(
			"target whitelist cannot be empty",
		)
	}

	return &Whitelist{
		allowedHosts:   allowedHosts,
		allowedTargets: allowedTargets,
	}, nil
}

// Allow 判断目标 Host + Port 是否允许访问。
//
// 匹配规则：
//
//  1. Host 在 allowedHosts 中
//     → 允许该 Host 所有端口。
//
//  2. Host + Port 在 allowedTargets 中
//     → 允许指定端口。
//
//  3. 都不匹配
//     → 不允许。
func (w *Whitelist) Allow(
	host string,
	port uint16,
) bool {
	host = strings.TrimSpace(host)

	if host == "" {
		return false
	}

	// IP。
	if ip := net.ParseIP(host); ip != nil {
		host = normalizeIP(ip)
	} else {
		// Domain。
		host = normalizeDomain(host)
	}

	// Host 全端口规则优先。
	if _, exists := w.allowedHosts[host]; exists {
		return true
	}

	target := net.JoinHostPort(
		host,
		strconv.Itoa(int(port)),
	)

	_, exists := w.allowedTargets[target]

	return exists
}

// normalizeIP 统一 IP 字符串格式。
func normalizeIP(ip net.IP) string {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String()
	}

	return ip.To16().String()
}

// normalizeDomain 统一域名格式。
//
// 域名不区分大小写，因此统一转换为小写。
// 同时去掉最后的 "."。
//
// 例如：
//
// Bastion.VOYAH.CN.
//
// 转换为：
//
// bastion.voyah.cn
func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimSuffix(domain, ".")

	return strings.ToLower(domain)
}
