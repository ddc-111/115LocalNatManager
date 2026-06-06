package service

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"115localnatmanager/api"
	"115localnatmanager/config"
)

type DownloadStatus string

const (
	StatusPending    DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted  DownloadStatus = "completed"
	StatusFailed     DownloadStatus = "failed"
)

type DownloadRecord struct {
	InfoHash    string         `json:"info_hash"`
	Name        string         `json:"name"`
	FileID      string         `json:"file_id"`
	Status      DownloadStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
	StartTime   int64          `json:"start_time,omitempty"`
	EndTime     int64          `json:"end_time,omitempty"`
	FileName    string         `json:"file_name,omitempty"`
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

type DownloadMonitor struct {
	client          *api.Client
	config          *config.Manager
	logger          *Logger
	stopCh          chan struct{}
	mu              sync.Mutex
	running         bool
	records         map[string]*DownloadRecord
	localDownloads  map[string]*LocalDownloadTask
	downloadSem     chan struct{}
	dbPath          string
}

func NewDownloadMonitor(client *api.Client, cfg *config.Manager, logger *Logger) *DownloadMonitor {
	dataDir := cfg.GetDataDir()
	dbPath := filepath.Join(dataDir, "downloads.json")

	m := &DownloadMonitor{
		client:         client,
		config:         cfg,
		logger:         logger,
		stopCh:         make(chan struct{}),
		records:        make(map[string]*DownloadRecord),
		localDownloads: make(map[string]*LocalDownloadTask),
		dbPath:         dbPath,
	}

	m.loadDB()
	m.updateConcurrency()

	return m
}

func (dm *DownloadMonitor) loadDB() {
	data, err := os.ReadFile(dm.dbPath)
	if err != nil {
		return
	}

	var records []*DownloadRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}

	for _, r := range records {
		dm.records[r.InfoHash] = r
	}
	dm.logger.Info("Loaded %d download records from database", len(dm.records))
}

func (dm *DownloadMonitor) saveDB() {
	dm.mu.Lock()
	records := make([]*DownloadRecord, 0, len(dm.records))
	for _, r := range dm.records {
		records = append(records, r)
	}
	dm.mu.Unlock()

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		dm.logger.Error("Failed to marshal download records: %v", err)
		return
	}

	if err := os.WriteFile(dm.dbPath, data, 0644); err != nil {
		dm.logger.Error("Failed to save download records: %v", err)
	}
}

func (dm *DownloadMonitor) updateConcurrency() {
	cfg := dm.config.GetConfig()
	concurrency := cfg.DownloadConcurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	dm.downloadSem = make(chan struct{}, concurrency)
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
	dm.logger.Info("Download monitor started")
}

func (dm *DownloadMonitor) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if !dm.running {
		return
	}
	close(dm.stopCh)
	dm.running = false
	dm.logger.Info("Download monitor stopped")
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

	dm.checkTasks()

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

	dm.updateConcurrency()

	result, err := dm.client.GetDownloadTaskList(1)
	if err != nil {
		dm.logger.Error("Failed to get task list: %v", err)
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

	videoExts := map[string]bool{
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".ts": true, ".rmvb": true, ".rm": true, ".3gp": true,
	}

	for _, task := range tasks {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}

		status, _ := taskMap["status"].(float64)
		infoHash, _ := taskMap["info_hash"].(string)
		name, _ := taskMap["name"].(string)
		fileID, _ := taskMap["file_id"].(string)

		if status != 2 {
			continue
		}

		dm.mu.Lock()
		record, exists := dm.records[infoHash]
		if exists && (record.Status == StatusCompleted || record.Status == StatusDownloading) {
			dm.mu.Unlock()
			continue
		}

		if !exists {
			record = &DownloadRecord{
				InfoHash: infoHash,
				Name:     name,
				FileID:   fileID,
				Status:   StatusPending,
			}
			dm.records[infoHash] = record
		}
		dm.mu.Unlock()

		if cfg.DownloadMode == "video" {
			ext := strings.ToLower(filepath.Ext(name))
			if !videoExts[ext] {
				dm.logger.Info("Skipping non-video file: %s", name)
				dm.mu.Lock()
				record.Status = StatusCompleted
				record.EndTime = time.Now().Unix()
				dm.mu.Unlock()
				continue
			}
		}

		go dm.downloadWithSemaphore(taskMap, record)
	}

	dm.saveDB()
}

func (dm *DownloadMonitor) downloadWithSemaphore(task map[string]interface{}, record *DownloadRecord) {
	dm.downloadSem <- struct{}{}
	defer func() { <-dm.downloadSem }()

	dm.downloadCompletedFile(task, record)
}

func (dm *DownloadMonitor) downloadCompletedFile(task map[string]interface{}, record *DownloadRecord) {
	name, _ := task["name"].(string)
	fileID, _ := task["file_id"].(string)

	if fileID == "" {
		dm.logger.Warn("No file_id for task: %s", name)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "No file_id"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("Processing completed task: %s (file_id: %s)", name, fileID)

	dm.mu.Lock()
	record.Status = StatusDownloading
	record.StartTime = time.Now().Unix()
	dm.mu.Unlock()

	fileInfo, err := dm.client.GetFileInfo(fileID)
	if err != nil {
		dm.logger.Error("Failed to get file info for %s: %v", name, err)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = err.Error()
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("File info response for %s: %v", name, fileInfo)

	data, ok := fileInfo["data"].(map[string]interface{})
	if !ok {
		dm.logger.Error("Invalid file info response for %s, data type: %T", name, fileInfo["data"])
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "Invalid file info response"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	pickCode, _ := data["pick_code"].(string)
	if pickCode == "" {
		dm.logger.Warn("No pick_code for file: %s, data: %v", name, data)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "No pick_code"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("Got pick_code for %s: %s", name, pickCode)

	downloadInfo, err := dm.client.GetDownloadURL(pickCode)
	if err != nil {
		dm.logger.Error("Failed to get download URL for %s: %v", name, err)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = err.Error()
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("Download URL response for %s: %v", name, downloadInfo)

	dlData, ok := downloadInfo["data"].(map[string]interface{})
	if !ok {
		dm.logger.Error("Invalid download info response for %s, data type: %T", name, downloadInfo["data"])
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "Invalid download info response"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	for _, v := range dlData {
		fileData, ok := v.(map[string]interface{})
		if !ok {
			dm.logger.Error("Invalid file data type for %s: %T", name, v)
			continue
		}

		urlObj, ok := fileData["url"].(map[string]interface{})
		if !ok {
			dm.logger.Error("Invalid url object for %s: %v", name, fileData["url"])
			continue
		}

		downloadURL, _ := urlObj["url"].(string)
		fileName, _ := fileData["file_name"].(string)

		if downloadURL != "" {
			if fileName == "" {
				fileName = name
			}

			dm.mu.Lock()
			record.FileName = fileName
			dm.mu.Unlock()

			dm.logger.Info("Starting download for %s: %s", name, downloadURL)
			dm.StartFileDownload(downloadURL, fileName, record)
			return
		}
	}

	dm.logger.Error("Could not find download URL in response for %s", name)
	dm.mu.Lock()
	record.Status = StatusFailed
	record.Error = "Could not get download URL"
	record.EndTime = time.Now().Unix()
	dm.mu.Unlock()
}

func (dm *DownloadMonitor) GetStatus() map[string]interface{} {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	cfg := dm.config.GetConfig()

	downloadedCount := 0
	for _, r := range dm.records {
		if r.Status == StatusCompleted {
			downloadedCount++
		}
	}

	return map[string]interface{}{
		"running":           dm.running,
		"download_dir":      cfg.DownloadDir,
		"monitor_interval":  cfg.MonitorInterval,
		"downloaded_count":  downloadedCount,
		"local_downloads":   len(dm.localDownloads),
		"concurrency":       cfg.DownloadConcurrency,
		"total_records":     len(dm.records),
	}
}

func (dm *DownloadMonitor) StartFileDownload(url, filename string, record *DownloadRecord) {
	dm.mu.Lock()
	task := &LocalDownloadTask{
		FileName:  filename,
		URL:       url,
		Status:    "downloading",
		StartTime: time.Now().Unix(),
	}
	dm.localDownloads[filename] = task
	dm.mu.Unlock()

	dm.logger.Info("Starting file download: %s", filename)
	go dm.downloadFileWithProgress(url, filename, task, record)
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

func (dm *DownloadMonitor) GetDownloadedFiles() map[string]string {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result := make(map[string]string)
	for _, r := range dm.records {
		result[r.Name] = string(r.Status)
	}
	return result
}

func (dm *DownloadMonitor) downloadFileWithProgress(url, filename string, task *LocalDownloadTask, record *DownloadRecord) {
	cfg := dm.config.GetConfig()
	destDir := cfg.DownloadDir
	os.MkdirAll(destDir, 0755)

	destPath := filepath.Join(destDir, filename)
	dm.logger.Info("Starting download: %s -> %s", filename, destPath)

	resp, err := http.Get(url)
	if err != nil {
		dm.mu.Lock()
		task.Status = "failed"
		task.Error = err.Error()
		if record != nil {
			record.Status = StatusFailed
			record.Error = err.Error()
			record.EndTime = time.Now().Unix()
		}
		dm.mu.Unlock()
		dm.logger.Error("Failed to download %s: %v", filename, err)
		dm.saveDB()
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
		if record != nil {
			record.Status = StatusFailed
			record.Error = err.Error()
			record.EndTime = time.Now().Unix()
		}
		dm.mu.Unlock()
		dm.logger.Error("Failed to create file %s: %v", destPath, err)
		dm.saveDB()
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
				if record != nil {
					record.Status = StatusFailed
					record.Error = writeErr.Error()
					record.EndTime = time.Now().Unix()
				}
				dm.mu.Unlock()
				dm.logger.Error("Failed to write file %s: %v", filename, writeErr)
				dm.saveDB()
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
		if record != nil {
			record.Status = StatusCompleted
			record.EndTime = time.Now().Unix()
		}
		dm.logger.Info("Downloaded %s (%d bytes) to %s", filename, downloaded, destPath)
	} else {
		task.Status = "failed"
		task.Error = "No data received"
		if record != nil {
			record.Status = StatusFailed
			record.Error = "No data received"
			record.EndTime = time.Now().Unix()
		}
		dm.logger.Error("Downloaded %s but no data received", filename)
	}
	dm.mu.Unlock()
	dm.saveDB()
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
