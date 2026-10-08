package client

import (
	"fmt"
	"net"
	"sync"

	"netmap/internal/config"
	"netmap/internal/logger"
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
// 7. 提供固定 TCP 端口 Mapping
type Server struct {
	config  *config.Config
	matcher *rule.Matcher

	listener net.Listener

	// mappingListeners 保存所有 Mapping 对应的 Listener。
	mappingListeners map[string]net.Listener

	started chan struct{}

	mu sync.Mutex
}

// NewServer 创建 Client Server。
func NewServer(config *config.Config) *Server {
	return &Server{
		config:           config,
		matcher:          rule.NewMatcher(config.Rules),
		mappingListeners: make(map[string]net.Listener),
		started:          make(chan struct{}),
	}
}

// Started 返回 Client 启动完成通知。
//
// 当主监听端口以及所有启用的 Mapping 监听端口
// 全部成功启动后，该 Channel 会收到通知。
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

	// 如果后续 Mapping 启动失败，
	// 主 Listener 也需要关闭。
	defer func() {
		s.stopMappings()

		s.mu.Lock()
		s.listener = nil
		s.mu.Unlock()

		_ = listener.Close()
	}()

	logger.Info(
		"client server listening addr=%s",
		addr,
	)

	// 启动固定 TCP Mapping。
	if err := s.startMappings(); err != nil {
		logger.Error(
			"start mappings failed error=%v",
			err,
		)

		return err
	}

	close(s.started)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			stopped := s.listener == nil
			s.mu.Unlock()

			if stopped {
				logger.Info("client server stopped")
				return nil
			}

			logger.Error(
				"accept client connection failed error=%v",
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

	if listener != nil {
		logger.Info("stopping client server")

		// 停止主 Proxy Listener。
		if err := listener.Close(); err != nil {
			logger.Error(
				"close client listener failed error=%v",
				err,
			)
		}
	}

	// 无论主 Listener 是否存在，都停止所有 Mapping Listener。
	s.stopMappings()
}
