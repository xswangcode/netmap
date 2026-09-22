package relay

import (
	"fmt"
	"log"
	"net"
	"sync"

	"netmap/internal/config"
	"netmap/internal/target"
)

// Server NetMap Relay 服务端。
//
// Relay 运行在跳板机电脑，负责：
//
// 1. 接收 Client 的连接
// 2. 解析 NetMap 内部连接请求
// 3. 校验目标地址白名单
// 4. 连接目标服务
// 5. 进行 TCP 双向数据转发
type Server struct {
	config    *config.Config
	whitelist *target.Whitelist

	listener net.Listener

	started chan struct{}

	mu sync.Mutex
}

// NewServer 创建 Relay Server。
func NewServer(config *config.Config) *Server {
	whitelist, err := target.NewWhitelist(
		config.Target.AllowedTargets,
	)
	if err != nil {
		panic(err)
	}

	return &Server{
		config:    config,
		whitelist: whitelist,
		started:   make(chan struct{}),
	}
}

// Started 返回 Relay 启动完成通知。
//
// 当监听端口成功后，该 Channel 会收到通知。
func (s *Server) Started() <-chan struct{} {
	return s.started
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

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.listener = nil
		s.mu.Unlock()

		_ = listener.Close()
	}()

	log.Printf(
		"relay server listening on %s",
		addr,
	)

	close(s.started)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			stopped := s.listener == nil
			s.mu.Unlock()

			if stopped {
				log.Printf("relay server stopped")
				return nil
			}

			log.Printf(
				"accept relay connection failed: %v",
				err,
			)
			continue
		}

		go s.handleConnection(conn)
	}
}

// Stop 停止 Relay Server。
func (s *Server) Stop() {
	s.mu.Lock()
	listener := s.listener
	s.listener = nil
	s.mu.Unlock()

	if listener == nil {
		return
	}

	log.Printf("stopping relay server")

	if err := listener.Close(); err != nil {
		log.Printf(
			"close relay listener failed: %v",
			err,
		)
	}
}

// isTargetAllowed 判断目标地址是否允许访问。
func (s *Server) isTargetAllowed(
	host string,
	port uint16,
) bool {
	return s.whitelist.Allow(host, port)
}
