package service

import (
	"encoding/json"
	"fmt"
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
	client         *api.Client
	config         *config.Manager
	stopCh         chan struct{}
	mu             sync.Mutex
	running        bool
	downloaded     map[string]bool
	localDownloads map[string]*LocalDownloadTask
}

type LocalDownloadTask struct {
	FileName    string  `json:"file_name"`
	URL         string  `json:"url"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	Size        int64   `json:"size"`
	Downloaded  int64   `json:"downloaded"`
	Error       string  `json:"error,omitempty"`
	StartTime   int64   `json:"start_time"`
}

func NewDownloadMonitor(client *api.Client, cfg *config.Manager) *DownloadMonitor {
	return &DownloadMonitor{
		client:         client,
		config:         cfg,
		stopCh:         make(chan struct{}),
		downloaded:     make(map[string]bool),
		localDownloads: make(map[string]*LocalDownloadTask),
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
	cfg := dm.config.GetConfig()
	if !cfg.LocalDownloadEnabled {
		return
	}

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
			dm.StartFileDownload(downloadURL, name)
			return
		}
	}
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
		"local_downloads":  len(dm.localDownloads),
	}
}

func (dm *DownloadMonitor) StartFileDownload(url, filename string) {
	dm.mu.Lock()
	task := &LocalDownloadTask{
		FileName:  filename,
		URL:       url,
		Status:    "downloading",
		StartTime: time.Now().Unix(),
	}
	dm.localDownloads[filename] = task
	dm.mu.Unlock()

	go dm.downloadFileWithProgress(url, filename, task)
}

func (dm *DownloadMonitor) GetLocalDownloadTasks() []LocalDownloadTask {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	tasks := make([]LocalDownloadTask, 0, len(dm.localDownloads))
	for _, task := range dm.localDownloads {
		tasks = append(tasks, *task)
	}
	return tasks
}

func (dm *DownloadMonitor) downloadFileWithProgress(url, filename string, task *LocalDownloadTask) {
	cfg := dm.config.GetConfig()
	destDir := cfg.DownloadDir
	os.MkdirAll(destDir, 0755)

	destPath := filepath.Join(destDir, filename)

	resp, err := http.Get(url)
	if err != nil {
		dm.mu.Lock()
		task.Status = "failed"
		task.Error = err.Error()
		dm.mu.Unlock()
		log.Printf("Failed to download %s: %v", filename, err)
		return
	}
	defer resp.Body.Close()

	dm.mu.Lock()
	task.Size = resp.ContentLength
	dm.mu.Unlock()

	file, err := os.Create(destPath)
	if err != nil {
		dm.mu.Lock()
		task.Status = "failed"
		task.Error = err.Error()
		dm.mu.Unlock()
		log.Printf("Failed to create file %s: %v", destPath, err)
		return
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	var downloaded int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				dm.mu.Lock()
				task.Status = "failed"
				task.Error = writeErr.Error()
				dm.mu.Unlock()
				log.Printf("Failed to write file %s: %v", filename, writeErr)
				return
			}
			downloaded += int64(n)
			dm.mu.Lock()
			task.Downloaded = downloaded
			if task.Size > 0 {
				task.Progress = float64(downloaded) / float64(task.Size) * 100
			}
			dm.mu.Unlock()
		}
		if err != nil {
			break
		}
	}

	dm.mu.Lock()
	if downloaded > 0 {
		task.Status = "completed"
		task.Progress = 100
		task.Downloaded = downloaded
	} else {
		task.Status = "failed"
		task.Error = "No data received"
	}
	dm.mu.Unlock()

	log.Printf("Downloaded %s (%d bytes) to %s", filename, downloaded, destPath)
}

func (dm *DownloadMonitor) CheckDownloadDir() map[string]interface{} {
	cfg := dm.config.GetConfig()
	result := map[string]interface{}{
		"download_dir": cfg.DownloadDir,
		"accessible":   false,
		"exists":       false,
		"writable":     false,
	}

	if cfg.DownloadDir == "" {
		result["message"] = "Download directory not configured"
		return result
	}

	info, err := os.Stat(cfg.DownloadDir)
	if err != nil {
		if os.IsNotExist(err) {
			result["message"] = "Download directory does not exist"
		} else {
			result["message"] = "Cannot access download directory: " + err.Error()
		}
		return result
	}
	result["exists"] = true

	if !info.IsDir() {
		result["message"] = "Download path is not a directory"
		return result
	}

	testFile := filepath.Join(cfg.DownloadDir, ".115manager_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		result["message"] = "Download directory is not writable: " + err.Error()
		return result
	}
	os.Remove(testFile)
	result["writable"] = true
	result["accessible"] = true
	result["message"] = "Download directory is ready"

	return result
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
