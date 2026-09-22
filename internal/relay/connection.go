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
//
// 日志级别约定：
//   - logger.Debug：连接级细节（地址、目标解析、转发开始/结束）
//   - logger.Info ：应用层面关键事件（客户端接入、成功连通目标、连接关闭）
//   - logger.Error：任何出错路径
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	local := conn.LocalAddr().String()

	// 应用层面：有客户端接入。
	logger.Info(
		"relay: client connected remote=%s local=%s",
		remote,
		local,
	)

	logger.Debug(
		"relay: reading connect request remote=%s",
		remote,
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

	// 应用层面：成功建立到目标后端的连接。
	logger.Debug(
		"relay: target connected remote=%s target=%s",
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

	// 应用层面：连接生命周期结束。
	logger.Debug(
		"relay: connection closed remote=%s target=%s",
		remote,
		target,
	)
}
