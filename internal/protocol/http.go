package protocol

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
)

// HTTPRequest HTTP 代理请求。
type HTTPRequest struct {
	Request *http.Request
	Host    string
	Port    uint16
}

// ReadHTTPProxyRequest 从客户端连接中读取 HTTP 代理请求。
func ReadHTTPProxyRequest(
	reader *bufio.Reader,
) (*HTTPRequest, error) {
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"read http request failed: %w",
			err,
		)
	}

	if request.URL == nil {
		return nil, fmt.Errorf("http request url is empty")
	}

	host, port, err := parseHTTPTarget(request)
	if err != nil {
		return nil, err
	}

	return &HTTPRequest{
		Request: request,
		Host:    host,
		Port:    port,
	}, nil
}

// parseHTTPTarget 解析 HTTP 代理请求中的目标地址。
func parseHTTPTarget(request *http.Request) (
	string,
	uint16,
	error,
) {
	host := request.URL.Host

	// 某些 HTTP 请求可能通过 Host 字段提供目标地址。
	if host == "" {
		host = request.Host
	}

	if host == "" {
		return "", 0, fmt.Errorf(
			"http target host is empty",
		)
	}

	hostName, port := splitHostPort(host)

	if port == 0 {
		if request.URL.Scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	return hostName, port, nil
}

// splitHostPort 解析 HTTP Host。
func splitHostPort(host string) (string, uint16) {
	u, err := url.Parse("//" + host)
	if err != nil {
		return host, 0
	}

	if u.Hostname() == "" {
		return host, 0
	}

	port := uint16(0)

	if u.Port() != "" {
		var value int

		if _, err := fmt.Sscanf(
			u.Port(),
			"%d",
			&value,
		); err == nil && value > 0 && value <= 65535 {
			port = uint16(value)
		}
	}

	return u.Hostname(), port
}
