package tg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const telegramAPI = "https://api.telegram.org/bot"

type UserProfilePhotos struct {
	TotalCount int           `json:"total_count"`
	Photos     [][]PhotoSize `json:"photos"`
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type APIResponse struct {
	Ok     bool              `json:"ok"`
	Result UserProfilePhotos `json:"result"`
}

type FileResponse struct {
	OK     bool `json:"ok"`
	Result File `json:"result"`
}

type File struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
}

func GetUserProfilePhotos(userID int64) (*UserProfilePhotos, error) {
	url := fmt.Sprintf("%s/getUserProfilePhotos?user_id=%d", telegramAPI+os.Getenv("TMA_TOKEN"), userID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 响应失败，状态码: %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if !apiResp.Ok {
		return nil, fmt.Errorf("API 响应失败: %v", apiResp)
	}

	return &apiResp.Result, nil
}

func GetFileURL(fileID string) (string, string, error) {
	url := fmt.Sprintf("%s/getFile?file_id=%s", telegramAPI+os.Getenv("TMA_TOKEN"), fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API 响应失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("解析响应失败: %v", err)
	}

	if !result.Ok {
		return "", "", fmt.Errorf("无法获取文件，错误: %v", result)
	}

	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TMA_TOKEN"), result.Result.FilePath), getFileType(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TMA_TOKEN"), result.Result.FilePath)), nil
}

func getFileType(url string) string {
	// 从 URL 中提取文件名
	parts := strings.Split(url, "/")
	fileName := parts[len(parts)-1] // 获取最后一部分

	// 获取文件扩展名
	ext := filepath.Ext(fileName)

	return ext
}
