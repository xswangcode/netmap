package client

import (
	"fmt"
	"net"

	"netmap/internal/config"
	"netmap/internal/forward"
	"netmap/internal/logger"
	"netmap/internal/protocol"
)

// startMappings 启动所有启用的 TCP Mapping。
func (s *Server) startMappings() error {
	for _, mapping := range s.config.Mappings {
		if !mapping.Enabled {
			logger.Debug(
				"mapping disabled name=%s",
				mapping.Name,
			)

			continue
		}

		if err := s.startMapping(mapping); err != nil {
			return err
		}
	}

	return nil
}

// startMapping 启动一个 TCP Mapping。
func (s *Server) startMapping(
	mapping config.MappingConfig,
) error {
	addr := net.JoinHostPort(
		mapping.Listen.Host,
		fmt.Sprintf("%d", mapping.Listen.Port),
	)

	listener, err := net.Listen(
		"tcp",
		addr,
	)
	if err != nil {
		return fmt.Errorf(
			"listen mapping failed name=%s addr=%s: %w",
			mapping.Name,
			addr,
			err,
		)
	}

	s.mu.Lock()

	if s.mappingListeners == nil {
		s.mappingListeners = make(map[string]net.Listener)
	}

	s.mappingListeners[mapping.Name] = listener

	s.mu.Unlock()

	target := net.JoinHostPort(
		mapping.Target.Host,
		fmt.Sprintf("%d", mapping.Target.Port),
	)

	logger.Info(
		"mapping started name=%s listen=%s target=%s",
		mapping.Name,
		addr,
		target,
	)

	go s.acceptMappingConnections(
		mapping,
		listener,
	)

	return nil
}

// acceptMappingConnections 接收 Mapping 的本地 TCP 连接。
func (s *Server) acceptMappingConnections(
	mapping config.MappingConfig,
	listener net.Listener,
) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()

			currentListener := s.mappingListeners[mapping.Name]

			s.mu.Unlock()

			if currentListener == nil ||
				currentListener != listener {
				logger.Info(
					"mapping stopped name=%s",
					mapping.Name,
				)

				return
			}

			logger.Error(
				"accept mapping connection failed name=%s error=%v",
				mapping.Name,
				err,
			)

			continue
		}

		go s.handleMappingConnection(
			mapping,
			conn,
		)
	}
}

// handleMappingConnection 处理一个 Mapping TCP 连接。
func (s *Server) handleMappingConnection(
	mapping config.MappingConfig,
	clientConn net.Conn,
) {
	defer clientConn.Close()

	target := net.JoinHostPort(
		mapping.Target.Host,
		fmt.Sprintf("%d", mapping.Target.Port),
	)

	logger.Info(
		"mapping connection established name=%s remote=%s target=%s",
		mapping.Name,
		clientConn.RemoteAddr(),
		target,
	)

	relayConn, err := s.connectRelay()
	if err != nil {
		logger.Error(
			"mapping connect relay failed name=%s target=%s error=%v",
			mapping.Name,
			target,
			err,
		)

		return
	}

	defer relayConn.Close()

	logger.Debug(
		"mapping relay connected name=%s relay=%s target=%s",
		mapping.Name,
		relayConn.RemoteAddr(),
		target,
	)

	if err := protocol.WriteConnectRequest(
		relayConn,
		&protocol.ConnectRequest{
			Host: mapping.Target.Host,
			Port: uint16(mapping.Target.Port),
		},
	); err != nil {
		logger.Error(
			"mapping send connect request failed name=%s target=%s error=%v",
			mapping.Name,
			target,
			err,
		)

		return
	}

	response, err := protocol.ReadConnectResponse(
		relayConn,
	)
	if err != nil {
		logger.Error(
			"mapping read connect response failed name=%s target=%s error=%v",
			mapping.Name,
			target,
			err,
		)

		return
	}

	if response.Status != protocol.ConnectSuccess {
		logger.Error(
			"mapping target connect failed name=%s target=%s status=%d",
			mapping.Name,
			target,
			response.Status,
		)

		return
	}

	logger.Info(
		"mapping target connected name=%s target=%s",
		mapping.Name,
		target,
	)

	forward.TCP(
		clientConn,
		relayConn,
	)

	logger.Info(
		"mapping connection closed name=%s target=%s",
		mapping.Name,
		target,
	)
}

// stopMappings 停止所有 Mapping Listener。
func (s *Server) stopMappings() {
	s.mu.Lock()

	listeners := s.mappingListeners

	s.mappingListeners = nil

	s.mu.Unlock()

	for name, listener := range listeners {
		logger.Info(
			"stopping mapping name=%s",
			name,
		)

		if err := listener.Close(); err != nil {
			logger.Error(
				"close mapping listener failed name=%s error=%v",
				name,
				err,
			)
		}
	}
}
