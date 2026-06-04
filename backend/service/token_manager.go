package service

import (
	"log"
	"sync"
	"time"

	"115localnatmanager/api"
	"115localnatmanager/config"
)

type TokenManager struct {
	client   *api.Client
	config   *config.Manager
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

func NewTokenManager(client *api.Client, cfg *config.Manager) *TokenManager {
	return &TokenManager{
		client: client,
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

func (tm *TokenManager) Start() {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return
	}
	tm.running = true
	tm.mu.Unlock()

	go tm.refreshLoop()
	log.Println("Token manager started")
}

func (tm *TokenManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.running {
		return
	}
	close(tm.stopCh)
	tm.running = false
	log.Println("Token manager stopped")
}

func (tm *TokenManager) refreshLoop() {
	ticker := time.NewTicker(50 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stopCh:
			return
		case <-ticker.C:
			tm.refreshToken()
		}
	}
}

func (tm *TokenManager) refreshToken() {
	token := tm.config.GetToken()
	if token.RefreshToken == "" {
		return
	}

	_, err := tm.client.GetUserInfo()
	if err != nil {
		log.Printf("Token refresh check failed: %v", err)
	}
}
