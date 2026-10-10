package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"netmap/internal/config"
	"os"
	"path/filepath"
	"sync"
)

// configEntry 保存配置缓存。
type configEntry struct {
	Config config.RoleConfig

	Path string

	Hash string
}

// ConfigStore 配置管理。
type ConfigStore struct {
	mu sync.Mutex

	entries map[string]*configEntry

	runningRole string

	runningPath string

	paths map[string]string
}

// NewConfigStore 创建配置管理。
func NewConfigStore(
	startupPath string,
	startupConfig config.RoleConfig,
) (*ConfigStore, error) {

	if startupConfig == nil {

		return nil, fmt.Errorf(
			"startup config is nil",
		)
	}

	absPath, err := filepath.Abs(
		startupPath,
	)

	if err != nil {

		return nil, err
	}

	paths := map[string]string{

		config.RoleClient: filepath.Join(
			filepath.Dir(absPath),
			"client.json",
		),

		config.RoleRelay: filepath.Join(
			filepath.Dir(absPath),
			"relay.json",
		),
	}

	paths[startupConfig.GetRole()] = absPath

	store := &ConfigStore{

		entries: make(map[string]*configEntry),

		runningRole: startupConfig.GetRole(),

		runningPath: absPath,

		paths: paths,
	}

	entry, err := loadConfigEntry(absPath)

	if err != nil {

		return nil, err
	}

	store.entries[startupConfig.GetRole()] = entry

	return store, nil
}

// RunningInfo 获取当前运行信息。
func (s *ConfigStore) RunningInfo() (
	string,
	string,
) {

	s.mu.Lock()

	defer s.mu.Unlock()

	return s.runningRole,
		s.runningPath
}

// GetConfig 获取配置。
func (s *ConfigStore) GetConfig(
	role string,
) (
	config.RoleConfig,
	string,
	error,
) {

	s.mu.Lock()

	defer s.mu.Unlock()

	entry, exists := s.entries[role]

	if !exists {

		var err error

		entry, err =
			loadConfigEntry(
				s.paths[role],
			)

		if err != nil {

			return nil, "", err
		}

		s.entries[role] = entry
	}

	return entry.Config,
		entry.Path,
		nil
}

// SaveConfig 保存配置。
func (s *ConfigStore) SaveConfig(
	role string,
	cfg config.RoleConfig,
) error {

	if cfg == nil {

		return fmt.Errorf(
			"config is nil",
		)
	}

	if cfg.GetRole() != role {

		return fmt.Errorf(
			"role mismatch",
		)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	s.mu.Lock()

	defer s.mu.Unlock()

	entry, exists := s.entries[role]

	if !exists {

		var err error
		entry, err =
			loadConfigEntry(
				s.paths[role],
			)

		if err != nil {

			return err
		}

		s.entries[role] = entry
	}

	old, err := os.ReadFile(
		entry.Path,
	)

	if err != nil {

		return err
	}

	if hashBytes(old) != entry.Hash {

		return fmt.Errorf(
			"config file changed externally",
		)
	}

	output, err := json.MarshalIndent(
		cfg,
		"",
		"  ",
	)

	if err != nil {

		return err
	}

	output = append(output, '\n')

	if err := writeConfigFile(
		entry.Path,
		output,
	); err != nil {

		return err
	}

	entry.Config = cfg

	entry.Hash = hashBytes(output)

	return nil
}

// ReloadConfig重新加载。
func (s *ConfigStore) ReloadConfig(
	role string,
) error {

	s.mu.Lock()

	defer s.mu.Unlock()

	entry, err := loadConfigEntry(
		s.paths[role],
	)

	if err != nil {

		return err
	}

	s.entries[role] = entry

	return nil
}

// ConfigPath获取路径。
func (s *ConfigStore) ConfigPath(
	role string,
) (string, error) {

	s.mu.Lock()

	defer s.mu.Unlock()

	return s.paths[role], nil
}

// loadConfigEntry加载配置。
func loadConfigEntry(
	path string,
) (*configEntry, error) {

	data, err := os.ReadFile(path)

	if err != nil {

		return nil, err
	}

	cfg, err := config.LoadRole(
		path,
	)

	if err != nil {

		return nil, err
	}

	return &configEntry{

		Config: cfg,

		Path: path,

		Hash: hashBytes(data),
	}, nil
}

func hashBytes(
	data []byte,
) string {

	sum := sha256.Sum256(data)

	return hex.EncodeToString(
		sum[:],
	)
}

// writeConfigFile 简单安全写入。
func writeConfigFile(
	path string,
	data []byte,
) error {

	return os.WriteFile(
		path,
		data,
		0644,
	)
}
