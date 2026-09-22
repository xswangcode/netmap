package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// TestConfig 测试配置。
type TestConfig struct {
	Proxy        ProxyConfig `json:"proxy"`
	HTTPTarget   string      `json:"httpTarget"`
	HTTPSTarget  string      `json:"httpsTarget"`
	SOCKS5Target string      `json:"socks5Target"`
}

// ProxyConfig 本地代理配置。
type ProxyConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func main() {
	configPath := "configs/test.json"

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	proxyAddr := net.JoinHostPort(
		config.Proxy.Host,
		fmt.Sprintf("%d", config.Proxy.Port),
	)

	log.Printf("test proxy=%s", proxyAddr)

	// HTTP 测试。
	if config.HTTPTarget != "" {
		testHTTP(
			proxyAddr,
			config.HTTPTarget,
		)
	}

	// HTTPS CONNECT 测试。
	if config.HTTPSTarget != "" {
		testHTTPS(
			proxyAddr,
			config.HTTPSTarget,
		)
	}

	// SOCKS5 测试。
	if config.SOCKS5Target != "" {
		testSOCKS5(
			proxyAddr,
			config.SOCKS5Target,
		)
	}
}

// loadConfig 加载测试配置。
func loadConfig(path string) (*TestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read test config failed: %w",
			err,
		)
	}

	var config TestConfig

	if err := json.Unmarshal(
		data,
		&config,
	); err != nil {
		return nil, fmt.Errorf(
			"parse test config failed: %w",
			err,
		)
	}

	return &config, nil
}

// testHTTP 测试 HTTP Proxy。
func testHTTP(
	proxyAddr string,
	targetURL string,
) {
	log.Printf(
		"http test start target=%s",
		targetURL,
	)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: func(
				_ *http.Request,
			) (*url.URL, error) {
				return url.Parse(
					"http://" + proxyAddr,
				)
			},
		},
		Timeout: 15 * time.Second,
	}

	response, err := client.Get(targetURL)
	if err != nil {
		log.Printf(
			"http test failed target=%s error=%v",
			targetURL,
			err,
		)
		return
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf(
			"http read response failed target=%s error=%v",
			targetURL,
			err,
		)
		return
	}

	log.Printf(
		"http test success status=%s bodyLength=%d",
		response.Status,
		len(body),
	)
}

// testHTTPS 测试 HTTPS CONNECT。
func testHTTPS(
	proxyAddr string,
	targetURL string,
) {
	log.Printf(
		"https test start target=%s",
		targetURL,
	)

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Printf(
			"parse https target failed target=%s error=%v",
			targetURL,
			err,
		)
		return
	}

	proxyConn, err := net.DialTimeout(
		"tcp",
		proxyAddr,
		10*time.Second,
	)
	if err != nil {
		log.Printf(
			"connect proxy failed proxy=%s error=%v",
			proxyAddr,
			err,
		)
		return
	}
	defer proxyConn.Close()

	// 向 HTTP Proxy 发送 CONNECT。
	connectRequest := fmt.Sprintf(
		"CONNECT %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"\r\n",
		target.Host,
		target.Host,
	)

	if _, err := io.WriteString(
		proxyConn,
		connectRequest,
	); err != nil {
		log.Printf(
			"write https connect request failed target=%s error=%v",
			targetURL,
			err,
		)
		return
	}

	reader := bufio.NewReader(proxyConn)

	response, err := http.ReadResponse(
		reader,
		&http.Request{
			Method: "CONNECT",
		},
	)
	if err != nil {
		log.Printf(
			"read https connect response failed target=%s error=%v",
			targetURL,
			err,
		)
		return
	}

	if response.StatusCode != http.StatusOK {
		log.Printf(
			"https connect failed target=%s status=%s",
			targetURL,
			response.Status,
		)
		return
	}

	log.Printf(
		"https connect success target=%s",
		targetURL,
	)

	// CONNECT 成功以后，
	// 这里就已经进入 TCP Tunnel。
	//
	// 简单发送一段 HTTPS ClientHello 不容易手写，
	// 所以这里只验证 CONNECT 是否成功。
}

// testSOCKS5 测试 SOCKS5 TCP CONNECT。
func testSOCKS5(
	proxyAddr string,
	targetAddr string,
) {
	log.Printf(
		"socks5 test start proxy=%s target=%s",
		proxyAddr,
		targetAddr,
	)

	// 连接本地 SOCKS5 代理。
	conn, err := net.DialTimeout(
		"tcp",
		proxyAddr,
		10*time.Second,
	)
	if err != nil {
		log.Printf(
			"socks5 connect proxy failed proxy=%s error=%v",
			proxyAddr,
			err,
		)
		return
	}
	defer conn.Close()

	// --------------------------------
	// 1. SOCKS5 握手
	// --------------------------------
	//
	// VER    = 0x05
	// NMETHODS = 1
	// METHOD = 0x00
	//
	// 表示：
	// SOCKS5 + 不需要认证。
	handshake := []byte{
		0x05,
		0x01,
		0x00,
	}

	if _, err := conn.Write(handshake); err != nil {
		log.Printf(
			"socks5 write handshake failed error=%v",
			err,
		)
		return
	}

	handshakeResponse := make([]byte, 2)

	if _, err := io.ReadFull(
		conn,
		handshakeResponse,
	); err != nil {
		log.Printf(
			"socks5 read handshake response failed error=%v",
			err,
		)
		return
	}

	if handshakeResponse[0] != 0x05 {
		log.Printf(
			"socks5 invalid response version=%d",
			handshakeResponse[0],
		)
		return
	}

	if handshakeResponse[1] != 0x00 {
		log.Printf(
			"socks5 server does not support no-auth method method=%d",
			handshakeResponse[1],
		)
		return
	}

	log.Printf(
		"socks5 handshake success",
	)

	// --------------------------------
	// 2. 解析目标地址
	// --------------------------------

	host, port, err := net.SplitHostPort(
		targetAddr,
	)
	if err != nil {
		log.Printf(
			"socks5 invalid target address target=%s error=%v",
			targetAddr,
			err,
		)
		return
	}

	portNumber, err := parsePort(port)
	if err != nil {
		log.Printf(
			"socks5 invalid target port target=%s error=%v",
			targetAddr,
			err,
		)
		return
	}

	// --------------------------------
	// 3. SOCKS5 CONNECT
	// --------------------------------
	//
	// 这里根据目标地址选择：
	//
	// IPv4：
	//
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	//
	// Domain：
	//
	// ATYP = 0x03
	// 后面是：
	//
	// +------+----------+
	// | LEN  | DOMAIN   |
	// +------+----------+

	ip := net.ParseIP(host)

	var request []byte

	if ipv4 := ip.To4(); ipv4 != nil {
		request = make([]byte, 10)

		request[0] = 0x05
		request[1] = 0x01
		request[2] = 0x00
		request[3] = 0x01

		copy(
			request[4:8],
			ipv4,
		)

		binary.BigEndian.PutUint16(
			request[8:10],
			uint16(portNumber),
		)
	} else if ip != nil {
		request = make([]byte, 22)

		request[0] = 0x05
		request[1] = 0x01
		request[2] = 0x00
		request[3] = 0x04

		copy(
			request[4:20],
			ip.To16(),
		)

		binary.BigEndian.PutUint16(
			request[20:22],
			uint16(portNumber),
		)
	} else {
		hostBytes := []byte(host)

		if len(hostBytes) == 0 ||
			len(hostBytes) > 255 {
			log.Printf(
				"socks5 invalid domain target=%s",
				targetAddr,
			)
			return
		}

		request = make([]byte, 7+len(hostBytes))

		request[0] = 0x05
		request[1] = 0x01
		request[2] = 0x00
		request[3] = 0x03
		request[4] = byte(len(hostBytes))

		copy(
			request[5:5+len(hostBytes)],
			hostBytes,
		)

		binary.BigEndian.PutUint16(
			request[5+len(hostBytes):7+len(hostBytes)],
			uint16(portNumber),
		)
	}

	if _, err := conn.Write(request); err != nil {
		log.Printf(
			"socks5 write connect request failed target=%s error=%v",
			targetAddr,
			err,
		)
		return
	}

	// --------------------------------
	// 4. 读取 SOCKS5 CONNECT 响应
	// --------------------------------

	responseHeader := make([]byte, 4)

	if _, err := io.ReadFull(
		conn,
		responseHeader,
	); err != nil {
		log.Printf(
			"socks5 read connect response failed target=%s error=%v",
			targetAddr,
			err,
		)
		return
	}

	if responseHeader[0] != 0x05 {
		log.Printf(
			"socks5 invalid connect response version=%d",
			responseHeader[0],
		)
		return
	}

	if responseHeader[1] != 0x00 {
		log.Printf(
			"socks5 connect target failed target=%s reply=%d",
			targetAddr,
			responseHeader[1],
		)
		return
	}

	// 根据 ATYP 读取 BND.ADDR。
	switch responseHeader[3] {
	case 0x01:
		// IPv4
		data := make([]byte, 4)

		if _, err := io.ReadFull(
			conn,
			data,
		); err != nil {
			log.Printf(
				"socks5 read ipv4 response address failed error=%v",
				err,
			)
			return
		}

	case 0x03:
		// Domain
		length := make([]byte, 1)

		if _, err := io.ReadFull(
			conn,
			length,
		); err != nil {
			log.Printf(
				"socks5 read domain response length failed error=%v",
				err,
			)
			return
		}

		data := make([]byte, int(length[0]))

		if _, err := io.ReadFull(
			conn,
			data,
		); err != nil {
			log.Printf(
				"socks5 read domain response address failed error=%v",
				err,
			)
			return
		}

	case 0x04:
		// IPv6
		data := make([]byte, 16)

		if _, err := io.ReadFull(
			conn,
			data,
		); err != nil {
			log.Printf(
				"socks5 read ipv6 response address failed error=%v",
				err,
			)
			return
		}

	default:
		log.Printf(
			"socks5 unsupported response address type=%d",
			responseHeader[3],
		)
		return
	}

	// BND.PORT
	portBytes := make([]byte, 2)

	if _, err := io.ReadFull(
		conn,
		portBytes,
	); err != nil {
		log.Printf(
			"socks5 read response port failed error=%v",
			err,
		)
		return
	}

	log.Printf(
		"socks5 connect success target=%s",
		targetAddr,
	)
}

// parsePort 解析端口。
func parsePort(value string) (int, error) {
	var port int

	if _, err := fmt.Sscanf(
		value,
		"%d",
		&port,
	); err != nil {
		return 0, err
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf(
			"port out of range: %d",
			port,
		)
	}

	return port, nil
}
