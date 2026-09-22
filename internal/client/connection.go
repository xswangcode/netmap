package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"netmap/internal/forward"
	"netmap/internal/logger"
	"netmap/internal/protocol"
)

// handleConnection 处理一个本地应用连接。
//
// Client 支持：
// 1. HTTP Proxy
// 2. HTTPS CONNECT
// 3. SOCKS5 TCP CONNECT
//
// 协议判断：
//
// 第一个字节为 0x05
//
//	→ SOCKS5
//
// 其他情况
//
//	→ HTTP
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	logger.Info(
		"client connection established remote=%s local=%s",
		conn.RemoteAddr(),
		conn.LocalAddr(),
	)

	reader := bufio.NewReader(conn)

	// 查看第一个字节，但不消费数据。
	firstByte, err := reader.Peek(1)
	if err != nil {
		logger.Error(
			"read protocol header failed remote=%s error=%v",
			conn.RemoteAddr(),
			err,
		)
		return
	}

	// SOCKS5 版本号为 0x05。
	if firstByte[0] == 0x05 {
		logger.Debug(
			"protocol detected type=socks5 remote=%s",
			conn.RemoteAddr(),
		)

		s.handleSOCKS5Connection(
			reader,
			conn,
		)
		return
	}

	logger.Debug(
		"protocol detected type=http remote=%s",
		conn.RemoteAddr(),
	)

	// 其他情况按照 HTTP Proxy 处理。
	s.handleHTTPConnection(
		reader,
		conn,
	)
}

// handleHTTPConnection 处理 HTTP Proxy。
func (s *Server) handleHTTPConnection(
	reader *bufio.Reader,
	clientConn net.Conn,
) {
	// 读取 HTTP Proxy 请求。
	request, err := protocol.ReadHTTPProxyRequest(reader)
	if err != nil {
		logger.Error(
			"read http proxy request failed remote=%s error=%v",
			clientConn.RemoteAddr(),
			err,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadRequest,
		)

		return
	}

	target := net.JoinHostPort(
		request.Host,
		fmt.Sprintf("%d", request.Port),
	)

	// 判断是否通过 Relay。
	shouldProxy := s.shouldProxy(
		request.Host,
		request.Port,
	)

	logger.Info(
		"http proxy request target=%s method=%s proxy=%t",
		target,
		request.Request.Method,
		shouldProxy,
	)

	// 命中规则，通过 Relay。
	if shouldProxy {
		s.handleHTTPViaRelay(
			clientConn,
			request,
			target,
		)
		return
	}

	// 未命中规则，本机直接访问。
	s.handleHTTPDirect(
		clientConn,
		request,
		target,
	)
}

// handleHTTPViaRelay 通过 Relay 访问目标。
func (s *Server) handleHTTPViaRelay(
	clientConn net.Conn,
	request *protocol.HTTPRequest,
	target string,
) {
	// 连接 Relay。
	relayConn, err := s.connectRelay()
	if err != nil {
		logger.Error(
			"connect relay failed target=%s error=%v",
			target,
			err,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadGateway,
		)

		return
	}
	defer relayConn.Close()

	logger.Debug(
		"relay connected relay=%s target=%s",
		relayConn.RemoteAddr(),
		target,
	)

	// 请求 Relay 连接目标。
	if err := protocol.WriteConnectRequest(
		relayConn,
		&protocol.ConnectRequest{
			Host: request.Host,
			Port: request.Port,
		},
	); err != nil {
		logger.Error(
			"send connect request failed target=%s error=%v",
			target,
			err,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadGateway,
		)

		return
	}

	// 等待 Relay 返回连接结果。
	response, err := protocol.ReadConnectResponse(
		relayConn,
	)
	if err != nil {
		logger.Error(
			"read connect response failed target=%s error=%v",
			target,
			err,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadGateway,
		)

		return
	}

	if response.Status != protocol.ConnectSuccess {
		logger.Error(
			"relay connect target failed target=%s status=%d",
			target,
			response.Status,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadGateway,
		)

		return
	}

	logger.Info(
		"relay target connected target=%s",
		target,
	)

	// HTTPS CONNECT。
	if request.Request.Method == http.MethodConnect {
		handleHTTPSConnect(
			clientConn,
			relayConn,
			target,
		)

		return
	}

	// 普通 HTTP。
	handleHTTPRequest(
		clientConn,
		relayConn,
		request.Request,
		target,
	)
}

// handleHTTPDirect 本机直接访问目标。
func (s *Server) handleHTTPDirect(
	clientConn net.Conn,
	request *protocol.HTTPRequest,
	target string,
) {
	// 本机直接连接目标。
	targetConn, err := s.connectTarget(target)
	if err != nil {
		logger.Error(
			"connect target directly failed target=%s error=%v",
			target,
			err,
		)

		writeHTTPError(
			clientConn,
			http.StatusBadGateway,
		)

		return
	}
	defer targetConn.Close()

	logger.Info(
		"target connected directly target=%s",
		target,
	)

	// HTTPS CONNECT。
	if request.Request.Method == http.MethodConnect {
		handleHTTPSConnect(
			clientConn,
			targetConn,
			target,
		)

		return
	}

	// 普通 HTTP。
	handleHTTPRequest(
		clientConn,
		targetConn,
		request.Request,
		target,
	)
}

// handleSOCKS5Connection 处理 SOCKS5 TCP CONNECT。
func (s *Server) handleSOCKS5Connection(
	reader *bufio.Reader,
	clientConn net.Conn,
) {
	// 读取 SOCKS5 CONNECT 请求。
	//
	// reader 负责读取，
	// clientConn 负责返回 SOCKS5 响应。
	request, err := protocol.ReadSOCKS5Request(
		reader,
		clientConn,
	)
	if err != nil {
		logger.Error(
			"read socks5 request failed remote=%s error=%v",
			clientConn.RemoteAddr(),
			err,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}

	target := net.JoinHostPort(
		request.Host,
		fmt.Sprintf("%d", request.Port),
	)

	// 判断是否通过 Relay。
	shouldProxy := s.shouldProxy(
		request.Host,
		request.Port,
	)

	logger.Info(
		"socks5 connect request target=%s proxy=%t",
		target,
		shouldProxy,
	)

	// 命中规则，通过 Relay。
	if shouldProxy {
		s.handleSOCKS5ViaRelay(
			clientConn,
			request,
			target,
		)
		return
	}

	// 未命中规则，本机直接访问。
	s.handleSOCKS5Direct(
		clientConn,
		target,
	)
}

// handleSOCKS5ViaRelay 通过 Relay 访问目标。
func (s *Server) handleSOCKS5ViaRelay(
	clientConn net.Conn,
	request *protocol.SOCKS5Request,
	target string,
) {
	// 连接 Relay。
	relayConn, err := s.connectRelay()
	if err != nil {
		logger.Error(
			"connect relay failed target=%s error=%v",
			target,
			err,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}
	defer relayConn.Close()

	logger.Debug(
		"relay connected relay=%s target=%s",
		relayConn.RemoteAddr(),
		target,
	)

	// 请求 Relay 连接目标。
	if err := protocol.WriteConnectRequest(
		relayConn,
		&protocol.ConnectRequest{
			Host: request.Host,
			Port: request.Port,
		},
	); err != nil {
		logger.Error(
			"send connect request failed target=%s error=%v",
			target,
			err,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}

	// 等待 Relay 返回连接结果。
	response, err := protocol.ReadConnectResponse(
		relayConn,
	)
	if err != nil {
		logger.Error(
			"read connect response failed target=%s error=%v",
			target,
			err,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}

	if response.Status != protocol.ConnectSuccess {
		logger.Error(
			"relay connect target failed target=%s status=%d",
			target,
			response.Status,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}

	logger.Info(
		"relay target connected target=%s",
		target,
	)

	// 告诉 SOCKS5 客户端：
	// 目标连接成功。
	if err := protocol.WriteSOCKS5SuccessResponse(
		clientConn,
	); err != nil {
		logger.Error(
			"write socks5 success response failed target=%s error=%v",
			target,
			err,
		)

		return
	}

	logger.Info(
		"socks5 connection established target=%s",
		target,
	)

	// SOCKS5 CONNECT 成功以后，
	// 后面的数据就是普通 TCP 数据。
	//
	// SOCKS5 Client
	//      ⇅
	// NetMap Client
	//      ⇅
	// NetMap Relay
	//      ⇅
	// Target
	forward.TCP(
		clientConn,
		relayConn,
	)

	logger.Info(
		"socks5 connection closed target=%s",
		target,
	)
}

// handleSOCKS5Direct 本机直接访问目标。
func (s *Server) handleSOCKS5Direct(
	clientConn net.Conn,
	target string,
) {
	// 本机直接连接目标。
	targetConn, err := s.connectTarget(target)
	if err != nil {
		logger.Error(
			"connect target directly failed target=%s error=%v",
			target,
			err,
		)

		_ = protocol.WriteSOCKS5FailureResponse(
			clientConn,
		)

		return
	}
	defer targetConn.Close()

	logger.Info(
		"target connected directly target=%s",
		target,
	)

	// 告诉 SOCKS5 客户端：
	// 目标连接成功。
	if err := protocol.WriteSOCKS5SuccessResponse(
		clientConn,
	); err != nil {
		logger.Error(
			"write socks5 success response failed target=%s error=%v",
			target,
			err,
		)

		return
	}

	logger.Info(
		"socks5 direct connection established target=%s",
		target,
	)

	// 后续直接进行 TCP 双向转发。
	//
	// SOCKS5 Client
	//      ⇅
	// NetMap Client
	//      ⇅
	// Target
	forward.TCP(
		clientConn,
		targetConn,
	)

	logger.Info(
		"socks5 direct connection closed target=%s",
		target,
	)
}

// shouldProxy 判断目标是否通过 Relay。
func (s *Server) shouldProxy(
	host string,
	port uint16,
) bool {
	return s.matcher.ShouldProxy(
		host,
		port,
	)
}

// connectRelay 连接跳板机电脑上的 Relay。
func (s *Server) connectRelay() (net.Conn, error) {
	relayAddr := net.JoinHostPort(
		s.config.Relay.Host,
		fmt.Sprintf("%d", s.config.Relay.Port),
	)

	timeout := time.Duration(
		s.config.Relay.ConnectTimeoutSeconds,
	) * time.Second

	conn, err := net.DialTimeout(
		"tcp",
		relayAddr,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"connect relay %s failed: %w",
			relayAddr,
			err,
		)
	}

	return conn, nil
}

// connectTarget 本机直接连接目标。
func (s *Server) connectTarget(
	target string,
) (net.Conn, error) {
	timeout := time.Duration(
		s.config.Relay.ConnectTimeoutSeconds,
	) * time.Second

	conn, err := net.DialTimeout(
		"tcp",
		target,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"connect target %s failed: %w",
			target,
			err,
		)
	}

	return conn, nil
}

// handleHTTPRequest 处理普通 HTTP Proxy 请求。
func handleHTTPRequest(
	clientConn net.Conn,
	targetConn net.Conn,
	request *http.Request,
	target string,
) {
	// HTTP Proxy 请求使用 absolute-form。
	//
	// 例如：
	//
	// GET http://10.82.51.231:8080/test HTTP/1.1
	//
	// 发送给真正目标服务器时，
	// 应该变成：
	//
	// GET /test HTTP/1.1
	request.RequestURI = request.URL.RequestURI()
	request.URL.Scheme = ""
	request.URL.Host = ""

	// 第一版暂时关闭 Keep-Alive。
	request.Close = true
	request.Header.Set(
		"Connection",
		"close",
	)

	if err := request.Write(targetConn); err != nil {
		logger.Error(
			"write http request to target failed target=%s error=%v",
			target,
			err,
		)

		return
	}

	logger.Debug(
		"http request forwarded target=%s",
		target,
	)

	// 目标响应返回给本地应用。
	_, err := io.Copy(
		clientConn,
		targetConn,
	)

	if err != nil && err != io.EOF {
		logger.Error(
			"forward http response failed target=%s error=%v",
			target,
			err,
		)

		return
	}

	logger.Info(
		"http request completed target=%s",
		target,
	)
}

// handleHTTPSConnect 处理 HTTPS CONNECT。
func handleHTTPSConnect(
	clientConn net.Conn,
	targetConn net.Conn,
	target string,
) {
	// 告诉本地应用：
	// HTTPS Tunnel 已建立。
	_, err := io.WriteString(
		clientConn,
		"HTTP/1.1 200 Connection Established\r\n"+
			"\r\n",
	)
	if err != nil {
		logger.Error(
			"write connect success response failed target=%s error=%v",
			target,
			err,
		)

		return
	}

	logger.Info(
		"https connect established target=%s",
		target,
	)

	// CONNECT 成功以后，
	// 后面的数据直接进行 TCP 双向转发。
	forward.TCP(
		clientConn,
		targetConn,
	)

	logger.Info(
		"https connection closed target=%s",
		target,
	)
}

// writeHTTPError 向本地应用返回 HTTP 错误。
func writeHTTPError(
	conn net.Conn,
	statusCode int,
) {
	statusText := http.StatusText(statusCode)

	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Connection: close\r\n"+
			"Content-Length: 0\r\n"+
			"\r\n",
		statusCode,
		statusText,
	)

	_, _ = io.WriteString(
		conn,
		response,
	)
}
