package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"115localnatmanager/api"
	"115localnatmanager/model"
	"115localnatmanager/service"

	"github.com/gorilla/mux"
)

type FileHandler struct {
	client  *api.Client
	monitor *service.DownloadMonitor
}

func NewFileHandler(client *api.Client, monitor *service.DownloadMonitor) *FileHandler {
	return &FileHandler{client: client, monitor: monitor}
}

func (h *FileHandler) GetFileList(w http.ResponseWriter, r *http.Request) {
	cid := r.URL.Query().Get("cid")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 20
	}

	result, err := h.client.GetFileList(cid, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	result, err := h.client.GetFileInfo(fileID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) SearchFiles(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if keyword == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "search query is required",
		})
		return
	}

	if limit <= 0 {
		limit = 20
	}

	result, err := h.client.SearchFiles(keyword, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentID string `json:"parent_id"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "folder name is required",
		})
		return
	}

	result, err := h.client.CreateFolder(req.ParentID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs string `json:"file_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.FileIDs == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "file_ids is required",
		})
		return
	}

	result, err := h.client.DeleteFiles(req.FileIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) RenameFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "new name is required",
		})
		return
	}

	result, err := h.client.RenameFile(fileID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) MoveFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs string `json:"file_ids"`
		ToCID   string `json:"to_cid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.FileIDs == "" || req.ToCID == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "file_ids and to_cid are required",
		})
		return
	}

	result, err := h.client.MoveFiles(req.FileIDs, req.ToCID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PickCode string `json:"pick_code"`
		FileName string `json:"file_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.PickCode == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "pick_code is required",
		})
		return
	}

	downloadInfo, err := h.client.GetDownloadURL(req.PickCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	dlData, ok := downloadInfo["data"].(map[string]interface{})
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: "Invalid download info response",
		})
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
		fileName, _ := fileData["file_name"].(string)

		if downloadURL != "" {
			if req.FileName != "" {
				fileName = req.FileName
			}
			if fileName == "" {
				fileName = "download"
			}

			h.monitor.StartFileDownload(downloadURL, fileName)

			writeJSON(w, http.StatusOK, model.APIResponse{
				State:   true,
				Message: "Download started",
				Data: map[string]interface{}{
					"file_name": fileName,
				},
			})
			return
		}
	}

	writeJSON(w, http.StatusInternalServerError, model.APIResponse{
		State:   false,
		Message: "Could not get download URL",
	})
}

func (h *FileHandler) GetLocalDownloads(w http.ResponseWriter, r *http.Request) {
	tasks := h.monitor.GetLocalDownloadTasks()
	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data:  tasks,
	})
}

func (h *FileHandler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PickCode string `json:"pick_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.PickCode == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "pick_code is required",
		})
		return
	}

	downloadInfo, err := h.client.GetDownloadURL(req.PickCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	dlData, ok := downloadInfo["data"].(map[string]interface{})
	if !ok {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: "Invalid download info response",
		})
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
			writeJSON(w, http.StatusOK, model.APIResponse{
				State: true,
				Data: map[string]interface{}{
					"url":       downloadURL,
					"file_name": fileData["file_name"],
				},
			})
			return
		}
	}

	writeJSON(w, http.StatusInternalServerError, model.APIResponse{
		State:   false,
		Message: "Could not get download URL",
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
