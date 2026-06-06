package handler

import (
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

	type dirResult struct {
		entries []os.DirEntry
		err     error
	}

	ch := make(chan dirResult, 1)
	go func() {
		entries, err := os.ReadDir(dir)
		ch <- dirResult{entries, err}
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			writeJSON(w, http.StatusBadRequest, model.APIResponse{
				State:   false,
				Message: "无法读取目录: " + result.err.Error(),
			})
			return
		}

		var dirs []DirEntry
		var files []DirEntry

		for _, entry := range result.entries {
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

		result2 := append(dirs, files...)

		writeJSON(w, http.StatusOK, model.APIResponse{
			State: true,
			Data: map[string]interface{}{
				"current": dir,
				"parent":  filepath.Dir(dir),
				"entries": result2,
			},
		})

	case <-time.After(10 * time.Second):
		writeJSON(w, http.StatusRequestTimeout, model.APIResponse{
			State:   false,
			Message: "读取目录超时，可能是SMB挂载不可用",
		})
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

		// macOS: 列出 /Volumes 下的挂载点
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
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data:  drives,
	})
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

	type testResult struct {
		info os.FileInfo
		err  error
	}

	ch := make(chan testResult, 1)
	go func() {
		info, err := os.Stat(dir)
		ch <- testResult{info, err}
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			writeJSON(w, http.StatusOK, model.APIResponse{
				State:   false,
				Message: "目录不存在或无法访问",
			})
			return
		}

		if !result.info.IsDir() {
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
				"name": result.info.Name(),
			},
		})

	case <-time.After(10 * time.Second):
		writeJSON(w, http.StatusRequestTimeout, model.APIResponse{
			State:   false,
			Message: "检测目录超时，可能是SMB挂载不可用",
		})
	}
}
