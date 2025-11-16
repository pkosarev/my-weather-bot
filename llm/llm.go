package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

type GroqRequest struct {
	Messages []GroqMessage `json:"messages"`
	Model    string        `json:"model"`
}
type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type GroqResponse struct {
	Choices []struct {
		Message GroqMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Client struct {
	apiKey string
	model  string
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  "groq/compound",
	}
}

func (c *Client) GetAnalysis(systemPrompt, userPrompt string) (string, error) {
	reqBody := GroqRequest{
		Model: c.model,
		Messages: []GroqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка маршалинга Groq запроса: %w", err)
	}

	req, err := http.NewRequestWithContext(context.TODO(), "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания Groq запроса: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}

	proxyURLStr := os.Getenv("PROXY_URL")

	if proxyURLStr != "" {
		log.Println("Обнаружен PROXY_URL, используем прокси...")

		proxyURL, err := url.Parse(proxyURLStr)
		if err != nil {
			return "", fmt.Errorf("не удалось распарсить PROXY_URL: %w", err)
		}

		var transport http.RoundTripper
		if proxyURL.Scheme == "socks5" {
			dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				return "", fmt.Errorf("не удалось создать SOCKS5 dialer: %w", err)
			}
			transport = &http.Transport{Dial: dialer.Dial}
		} else {
			transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}

		client.Transport = transport

	} else {
		log.Println("PROXY_URL не найден, идем напрямую...")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка вызова Groq API (через прокси?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Groq API (через прокси?) вернул статус %d", resp.StatusCode)
		return "", fmt.Errorf("groq API вернул статус %d", resp.StatusCode)
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("ошибка декодирования Groq ответа: %w", err)
	}
	if groqResp.Error != nil {
		return "", fmt.Errorf("groq API вернул ошибку: %s", groqResp.Error.Message)
	}
	if len(groqResp.Choices) > 0 && groqResp.Choices[0].Message.Content != "" {
		return groqResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("groq API не вернул ответ (пустое 'choices')")
}
