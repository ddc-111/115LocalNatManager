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
	Speed       int64   `json:"speed"`
	Error       string  `json:"error,omitempty"`
	StartTime   int64   `json:"start_time"`
	Cancel      chan struct{} `json:"-"`
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
	dirAccessible   bool
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
		dirAccessible:  true,
	}

	m.loadDB()
	m.updateConcurrency()
	m.checkDownloadDirStatus()

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

	dirCheckTicker := time.NewTicker(30 * time.Second)
	defer dirCheckTicker.Stop()

	for {
		select {
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.checkTasks()
		case <-dirCheckTicker.C:
			dm.checkDownloadDirStatus()
		}
	}
}

func (dm *DownloadMonitor) checkDownloadDirStatus() {
	cfg := dm.config.GetConfig()
	accessible := dm.isDirAccessible(cfg.DownloadDir)
	
	dm.mu.Lock()
	dm.dirAccessible = accessible
	dm.mu.Unlock()
	
	if !accessible {
		dm.logger.Warn("[监控] 下载目录不可访问: %s", cfg.DownloadDir)
	}
}

func (dm *DownloadMonitor) checkTasks() {
	cfg := dm.config.GetConfig()
	if !cfg.LocalDownloadEnabled {
		dm.logger.Info("[检测] 本地下载未启用，跳过检测")
		return
	}

	dm.updateConcurrency()

	dm.logger.Info("[检测] 开始检查云下载任务...")

	result, err := dm.client.GetDownloadTaskList(1)
	if err != nil {
		dm.logger.Error("[检测] 获取任务列表失败: %v", err)
		return
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		dm.logger.Error("[检测] 任务列表响应格式错误")
		return
	}

	tasks, ok := data["tasks"].([]interface{})
	if !ok {
		dm.logger.Error("[检测] 任务列表为空或格式错误")
		return
	}

	dm.logger.Info("[检测] 获取到 %d 个任务", len(tasks))

	videoExts := map[string]bool{
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".ts": true, ".rmvb": true, ".rm": true, ".3gp": true,
	}

	completedCount := 0
	pendingCount := 0
	skippedCount := 0

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
		completedCount++

		dm.mu.Lock()
		record, exists := dm.records[infoHash]
		if exists && record.Status == StatusCompleted {
			dm.mu.Unlock()
			skippedCount++
			continue
		}
		if exists && record.Status == StatusDownloading {
			dm.mu.Unlock()
			dm.logger.Info("[检测] 任务 %s 正在下载中，跳过", name)
			skippedCount++
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
				dm.logger.Info("[检测] 跳过非视频文件: %s (扩展名: %s)", name, ext)
				dm.mu.Lock()
				if record != nil {
					record.Status = StatusCompleted
					record.EndTime = time.Now().Unix()
				} else {
					record = &DownloadRecord{
						InfoHash: infoHash,
						Name:     name,
						FileID:   fileID,
						Status:   StatusCompleted,
						EndTime:  time.Now().Unix(),
					}
					dm.records[infoHash] = record
				}
				dm.mu.Unlock()
				skippedCount++
				continue
			}
		}

		pendingCount++
		dm.logger.Info("[检测] 发现待下载任务: %s (file_id: %s)", name, fileID)
		go dm.downloadWithSemaphore(taskMap, record)
	}

	dm.logger.Info("[检测] 检测完成: 共 %d 个已完成任务, %d 个待下载, %d 个已跳过", completedCount, pendingCount, skippedCount)
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
	fileCategory, _ := task["file_category"].(float64)

	if fileID == "" {
		dm.logger.Error("[下载] 任务 %s 没有 file_id", name)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "No file_id"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 开始处理任务: %s (file_id: %s, category: %v)", name, fileID, fileCategory)

	// 如果是文件夹，获取文件夹内的文件列表
	if fileCategory == 0 {
		dm.logger.Info("[下载] %s 是文件夹，获取文件列表...", name)
		dm.processFolder(name, fileID, record)
		return
	}

	dm.downloadSingleFile(name, fileID, record)
}

func (dm *DownloadMonitor) processFolder(folderName, folderID string, record *DownloadRecord) {
	fileList, err := dm.client.GetFileList(folderID, 100, 0)
	if err != nil {
		dm.logger.Error("[下载] 获取文件夹 %s 内容失败: %v", folderName, err)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "获取文件夹内容失败: " + err.Error()
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	data, ok := fileList["data"].([]interface{})
	if !ok {
		dm.logger.Error("[下载] 文件夹 %s 响应格式错误", folderName)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "文件夹响应格式错误"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 文件夹 %s 包含 %d 个文件", folderName, len(data))

	dm.mu.Lock()
	record.Status = StatusCompleted
	record.EndTime = time.Now().Unix()
	dm.mu.Unlock()

	for i, item := range data {
		fileData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		fileID, _ := fileData["fid"].(string)
		fileName, _ := fileData["fn"].(string)
		fc, _ := fileData["fc"].(string)

		if fc == "0" {
			dm.logger.Info("[下载] 跳过子文件夹: %s", fileName)
			continue
		}

		dm.logger.Info("[下载] 处理文件夹 %s 中的文件 %d/%d: %s", folderName, i+1, len(data), fileName)

		newRecord := &DownloadRecord{
			InfoHash: folderID + "_" + fileID,
			Name:     fileName,
			FileID:   fileID,
			Status:   StatusPending,
		}

		dm.mu.Lock()
		dm.records[newRecord.InfoHash] = newRecord
		dm.mu.Unlock()

		go dm.downloadSingleFile(fileName, fileID, newRecord)
	}
}

func (dm *DownloadMonitor) downloadSingleFile(name, fileID string, record *DownloadRecord) {

	dm.mu.Lock()
	record.Status = StatusDownloading
	record.StartTime = time.Now().Unix()
	dm.mu.Unlock()

	dm.logger.Info("[下载] 步骤1: 获取文件信息...")
	fileInfo, err := dm.client.GetFileInfo(fileID)
	if err != nil {
		dm.logger.Error("[下载] 步骤1失败: 获取文件信息失败 %s: %v", name, err)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "获取文件信息失败: " + err.Error()
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 步骤1完成: 文件信息响应 %s", name)

	data, ok := fileInfo["data"].(map[string]interface{})
	if !ok {
		dm.logger.Error("[下载] 步骤1失败: 文件信息响应格式错误 %s, data类型: %T", name, fileInfo["data"])
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "文件信息响应格式错误"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	pickCode, _ := data["pick_code"].(string)
	if pickCode == "" {
		dm.logger.Error("[下载] 步骤1失败: 文件 %s 没有 pick_code, data: %v", name, data)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "没有 pick_code"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 步骤1成功: 获取到 pick_code: %s", pickCode)

	dm.logger.Info("[下载] 步骤2: 获取下载链接...")
	downloadInfo, err := dm.client.GetDownloadURL(pickCode)
	if err != nil {
		dm.logger.Error("[下载] 步骤2失败: 获取下载链接失败 %s: %v", name, err)
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "获取下载链接失败: " + err.Error()
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 步骤2完成: 下载链接响应 %s: %v", name, downloadInfo)

	dlData, ok := downloadInfo["data"].(map[string]interface{})
	if !ok {
		// 处理空数组情况
		if _, isArray := downloadInfo["data"].([]interface{}); isArray {
			dm.logger.Error("[下载] 步骤2失败: 下载链接为空，文件可能不可下载 %s", name)
			dm.mu.Lock()
			record.Status = StatusFailed
			record.Error = "下载链接为空，文件可能不可下载"
			record.EndTime = time.Now().Unix()
			dm.mu.Unlock()
			return
		}
		dm.logger.Error("[下载] 步骤2失败: 下载链接响应格式错误 %s, data类型: %T", name, downloadInfo["data"])
		dm.mu.Lock()
		record.Status = StatusFailed
		record.Error = "下载链接响应格式错误"
		record.EndTime = time.Now().Unix()
		dm.mu.Unlock()
		return
	}

	dm.logger.Info("[下载] 步骤2成功: 解析下载链接数据，共 %d 个文件", len(dlData))

	for k, v := range dlData {
		fileData, ok := v.(map[string]interface{})
		if !ok {
			dm.logger.Error("[下载] 文件 %s 的数据格式错误: %T", k, v)
			continue
		}

		urlObj, ok := fileData["url"].(map[string]interface{})
		if !ok {
			dm.logger.Error("[下载] 文件 %s 的 URL 格式错误: %v", k, fileData["url"])
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

			dm.logger.Info("[下载] 步骤3: 开始下载文件 %s -> %s", name, fileName)
			dm.StartFileDownload(downloadURL, fileName, record)
			return
		}
	}

	dm.logger.Error("[下载] 步骤2失败: 在响应中找不到下载链接 %s", name)
	dm.mu.Lock()
	record.Status = StatusFailed
	record.Error = "找不到下载链接"
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
		"dir_accessible":    dm.dirAccessible,
	}
}

func (dm *DownloadMonitor) StartFileDownload(url, filename string, record *DownloadRecord) {
	dm.mu.Lock()
	task := &LocalDownloadTask{
		FileName:  filename,
		URL:       url,
		Status:    "downloading",
		StartTime: time.Now().Unix(),
		Cancel:    make(chan struct{}),
	}
	dm.localDownloads[filename] = task
	dm.mu.Unlock()

	dm.logger.Info("[下载] 开始下载: %s", filename)
	go dm.downloadFileWithProgress(url, filename, task, record)
}

func (dm *DownloadMonitor) CancelDownload(filename string) bool {
	dm.mu.Lock()
	task, exists := dm.localDownloads[filename]
	dm.mu.Unlock()

	if !exists || task.Status != "downloading" {
		return false
	}

	close(task.Cancel)
	task.Status = "cancelled"
	task.Error = "用户取消下载"

	cfg := dm.config.GetConfig()
	destPath := filepath.Join(cfg.DownloadDir, filename)
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		dm.logger.Error("[下载] 删除文件失败: %s: %v", destPath, err)
	} else {
		dm.logger.Info("[下载] 已删除未完成文件: %s", destPath)
	}

	return true
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

func (dm *DownloadMonitor) GetDownloadedFiles() map[string]interface{} {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result := make(map[string]interface{})
	for _, r := range dm.records {
		result[r.Name] = map[string]interface{}{
			"status": string(r.Status),
			"error":  r.Error,
			"name":   r.Name,
		}
	}
	return result
}

func (dm *DownloadMonitor) downloadFileWithProgress(url, filename string, task *LocalDownloadTask, record *DownloadRecord) {
	cfg := dm.config.GetConfig()
	destDir := cfg.DownloadDir
	
	// 检查下载目录是否可访问
	if !dm.isDirAccessible(destDir) {
		dm.logger.Warn("[下载] 下载目录不可访问: %s，等待目录恢复...", destDir)
		dm.mu.Lock()
		task.Status = "waiting"
		task.Error = "下载目录不可访问，等待恢复"
		dm.mu.Unlock()
		
		// 等待目录恢复
		for {
			time.Sleep(10 * time.Second)
			if dm.isDirAccessible(destDir) {
				dm.logger.Info("[下载] 下载目录已恢复: %s", destDir)
				break
			}
		}
	}

	os.MkdirAll(destDir, 0755)

	destPath := filepath.Join(destDir, filename)
	dm.logger.Info("[下载] 开始下载: %s -> %s", filename, destPath)

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
		dm.logger.Error("[下载] 下载失败 %s: %v", filename, err)
		dm.saveDB()
		return
	}
	defer resp.Body.Close()

	dm.mu.Lock()
	task.Size = resp.ContentLength
	task.Status = "downloading"
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
		dm.logger.Error("[下载] 创建文件失败 %s: %v", destPath, err)
		dm.saveDB()
		return
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	var downloaded int64
	var lastSpeedUpdate int64
	var lastDownloaded int64
	speedTicker := time.NewTicker(time.Second)
	defer speedTicker.Stop()

	stopSpeed := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-task.Cancel:
				return
			case <-stopSpeed:
				return
			case <-speedTicker.C:
				dm.mu.Lock()
				now := time.Now().Unix()
				if lastSpeedUpdate > 0 {
					elapsed := now - lastSpeedUpdate
					if elapsed > 0 {
						task.Speed = (downloaded - lastDownloaded) / elapsed
					}
				}
				lastSpeedUpdate = now
				lastDownloaded = downloaded
				dm.mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-task.Cancel:
			dm.logger.Info("[下载] 下载被取消: %s", filename)
			close(stopSpeed)
			file.Close()
			os.Remove(destPath)
			return
		default:
		}

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
				dm.logger.Error("[下载] 写入文件失败 %s: %v", filename, writeErr)
				close(stopSpeed)
				os.Remove(destPath)
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

	close(stopSpeed)
	<-done

	dm.mu.Lock()
	if task.Status == "cancelled" {
		dm.mu.Unlock()
		return
	}

	if downloaded > 0 {
		task.Status = "completed"
		task.Progress = 100
		task.Downloaded = downloaded
		task.Speed = 0
		if record != nil {
			record.Status = StatusCompleted
			record.EndTime = time.Now().Unix()
		}
		dm.logger.Info("[下载] 下载完成 %s (%d bytes) -> %s", filename, downloaded, destPath)
	} else {
		task.Status = "failed"
		task.Error = "没有接收到数据"
		if record != nil {
			record.Status = StatusFailed
			record.Error = "没有接收到数据"
			record.EndTime = time.Now().Unix()
		}
		dm.logger.Error("[下载] 下载失败 %s: 没有接收到数据", filename)
		os.Remove(destPath)
	}
	dm.mu.Unlock()
	dm.saveDB()
}

func (dm *DownloadMonitor) isDirAccessible(dir string) bool {
	if dir == "" {
		return false
	}
	
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	
	testFile := filepath.Join(dir, ".115manager_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return false
	}
	os.Remove(testFile)
	return true
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
