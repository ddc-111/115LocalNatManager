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

func parseState(state interface{}) bool {
	switch v := state.(type) {
	case bool:
		return v
	case float64:
		return v == 1
	case int:
		return v == 1
	default:
		return false
	}
}

func parseAPIError(result map[string]interface{}) error {
	code, _ := result["code"].(float64)
	msg, _ := result["message"].(string)
	errMsg, _ := result["error"].(string)
	
	return &APIError{
		Code:    int(code),
		Message: msg,
		Error:   errMsg,
	}
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

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !parseState(result["state"]) {
		code, _ := result["code"].(float64)
		msg, _ := result["message"].(string)
		errMsg, _ := result["error"].(string)
		
		apiErr := &APIError{
			Code:    int(code),
			Message: msg,
			Error:   errMsg,
		}
		return apiErr
	}

	dataMap, _ := result["data"].(map[string]interface{})
	if dataMap == nil {
		return fmt.Errorf("invalid response data")
	}

	accessToken, _ := dataMap["access_token"].(string)
	refreshToken, _ := dataMap["refresh_token"].(string)
	expiresIn, _ := dataMap["expires_in"].(float64)

	if accessToken == "" {
		return fmt.Errorf("no access_token in response")
	}

	c.config.SetAccessToken(accessToken, int(expiresIn))
	if refreshToken != "" {
		c.config.SetRefreshToken(refreshToken)
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
