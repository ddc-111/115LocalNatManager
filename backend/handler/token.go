package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"115localnatmanager/config"
	"115localnatmanager/model"
)

type TokenHandler struct {
	config *config.Manager
}

func NewTokenHandler(cfg *config.Manager) *TokenHandler {
	return &TokenHandler{config: cfg}
}

func (h *TokenHandler) SetToken(w http.ResponseWriter, r *http.Request) {
	var req model.SetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "Invalid request body",
		})
		return
	}

	if req.RefreshToken != "" {
		h.config.SetRefreshToken(req.RefreshToken)
	}
	
	if req.AccessToken != "" {
		h.config.SetAccessToken(req.AccessToken, 7200)
	}

	if req.RefreshToken == "" && req.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResponse{
			State:   false,
			Message: "refresh_token or access_token is required",
		})
		return
	}

	if err := h.config.SaveToken(); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResponse{
			State:   false,
			Message: "Failed to save token",
		})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResponse{
		State:   true,
		Message: "Token saved successfully",
	})
}

func (h *TokenHandler) GetTokenStatus(w http.ResponseWriter, r *http.Request) {
	token := h.config.GetToken()

	hasRefreshToken := token.RefreshToken != ""
	hasAccessToken := token.AccessToken != ""
	isExpired := token.ExpiresAt.IsZero() || token.ExpiresAt.Before(time.Now())

	writeJSON(w, http.StatusOK, model.APIResponse{
		State: true,
		Data: map[string]interface{}{
			"has_refresh_token": hasRefreshToken,
			"has_access_token":  hasAccessToken,
			"is_expired":        isExpired,
			"expires_at":        token.ExpiresAt,
		},
	})
}

func timeNow() time.Time {
	return time.Now()
}
