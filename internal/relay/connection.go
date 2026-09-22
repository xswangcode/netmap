package relay

import (
	"fmt"
	"log"
	"net"
	"time"

	"netmap/internal/forward"
	"netmap/internal/protocol"
)

// handleConnection 处理一个客户端电脑的连接。
//
// Relay 不解析 HTTP、HTTPS、SOCKS5，
// 只负责解析 Client 和 Relay 之间的 NetMap 协议。
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf(
		"client connected remote=%s local=%s",
		conn.RemoteAddr(),
		conn.LocalAddr(),
	)

	request, err := protocol.ReadConnectRequest(conn)
	if err != nil {
		log.Printf(
			"read connect request failed remote=%s error=%v",
			conn.RemoteAddr(),
			err,
		)
		return
	}

	target := net.JoinHostPort(
		request.Host,
		fmt.Sprintf("%d", request.Port),
	)

	log.Printf(
		"connect request remote=%s target=%s",
		conn.RemoteAddr(),
		target,
	)

	// 检查目标 IP + Port 是否允许。
	//
	// 白名单支持：
	//
	// 10.82.51.231
	//     → 允许该 IP 所有端口
	//
	// 10.82.51.232:8080
	//     → 只允许 8080
	if !s.isTargetAllowed(
		request.Host,
		request.Port,
	) {
		log.Printf(
			"target is not allowed remote=%s target=%s",
			conn.RemoteAddr(),
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

	targetConn, err := net.DialTimeout(
		"tcp",
		target,
		timeout,
	)
	if err != nil {
		log.Printf(
			"connect target failed remote=%s target=%s error=%v",
			conn.RemoteAddr(),
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

	if err := protocol.WriteConnectResponse(
		conn,
		&protocol.ConnectResponse{
			Status: protocol.ConnectSuccess,
		},
	); err != nil {
		log.Printf(
			"send connect response failed remote=%s target=%s error=%v",
			conn.RemoteAddr(),
			target,
			err,
		)
		return
	}

	log.Printf(
		"target connected remote=%s target=%s",
		conn.RemoteAddr(),
		target,
	)

	forward.TCP(
		conn,
		targetConn,
	)

	log.Printf(
		"connection closed remote=%s target=%s",
		conn.RemoteAddr(),
		target,
	)
}
