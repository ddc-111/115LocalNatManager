package model

import "time"

type TokenData struct {
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Config struct {
	Port            string `json:"port"`
	DownloadDir     string `json:"download_dir"`
	MonitorEnabled  bool   `json:"monitor_enabled"`
	MonitorInterval int    `json:"monitor_interval"`
	DefaultSavePath string `json:"default_save_path"`
}

type APIResponse struct {
	State   bool        `json:"state"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
}

type FileInfo struct {
	FID        string `json:"fid"`
	ParentID   string `json:"pid"`
	FileName   string `json:"fn"`
	FileSize   int64  `json:"fs"`
	FileType   string `json:"fc"`
	SHA1       string `json:"sha1"`
	PickCode   string `json:"pc"`
	IsDir      bool   `json:"is_dir"`
	UpdateTime int64  `json:"upt"`
	UploadTime int64  `json:"uppt"`
	Icon       string `json:"ico"`
}

type FolderInfo struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
}

type DownloadTask struct {
	InfoHash    string `json:"info_hash"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	PercentDone int    `json:"percentDone"`
	Status      int    `json:"status"`
	AddTime     int64  `json:"add_time"`
	URL         string `json:"url"`
	FileID      string `json:"file_id"`
}

type CloudDownloadRequest struct {
	URLs     string `json:"urls"`
	PathID   string `json:"path_id"`
}

type SetTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type ConfigUpdateRequest struct {
	Port            string `json:"port,omitempty"`
	DownloadDir     string `json:"download_dir,omitempty"`
	MonitorEnabled  *bool  `json:"monitor_enabled,omitempty"`
	MonitorInterval *int   `json:"monitor_interval,omitempty"`
	DefaultSavePath string `json:"default_save_path,omitempty"`
}
