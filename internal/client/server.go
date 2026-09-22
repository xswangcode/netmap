package client

import (
	"fmt"
	"log"
	"net"

	"netmap/internal/config"
	"netmap/internal/rule"
)

// Server NetMap Client 服务端。
//
// Client 运行在客户端电脑，负责：
//
// 1. 接收本地应用的代理连接
// 2. 解析 HTTP / HTTPS / SOCKS5 请求
// 3. 根据本地规则判断是否通过 Relay
// 4. 命中规则时连接 Relay
// 5. 未命中规则时本机直接连接目标
// 6. 进行 TCP 双向数据转发
type Server struct {
	config  *config.Config
	matcher *rule.Matcher
}

// NewServer 创建 Client Server。
func NewServer(config *config.Config) *Server {
	return &Server{
		config:  config,
		matcher: rule.NewMatcher(config.Rules),
	}
}

// Start 启动 Client Server。
func (s *Server) Start() error {
	addr := fmt.Sprintf(
		"%s:%d",
		s.config.Listen.Host,
		s.config.Listen.Port,
	)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf(
			"listen client server failed: %w",
			err,
		)
	}
	defer listener.Close()

	log.Printf(
		"client server listening on %s",
		addr,
	)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf(
				"accept connection failed: %v",
				err,
			)
			continue
		}

		go s.handleConnection(conn)
	}
}
