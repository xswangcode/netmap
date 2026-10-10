package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"netmap/internal/config"
)

const (
	RoleClient = config.RoleClient
	RoleRelay  = config.RoleRelay
)

type Handler struct {
	store *ConfigStore
}

func NewHandler(
	store *ConfigStore,
) *Handler {
	return &Handler{
		store: store,
	}
}

func (h *Handler) Register(
	mux *http.ServeMux,
) {

	mux.HandleFunc(
		"/api/status",
		h.handleStatus,
	)

	mux.HandleFunc(
		"/api/config",
		h.handleConfig,
	)

	mux.HandleFunc(
		"/api/config/reload",
		h.handleReload,
	)
}

// 获取运行状态。
func (h *Handler) handleStatus(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	if !checkLocalRequest(w, r) {
		return
	}

	role, path := h.store.RunningInfo()

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"runningRole":  role,
			"runningPath":  path,
			"adminAddress": "127.0.0.1:18081",
		},
	)
}

// 配置读取和保存。
func (h *Handler) handleConfig(
	w http.ResponseWriter,
	r *http.Request,
) {

	if !checkLocalRequest(w, r) {
		return
	}

	role := strings.TrimSpace(
		r.URL.Query().Get("role"),
	)

	if !validRole(role) {

		writeError(
			w,
			http.StatusBadRequest,
			"role must be client or relay",
		)

		return
	}

	switch r.Method {

	case http.MethodGet:

		cfg, path, err := h.store.GetConfig(role)

		if err != nil {

			writeError(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)

			return
		}

		writeJSON(
			w,
			http.StatusOK,
			map[string]interface{}{
				"role":   role,
				"path":   path,
				"config": cfg,
			},
		)

	case http.MethodPut:

		r.Body = http.MaxBytesReader(
			w,
			r.Body,
			2<<20,
		)

		decoder := json.NewDecoder(
			r.Body,
		)

		decoder.DisallowUnknownFields()

		cfg, err := decodeConfig(role, decoder)
		if err != nil {
			writeError(
				w,
				http.StatusBadRequest,
				"invalid config JSON: "+err.Error(),
			)

			return
		}

		var extra interface{}

		if err := decoder.Decode(&extra); err != io.EOF {

			writeError(
				w,
				http.StatusBadRequest,
				"request body must contain one JSON object",
			)

			return
		}

		if err := h.store.SaveConfig(
			role,
			cfg,
		); err != nil {

			status := http.StatusBadRequest

			if strings.Contains(
				err.Error(),
				"changed externally",
			) {

				status = http.StatusConflict
			}

			writeError(
				w,
				status,
				err.Error(),
			)

			return
		}

		_, path, err := h.store.GetConfig(role)

		if err != nil {

			writeError(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)

			return
		}

		writeJSON(
			w,
			http.StatusOK,
			map[string]interface{}{
				"message": "config saved",
				"role":    role,
				"path":    path,
			},
		)

	default:

		methodNotAllowed(w)
	}
}

func decodeConfig(
	role string,
	decoder *json.Decoder,
) (config.RoleConfig, error) {
	var cfg config.RoleConfig

	switch role {
	case config.RoleClient:
		cfg = &config.ClientConfig{}
	case config.RoleRelay:
		cfg = &config.RelayConfig{}
	default:
		return nil, http.ErrNotSupported
	}

	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}
	if cfg.GetRole() != role {
		return nil, fmt.Errorf("role does not match request")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// 重新加载配置。
func (h *Handler) handleReload(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodPost {

		methodNotAllowed(w)

		return
	}

	if !checkLocalRequest(w, r) {

		return
	}

	role := strings.TrimSpace(
		r.URL.Query().Get("role"),
	)

	if !validRole(role) {

		writeError(
			w,
			http.StatusBadRequest,
			"role must be client or relay",
		)

		return
	}

	if err := h.store.ReloadConfig(role); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

		return
	}

	cfg, path, err := h.store.GetConfig(role)

	if err != nil {

		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)

		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"message": "config reloaded",
			"role":    role,
			"path":    path,
			"config":  cfg,
		},
	)
}

func checkLocalRequest(
	w http.ResponseWriter,
	r *http.Request,
) bool {

	host := r.Host

	if strings.Contains(host, ":") {

		var err error

		host, _, err = net.SplitHostPort(
			r.Host,
		)

		if err != nil {

			writeError(
				w,
				http.StatusBadRequest,
				"invalid host",
			)

			return false
		}
	}

	if !strings.EqualFold(
		host,
		"localhost",
	) {

		ip := net.ParseIP(host)

		if ip == nil ||
			!ip.IsLoopback() {

			writeError(
				w,
				http.StatusForbidden,
				"local access only",
			)

			return false
		}
	}

	if origin := r.Header.Get("Origin"); origin != "" {

		u, err := url.Parse(origin)

		if err != nil ||
			!strings.EqualFold(
				u.Host,
				r.Host,
			) {

			writeError(
				w,
				http.StatusForbidden,
				"cross-origin request rejected",
			)

			return false
		}
	}

	return true
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value interface{},
) {

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {

	writeJSON(
		w,
		status,
		map[string]string{
			"error": message,
		},
	)
}

func methodNotAllowed(
	w http.ResponseWriter,
) {

	w.Header().Set(
		"Allow",
		"GET, PUT, POST",
	)

	writeError(
		w,
		http.StatusMethodNotAllowed,
		"method not allowed",
	)
}
func validRole(role string) bool {
	return role == RoleClient ||
		role == RoleRelay
}
