package admin

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"netmap/internal/logger"
)

//go:embed web/*
var webFiles embed.FS

// Server Web 管理服务。
type Server struct {
	httpServer *http.Server
}

// NewServer 创建 Web 管理服务。
func NewServer(store *ConfigStore) *Server {
	mux := http.NewServeMux()

	handler := NewHandler(store)
	handler.Register(mux)

	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}

	// API 路由优先匹配，其他路径由内嵌静态文件服务处理。
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	return &Server{
		httpServer: &http.Server{
			Addr:              "127.0.0.1:18081",
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Start 启动 Web 管理服务。
func (s *Server) Start() error {
	logger.Info("admin: starting web server at http://%s", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	if err != nil {
		return fmt.Errorf("start admin server failed: %w", err)
	}

	return nil
}

// Stop 停止 Web 管理服务。
func (s *Server) Stop() error {
	return s.httpServer.Close()
}
