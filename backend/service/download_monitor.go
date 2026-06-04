package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"115localnatmanager/api"
	"115localnatmanager/config"
)

type DownloadMonitor struct {
	client     *api.Client
	config     *config.Manager
	stopCh     chan struct{}
	mu         sync.Mutex
	running    bool
	downloaded map[string]bool
}

func NewDownloadMonitor(client *api.Client, cfg *config.Manager) *DownloadMonitor {
	return &DownloadMonitor{
		client:     client,
		config:     cfg,
		stopCh:     make(chan struct{}),
		downloaded: make(map[string]bool),
	}
}

func (dm *DownloadMonitor) Start() {
	dm.mu.Lock()
	if dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = true
	dm.mu.Unlock()

	go dm.monitorLoop()
	log.Println("Download monitor started")
}

func (dm *DownloadMonitor) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if !dm.running {
		return
	}
	close(dm.stopCh)
	dm.running = false
	log.Println("Download monitor stopped")
}

func (dm *DownloadMonitor) IsRunning() bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.running
}

func (dm *DownloadMonitor) monitorLoop() {
	cfg := dm.config.GetConfig()
	interval := time.Duration(cfg.MonitorInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.checkTasks()
		}
	}
}

func (dm *DownloadMonitor) checkTasks() {
	result, err := dm.client.GetDownloadTaskList(1)
	if err != nil {
		log.Printf("Failed to get task list: %v", err)
		return
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return
	}

	tasks, ok := data["tasks"].([]interface{})
	if !ok {
		return
	}

	for _, task := range tasks {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}

		status, _ := taskMap["status"].(float64)
		infoHash, _ := taskMap["info_hash"].(string)

		if status == 2 {
			dm.mu.Lock()
			if dm.downloaded[infoHash] {
				dm.mu.Unlock()
				continue
			}
			dm.downloaded[infoHash] = true
			dm.mu.Unlock()

			go dm.downloadCompletedFile(taskMap)
		}
	}
}

func (dm *DownloadMonitor) downloadCompletedFile(task map[string]interface{}) {
	name, _ := task["name"].(string)
	fileID, _ := task["file_id"].(string)

	if fileID == "" {
		log.Printf("No file_id for task: %s", name)
		return
	}

	fileInfo, err := dm.client.GetFileInfo(fileID)
	if err != nil {
		log.Printf("Failed to get file info for %s: %v", name, err)
		return
	}

	data, ok := fileInfo["data"].(map[string]interface{})
	if !ok {
		return
	}

	pickCode, _ := data["pick_code"].(string)
	if pickCode == "" {
		log.Printf("No pick_code for file: %s", name)
		return
	}

	downloadInfo, err := dm.client.GetDownloadURL(pickCode)
	if err != nil {
		log.Printf("Failed to get download URL for %s: %v", name, err)
		return
	}

	dlData, ok := downloadInfo["data"].(map[string]interface{})
	if !ok {
		return
	}

	for _, v := range dlData {
		fileData, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		urlObj, ok := fileData["url"].(map[string]interface{})
		if !ok {
			continue
		}

		downloadURL, _ := urlObj["url"].(string)
		if downloadURL != "" {
			go dm.downloadFile(downloadURL, name)
			return
		}
	}
}

func (dm *DownloadMonitor) downloadFile(url, filename string) {
	cfg := dm.config.GetConfig()
	destDir := cfg.DownloadDir
	os.MkdirAll(destDir, 0755)

	destPath := filepath.Join(destDir, filename)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to download %s: %v", filename, err)
		return
	}
	defer resp.Body.Close()

	file, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", destPath, err)
		return
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Failed to write file %s: %v", filename, err)
		return
	}

	log.Printf("Downloaded %s (%d bytes) to %s", filename, written, destPath)
}

func (dm *DownloadMonitor) GetStatus() map[string]interface{} {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	cfg := dm.config.GetConfig()
	return map[string]interface{}{
		"running":          dm.running,
		"download_dir":     cfg.DownloadDir,
		"monitor_interval": cfg.MonitorInterval,
		"downloaded_count": len(dm.downloaded),
	}
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func jsonMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
