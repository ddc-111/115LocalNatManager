package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"115localnatmanager/model"
)

type ConfigUpdateRequest = model.ConfigUpdateRequest

type Manager struct {
	mu       sync.RWMutex
	config   model.Config
	token    model.TokenData
	dataDir  string
}

func NewManager(dataDir string) *Manager {
	m := &Manager{
		dataDir: dataDir,
		config: model.Config{
			Port:            "11580",
			DownloadDir:     getDefaultDownloadDir(),
			MonitorEnabled:  true,
			MonitorInterval: 30,
		},
	}
	os.MkdirAll(dataDir, 0755)
	m.loadConfig()
	m.loadToken()
	return m
}

func getDefaultDownloadDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "115cloud")
}

func (m *Manager) loadConfig() {
	path := filepath.Join(m.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &m.config)
}

func (m *Manager) loadToken() {
	path := filepath.Join(m.dataDir, "token.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &m.token)
}

func (m *Manager) SaveConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := filepath.Join(m.dataDir, "config.json")
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) SaveToken() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := filepath.Join(m.dataDir, "token.json")
	data, err := json.MarshalIndent(m.token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) GetConfig() model.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) UpdateConfig(req model.ConfigUpdateRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 始终更新这些字段
	m.config.DownloadDir = req.DownloadDir
	m.config.DefaultSavePath = req.DefaultSavePath
	m.config.DefaultSaveName = req.DefaultSaveName
	
	// 只在有值时更新
	if req.Port != "" {
		m.config.Port = req.Port
	}
	if req.MonitorEnabled != nil {
		m.config.MonitorEnabled = *req.MonitorEnabled
	}
	if req.MonitorInterval != nil {
		m.config.MonitorInterval = *req.MonitorInterval
	}
}

func (m *Manager) GetToken() model.TokenData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.token
}

func (m *Manager) SetRefreshToken(rt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token.RefreshToken = rt
}

func (m *Manager) SetAccessToken(at string, expires int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token.AccessToken = at
	m.token.ExpiresAt = timeNow().Add(timeDuration(expires))
}

func timeNow() time.Time {
	return time.Now()
}

func timeDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
