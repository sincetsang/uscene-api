package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"TMA/pkg/nucl"
	"TMA/pkg/sdlog"
	"TMA/service/db/model"
)

const (
	openaiAPI      = "https://api.openai.com/v1/chat/completions"
	maxRetries     = 3
	requestTimeout = 30 * time.Second
)

type openaiRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// DetectLanguage 检测文本语言
func DetectLanguage(openaiAPIKey, text string) (string, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		reqBody, err := json.Marshal(openaiRequest{
			Model: "gpt-3.5-turbo",
			Messages: []message{
				{
					Role:    "system",
					Content: "你是一个语言检测专家。请检测以下文本的语言，只返回语言代码（如：zh、en、ja、ko等）。不要添加任何解释或标点符号。",
				},
				{
					Role:    "user",
					Content: text,
				},
			},
			Stream: false,
		})
		if err != nil {
			return "", fmt.Errorf("marshal request failed: %v", err)
		}

		client := &http.Client{Timeout: requestTimeout}
		req, err := http.NewRequest("POST", openaiAPI, bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("create request failed: %v", err)
			sdlog.Warnf("检测语言第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("detect request failed: %v", err)
			sdlog.Warnf("检测语言第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("detect request failed with status %d: %s", resp.StatusCode, string(body))
			sdlog.Warnf("检测语言第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		var result openaiResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			lastErr = fmt.Errorf("decode response failed: %v", err)
			sdlog.Warnf("检测语言第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		if len(result.Choices) == 0 {
			lastErr = fmt.Errorf("no language detected")
			sdlog.Warnf("检测语言第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("after %d retries: %v", maxRetries, lastErr)
}

// TranslateText 翻译文本
func TranslateText(apiKey, text, source, target string) (string, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		reqBody, err := json.Marshal(openaiRequest{
			Model: "gpt-3.5-turbo",
			Messages: []message{
				{
					Role:    "system",
					Content: fmt.Sprintf("你是一个翻译专家。请将以下%s文本翻译成%s，只返回翻译结果，不要添加任何解释或标点符号。保持原文的格式和风格。", source, target),
				},
				{
					Role:    "user",
					Content: text,
				},
			},
			Stream: false,
		})
		if err != nil {
			return "", fmt.Errorf("marshal request failed: %v", err)
		}

		client := &http.Client{Timeout: requestTimeout}
		req, err := http.NewRequest("POST", openaiAPI, bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("create request failed: %v", err)
			sdlog.Warnf("翻译文本第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("translate request failed: %v", err)
			sdlog.Warnf("翻译文本第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("translate request failed with status %d: %s", resp.StatusCode, string(body))
			sdlog.Warnf("翻译文本第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		var result openaiResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			lastErr = fmt.Errorf("decode response failed: %v", err)
			sdlog.Warnf("翻译文本第 %d 次尝试失败: %v", i+1, err)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		if len(result.Choices) == 0 {
			lastErr = fmt.Errorf("no translation result")
			sdlog.Warnf("翻译文本第 %d 次尝试失败: %v", i+1, lastErr)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("after %d retries: %v", maxRetries, lastErr)
}

// TranslateSpotContent 异步翻译地标内容
func TranslateSpotContent(apiKey string, spotID int64, title, description string) {
	// 创建新的上下文，增加超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 使用全局数据库连接
	db := nucl.GetGlobalDB().WithContext(ctx)

	// 检测标题语言
	titleLang, err := DetectLanguage(apiKey, title)
	if err != nil {
		sdlog.Errorf("[TranslateSpotContent] 检测标题语言失败: %v", err)
		return
	}

	// 检测描述语言
	descLang, err := DetectLanguage(apiKey, description)
	if err != nil {
		sdlog.Errorf("[TranslateSpotContent] 检测描述语言失败: %v", err)
		return
	}

	// 根据检测到的语言进行翻译
	var titleEn, titleZh, descriptionEn, descriptionZh string

	// 处理标题翻译
	if titleLang == "en" {
		// 如果是英文，只翻译为中文
		titleZh, err = TranslateText(apiKey, title, "en", "zh")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译标题为中文失败: %v", err)
			return
		}
		titleEn = title
	} else if titleLang == "zh" {
		// 如果是中文，只翻译为英文
		titleEn, err = TranslateText(apiKey, title, "zh", "en")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译标题为英文失败: %v", err)
			return
		}
		titleZh = title
	} else {
		// 如果是其他语言，同时翻译为中文和英文
		titleEn, err = TranslateText(apiKey, title, titleLang, "en")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译标题为英文失败: %v", err)
			return
		}
		titleZh, err = TranslateText(apiKey, title, titleLang, "zh")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译标题为中文失败: %v", err)
			return
		}
	}

	// 处理描述翻译
	if descLang == "en" {
		// 如果是英文，只翻译为中文
		descriptionZh, err = TranslateText(apiKey, description, "en", "zh")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译描述为中文失败: %v", err)
			return
		}
		descriptionEn = description
	} else if descLang == "zh" {
		// 如果是中文，只翻译为英文
		descriptionEn, err = TranslateText(apiKey, description, "zh", "en")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译描述为英文失败: %v", err)
			return
		}
		descriptionZh = description
	} else {
		// 如果是其他语言，同时翻译为中文和英文
		descriptionEn, err = TranslateText(apiKey, description, descLang, "en")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译描述为英文失败: %v", err)
			return
		}
		descriptionZh, err = TranslateText(apiKey, description, descLang, "zh")
		if err != nil {
			sdlog.Errorf("[TranslateSpotContent] 翻译描述为中文失败: %v", err)
			return
		}
	}

	// 更新数据库
	err = db.Model(&model.CheckInPoint{}).
		Where("id = ?", spotID).
		Updates(map[string]interface{}{
			"title_en":       titleEn,
			"title_cn":       titleZh,
			"description_en": descriptionEn,
			"description_cn": descriptionZh,
			"updated_at":     time.Now(),
		}).Error
	if err != nil {
		sdlog.Errorf("[TranslateSpotContent] 更新翻译结果失败: %v", err)
		return
	}
	sdlog.Infof("[TranslateSpotContent] 地标翻译完成: spot_id=%d", spotID)
}
