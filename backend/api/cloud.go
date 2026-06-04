package api

import (
	"fmt"
	"net/url"
)

func (c *Client) AddDownloadTask(urls string, pathID string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("urls", urls)
	if pathID != "" {
		data.Set("wp_path_id", pathID)
	}

	result, err := c.doFormRequest("POST", BaseURL+"/open/offline/add_task_urls", data)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("add download task failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) GetDownloadTaskList(page int) (map[string]interface{}, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}

	result, err := c.doQueryRequest("GET", BaseURL+"/open/offline/get_task_list", params)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("get task list failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) DeleteDownloadTask(infoHash string, delSource bool) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("info_hash", infoHash)
	if delSource {
		data.Set("del_source_file", "1")
	} else {
		data.Set("del_source_file", "0")
	}

	result, err := c.doFormRequest("POST", BaseURL+"/open/offline/del_task", data)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("delete task failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) ClearDownloadTasks(flag int) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("flag", fmt.Sprintf("%d", flag))

	result, err := c.doFormRequest("POST", BaseURL+"/open/offline/clear_task", data)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("clear tasks failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) GetDownloadQuota() (map[string]interface{}, error) {
	result, err := c.doQueryRequest("GET", BaseURL+"/open/offline/get_quota_info", nil)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("get quota failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) ParseTorrent(sha1, pickCode string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("torrent_sha1", sha1)
	data.Set("pick_code", pickCode)

	result, err := c.doFormRequest("POST", BaseURL+"/open/offline/torrent", data)
	if err != nil {
		return nil, err
	}
	if state, ok := result["state"].(bool); !ok || !state {
		return nil, fmt.Errorf("parse torrent failed: %v", result["message"])
	}
	return result, nil
}
