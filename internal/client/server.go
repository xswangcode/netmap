package client

import (
	"fmt"
	"log"
	"net"
	"sync"

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

	listener net.Listener

	started chan struct{}

	mu sync.Mutex
}

// NewServer 创建 Client Server。
func NewServer(config *config.Config) *Server {
	return &Server{
		config:  config,
		matcher: rule.NewMatcher(config.Rules),
		started: make(chan struct{}),
	}
}

// Started 返回 Client 启动完成通知。
//
// 当监听端口成功后，该 Channel 会收到通知。
func (s *Server) Started() <-chan struct{} {
	return s.started
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
		"client server listening on %s",
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
				log.Printf("client server stopped")
				return nil
			}

			log.Printf(
				"accept connection failed: %v",
				err,
			)
			continue
		}

		go s.handleConnection(conn)
	}
}

// Stop 停止 Client Server。
func (s *Server) Stop() {
	s.mu.Lock()
	listener := s.listener
	s.listener = nil
	s.mu.Unlock()

	if listener == nil {
		return
	}

	log.Printf("stopping client server")

	if err := listener.Close(); err != nil {
		log.Printf(
			"close client listener failed: %v",
			err,
		)
	}
}
