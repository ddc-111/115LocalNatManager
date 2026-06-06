package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"115localnatmanager/api"
	"115localnatmanager/config"
	"115localnatmanager/model"
	"115localnatmanager/service"

	"github.com/gorilla/mux"
)

func NewRouter(client *api.Client, cfg *config.Manager, monitor *service.DownloadMonitor, logger *service.Logger) *mux.Router {
	r := mux.NewRouter()
	r.Use(CORSMiddleware)

	tokenHandler := NewTokenHandler(cfg)
	fileHandler := NewFileHandler(client, monitor)
	downloadHandler := NewDownloadHandler(client, monitor)
	systemHandler := NewSystemHandler(cfg)

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/token", tokenHandler.SetToken).Methods("POST", "OPTIONS")
	api.HandleFunc("/token", tokenHandler.GetTokenStatus).Methods("GET", "OPTIONS")

	api.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		result, err := client.GetUserInfo()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, model.APIResponse{
				State:   false,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/files", fileHandler.GetFileList).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/search", fileHandler.SearchFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/delete", fileHandler.DeleteFiles).Methods("POST", "OPTIONS")
	api.HandleFunc("/files/move", fileHandler.MoveFiles).Methods("POST", "OPTIONS")
	api.HandleFunc("/files/download", fileHandler.DownloadFile).Methods("POST", "OPTIONS")
	api.HandleFunc("/files/download-url", fileHandler.GetDownloadURL).Methods("POST", "OPTIONS")
	api.HandleFunc("/files/local-downloads", fileHandler.GetLocalDownloads).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/{id}", fileHandler.GetFileInfo).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/{id}", fileHandler.RenameFile).Methods("PUT", "OPTIONS")
	api.HandleFunc("/folders", fileHandler.CreateFolder).Methods("POST", "OPTIONS")

	api.HandleFunc("/download", downloadHandler.AddTask).Methods("POST", "OPTIONS")
	api.HandleFunc("/download/tasks", downloadHandler.GetTaskList).Methods("GET", "OPTIONS")
	api.HandleFunc("/download/clear", downloadHandler.ClearTasks).Methods("POST", "OPTIONS")
	api.HandleFunc("/download/quota", downloadHandler.GetQuota).Methods("GET", "OPTIONS")
	api.HandleFunc("/download/check-dir", downloadHandler.CheckDownloadDir).Methods("GET", "OPTIONS")
	api.HandleFunc("/download/downloaded", downloadHandler.GetDownloadedFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/download/monitor", downloadHandler.GetMonitorStatus).Methods("GET", "OPTIONS")
	api.HandleFunc("/download/monitor", downloadHandler.ToggleMonitor).Methods("POST", "OPTIONS")
	api.HandleFunc("/download/tasks/{hash}", downloadHandler.DeleteTask).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, model.APIResponse{
			State: true,
			Data:  cfg.GetConfig(),
		})
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		var req model.ConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, model.APIResponse{
				State:   false,
				Message: "Invalid request body",
			})
			return
		}
		cfg.UpdateConfig(req)
		cfg.SaveConfig()
		writeJSON(w, http.StatusOK, model.APIResponse{
			State: true,
			Data:  cfg.GetConfig(),
		})
	}).Methods("PUT", "OPTIONS")

	api.HandleFunc("/system/drives", systemHandler.GetDrives).Methods("GET", "OPTIONS")
	api.HandleFunc("/system/dirs", systemHandler.ListDirectory).Methods("GET", "OPTIONS")
	api.HandleFunc("/system/dirs/create", systemHandler.CreateDirectory).Methods("POST", "OPTIONS")
	api.HandleFunc("/system/dirs/test", systemHandler.TestDirectory).Methods("GET", "OPTIONS")

	api.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		level := r.URL.Query().Get("level")
		limit := 100
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}
		logs := logger.GetLogs(level, limit)
		writeJSON(w, http.StatusOK, model.APIResponse{
			State: true,
			Data:  logs,
		})
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		logger.Clear()
		writeJSON(w, http.StatusOK, model.APIResponse{
			State:   true,
			Message: "Logs cleared",
		})
	}).Methods("POST", "OPTIONS")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods("GET")

	return r
}
