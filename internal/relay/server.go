package relay

import (
	"fmt"
	"log"
	"net"

	"netmap/internal/config"
	"netmap/internal/target"
)

// Server NetMap Relay 服务端。
//
// Relay 运行在跳板机电脑，负责：
// 1. 接收客户端电脑的连接
// 2. 根据 NetMap 协议获取目标地址
// 3. 检查目标地址白名单
// 4. 连接目标服务
// 5. 在客户端电脑和目标服务之间进行 TCP 双向转发
type Server struct {
	config    *config.Config
	whitelist *target.Whitelist
}

// NewServer 创建 Relay Server。
func NewServer(config *config.Config) *Server {
	whitelist, err := target.NewWhitelist(
		config.Target.AllowedTargets,
	)
	if err != nil {
		log.Fatalf("create target whitelist failed: %v", err)
	}

	return &Server{
		config:    config,
		whitelist: whitelist,
	}
}

// Start 启动 Relay Server。
func (s *Server) Start() error {
	addr := fmt.Sprintf(
		"%s:%d",
		s.config.Listen.Host,
		s.config.Listen.Port,
	)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf(
			"listen relay server failed: %w",
			err,
		)
	}
	defer listener.Close()

	log.Printf(
		"relay server listening on %s",
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

// isTargetAllowed 判断目标 IP + Port 是否在白名单中。
func (s *Server) isTargetAllowed(
	host string,
	port uint16,
) bool {
	return s.whitelist.Allow(
		host,
		port,
	)
}
