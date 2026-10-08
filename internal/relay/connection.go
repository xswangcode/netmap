package relay

import (
	"fmt"
	"net"
	"time"

	"netmap/internal/forward"
	"netmap/internal/logger"
	"netmap/internal/protocol"
)

// handleConnection 处理一个客户端电脑的连接。
//
// Relay 不解析 HTTP、HTTPS、SOCKS5，
// 只负责解析 Client 和 Relay 之间的 NetMap 协议。
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	local := conn.LocalAddr().String()

	logger.Debug(
		"relay: client connected remote=%s local=%s",
		remote,
		local,
	)

	request, err := protocol.ReadConnectRequest(conn)
	if err != nil {
		logger.Error(
			"relay: read connect request failed remote=%s error=%v",
			remote,
			err,
		)

		return
	}

	target := net.JoinHostPort(
		request.Host,
		fmt.Sprintf("%d", request.Port),
	)

	logger.Debug(
		"relay: connect request remote=%s target=%s",
		remote,
		target,
	)

	if !s.isTargetAllowed(
		request.Host,
		request.Port,
	) {
		logger.Error(
			"relay: target is not allowed remote=%s target=%s",
			remote,
			target,
		)

		_ = protocol.WriteConnectResponse(
			conn,
			&protocol.ConnectResponse{
				Status: protocol.ConnectFailed,
			},
		)

		return
	}

	timeout := time.Duration(
		s.config.Target.ConnectTimeoutSeconds,
	) * time.Second

	logger.Debug(
		"relay: dialing target remote=%s target=%s timeout=%s",
		remote,
		target,
		timeout,
	)

	targetConn, err := net.DialTimeout(
		"tcp",
		target,
		timeout,
	)
	if err != nil {
		logger.Error(
			"relay: connect target failed remote=%s target=%s error=%v",
			remote,
			target,
			err,
		)

		_ = protocol.WriteConnectResponse(
			conn,
			&protocol.ConnectResponse{
				Status: protocol.ConnectFailed,
			},
		)

		return
	}

	defer targetConn.Close()

	logger.Debug(
		"relay: target tcp connected remote=%s target=%s",
		remote,
		target,
	)

	if err := protocol.WriteConnectResponse(
		conn,
		&protocol.ConnectResponse{
			Status: protocol.ConnectSuccess,
		},
	); err != nil {
		logger.Error(
			"relay: send connect response failed remote=%s target=%s error=%v",
			remote,
			target,
			err,
		)

		return
	}

	logger.Info(
		"relay: connect success remote=%s target=%s",
		remote,
		target,
	)

	logger.Debug(
		"relay: start forwarding remote=%s target=%s",
		remote,
		target,
	)

	forward.TCP(
		conn,
		targetConn,
	)

	logger.Debug(
		"relay: connection closed remote=%s target=%s",
		remote,
		target,
	)
}
