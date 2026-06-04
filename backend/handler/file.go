package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"115localnatmanager/api"
	"115localnatmanager/model"

	"github.com/gorilla/mux"
)

type FileHandler struct {
	client *api.Client
}

func NewFileHandler(client *api.Client) *FileHandler {
	return &FileHandler{client: client}
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
