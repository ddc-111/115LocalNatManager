package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetUserInfo() (map[string]interface{}, error) {
	result, err := c.doQueryRequest("GET", BaseURL+"/open/user/info", nil)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("get user info failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) GetFileList(cid string, limit, offset int) (map[string]interface{}, error) {
	params := url.Values{}
	if cid != "" {
		params.Set("cid", cid)
	}
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", fmt.Sprintf("%d", offset))

	result, err := c.doQueryRequest("GET", BaseURL+"/open/ufile/files", params)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("get file list failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) GetFileInfo(fileID string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("file_id", fileID)

	result, err := c.doQueryRequest("GET", BaseURL+"/open/folder/get_info", params)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("get file info failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) SearchFiles(keyword string, limit, offset int) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("search_value", keyword)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", fmt.Sprintf("%d", offset))

	result, err := c.doQueryRequest("GET", BaseURL+"/open/ufile/search", params)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("search files failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) CreateFolder(pid, name string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("pid", pid)
	data.Set("file_name", name)

	result, err := c.doFormRequest("POST", BaseURL+"/open/folder/add", data)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("create folder failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) DeleteFiles(fileIDs string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("file_ids", fileIDs)

	result, err := c.doFormRequest("POST", BaseURL+"/open/ufile/delete", data)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("delete files failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) RenameFile(fileID, newName string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("file_id", fileID)
	data.Set("file_name", newName)

	result, err := c.doFormRequest("POST", BaseURL+"/open/ufile/update", data)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("rename file failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) MoveFiles(fileIDs, toCID string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("file_ids", fileIDs)
	data.Set("to_cid", toCID)

	result, err := c.doFormRequest("POST", BaseURL+"/open/ufile/move", data)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("move files failed: %v", result["message"])
	}
	return result, nil
}

func (c *Client) GetDownloadURL(pickCode string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("pick_code", pickCode)

	result, err := c.doFormRequest("POST", BaseURL+"/open/ufile/downurl", data)
	if err != nil {
		return nil, err
	}
	if !parseState(result["state"]) {
		return nil, fmt.Errorf("get download url failed: %v", result["message"])
	}
	return result, nil
}
