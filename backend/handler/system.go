package handler

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"115localnatmanager/model"
)

type SystemHandler struct {
	config interface {
		GetConfig() model.Config
		SaveConfig() error
		UpdateConfig(model.ConfigUpdateRequest)
	}
}

func NewSystemHandler(cfg interface {
	GetConfig() model.Config
	SaveConfig() error
	UpdateConfig(model.ConfigUpdateRequest)
}) *SystemHandler {
	return &SystemHandler{config: cfg}
}

type DirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

func (h *SystemHandler) ListDirectory(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		if runtime.GOOS == "windows" {
			dir = "C:\\"
		} else {
			dir = "/"
		}
	}

	// 快速检测目录是否可访问
	if !isPathAccessible(dir) {
		writeJSON(w, http.StatusRequestTimeout, model.APIResponse{
			State:   false,
			Message: "目录不可访问或超时，可能是SMB挂载不可用",
		})
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "无法读取目录: " + err.Error(),
		})
		return
	}

	var dirs []DirEntry
	var files []DirEntry

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := DirEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(dir, entry.Name()),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		}

		if entry.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	result := append(dirs, files...)

	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data: map[string]interface{}{
			"current": dir,
			"parent":  filepath.Dir(dir),
			"entries": result,
		},
	})
}

func isNetworkMount(path string) bool {
	return strings.Contains(path, "/Volumes/") ||
		strings.Contains(path, "\\\\") ||
		strings.HasPrefix(path, "//") ||
		strings.HasPrefix(path, "/mnt/") ||
		strings.HasPrefix(path, "/media/")
}

func isPathAccessible(path string) bool {
	type result struct {
		accessible bool
	}

	ch := make(chan result, 1)
	go func() {
		f, err := os.Open(path)
		if err != nil {
			ch <- result{false}
			return
		}
		f.Close()
		ch <- result{true}
	}()

	timeout := 3 * time.Second
	if isNetworkMount(path) {
		timeout = 30 * time.Second
	}

	select {
	case r := <-ch:
		return r.accessible
	case <-time.After(timeout):
		return false
	}
}

func (h *SystemHandler) GetDrives(w http.ResponseWriter, r *http.Request) {
	var drives []DirEntry

	if runtime.GOOS == "windows" {
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			path := string(drive) + ":\\"
			if _, err := os.Stat(path); err == nil {
				drives = append(drives, DirEntry{
					Name:  string(drive) + ":",
					Path:  path,
					IsDir: true,
				})
			}
		}
	} else {
		drives = append(drives, DirEntry{
			Name:  "/",
			Path:  "/",
			IsDir: true,
		})

		if runtime.GOOS == "darwin" {
			volumes, err := os.ReadDir("/Volumes")
			if err == nil {
				for _, vol := range volumes {
					if vol.IsDir() && !strings.HasPrefix(vol.Name(), ".") {
						drives = append(drives, DirEntry{
							Name:  vol.Name(),
							Path:  filepath.Join("/Volumes", vol.Name()),
							IsDir: true,
						})
					}
				}
			}
		}

		if runtime.GOOS == "linux" {
			added := make(map[string]bool)
			added["/"] = true

			for _, mountDir := range []string{"/mnt", "/media"} {
				entries, err := os.ReadDir(mountDir)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
						continue
					}
					mountPath := filepath.Join(mountDir, entry.Name())
					if !added[mountPath] {
						drives = append(drives, DirEntry{
							Name:  entry.Name(),
							Path:  mountPath,
							IsDir: true,
						})
						added[mountPath] = true
					}
				}
			}

			networkMounts := getLinuxNetworkMounts()
			for _, mp := range networkMounts {
				if !added[mp] {
					name := filepath.Base(mp)
					drives = append(drives, DirEntry{
						Name:  name,
						Path:  mp,
						IsDir: true,
					})
					added[mp] = true
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data:  drives,
	})
}

func getLinuxNetworkMounts() []string {
	var mounts []string
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return mounts
	}
	defer f.Close()

	networkFS := map[string]bool{
		"nfs": true, "nfs4": true, "cifs": true, "smbfs": true,
		"fuse.sshfs": true, "fuse.gvfsd-fuse": true, "fuse.s3fs": true,
		"fuse.jmtpfs": true, "fuse.go-mtpfs": true, "fuse.bindfs": true,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		mountPoint := fields[1]
		fsType := fields[2]
		if networkFS[fsType] && mountPoint != "/" && !strings.HasPrefix(mountPoint, "/proc") &&
			!strings.HasPrefix(mountPoint, "/sys") && !strings.HasPrefix(mountPoint, "/dev") {
			mounts = append(mounts, mountPoint)
		}
	}
	return mounts
}

func (h *SystemHandler) CreateDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "无效请求",
		})
		return
	}

	fullPath := filepath.Join(req.Path, req.Name)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: "创建目录失败: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State:   true,
		Message: "目录创建成功",
	})
}

func (h *SystemHandler) TestDirectory(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "目录路径不能为空",
		})
		return
	}

	// 快速检测目录是否可访问
	if !isPathAccessible(dir) {
		writeJSON(w, http.StatusRequestTimeout, model.APIResponse{
			State:   false,
			Message: "目录不可访问或超时，可能是SMB挂载不可用",
		})
		return
	}

	info, err := os.Stat(dir)
	if err != nil {
		writeJSON(w, http.StatusOK, model.APIResponse{
			State:   false,
			Message: "目录不存在或无法访问",
		})
		return
	}

	if !info.IsDir() {
		writeJSON(w, http.StatusOK, model.APIResponse{
			State:   false,
			Message: "路径不是目录",
		})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State:   true,
		Message: "目录有效",
		Data: map[string]interface{}{
			"path": dir,
			"name": info.Name(),
		},
	})
}
