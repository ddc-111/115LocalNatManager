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

type DownloadHandler struct {
	client    *api.Client
	monitor   *service.DownloadMonitor
}

func NewDownloadHandler(client *api.Client, monitor *service.DownloadMonitor) *DownloadHandler {
	return &DownloadHandler{
		client:  client,
		monitor: monitor,
	}
}

func (h *DownloadHandler) AddTask(w http.ResponseWriter, r *http.Request) {
	var req model.CloudDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.URLs == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "urls is required",
		})
		return
	}

	result, err := h.client.AddDownloadTask(req.URLs, req.PathID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DownloadHandler) GetTaskList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	result, err := h.client.GetDownloadTaskList(page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DownloadHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	infoHash := vars["hash"]

	var req struct {
		DelSource bool `json:"del_source"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := h.client.DeleteDownloadTask(infoHash, req.DelSource)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DownloadHandler) ClearTasks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Flag int `json:"flag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	result, err := h.client.ClearDownloadTasks(req.Flag)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DownloadHandler) GetQuota(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.GetDownloadQuota()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DownloadHandler) GetMonitorStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data:  h.monitor.GetStatus(),
	})
}

func (h *DownloadHandler) ToggleMonitor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	switch req.Action {
	case "start":
		h.monitor.Start()
	case "stop":
		h.monitor.Stop()
	default:
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "action must be start or stop",
		})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State:   true,
		Message: "Monitor " + req.Action + "ed",
		Data:    h.monitor.GetStatus(),
	})
}

func (h *DownloadHandler) CheckDownloadDir(w http.ResponseWriter, r *http.Request) {
	result := h.monitor.CheckDownloadDir()
	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data:  result,
	})
}
