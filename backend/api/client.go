package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"115localnatmanager/config"
)

const (
	BaseURL      = "https://proapi.115.com"
	PassportURL  = "https://passportapi.115.com"
)

type Client struct {
	httpClient *http.Client
	config     *config.Manager
	mu         sync.Mutex
}

func NewClient(cfg *config.Manager) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		config:     cfg,
	}
}

func (c *Client) getAccessToken() (string, error) {
	token := c.config.GetToken()
	if token.AccessToken == "" || time.Now().After(token.ExpiresAt) {
		if err := c.refreshAccessToken(); err != nil {
			return "", fmt.Errorf("token refresh failed: %w", err)
		}
		token = c.config.GetToken()
	}
	return token.AccessToken, nil
}

func (c *Client) refreshAccessToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	token := c.config.GetToken()
	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token configured")
	}

	data := url.Values{}
	data.Set("refresh_token", token.RefreshToken)

	req, err := http.NewRequest("POST", PassportURL+"/open/refreshToken", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		State   bool   `json:"state"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.State {
		return fmt.Errorf("refresh failed: %s", result.Message)
	}

	c.config.SetAccessToken(result.Data.AccessToken, result.Data.ExpiresIn)
	if result.Data.RefreshToken != "" {
		c.config.SetRefreshToken(result.Data.RefreshToken)
	}
	c.config.SaveToken()
	return nil
}

func (c *Client) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doFormRequest(method, path string, data url.Values) (map[string]interface{}, error) {
	resp, err := c.doRequest(method, path, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) doQueryRequest(method, path string, params url.Values) (map[string]interface{}, error) {
	if params != nil {
		path = path + "?" + params.Encode()
	}
	resp, err := c.doRequest(method, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
