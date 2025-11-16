package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка вызова Groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq API вернул статус %d", resp.StatusCode)
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("ошибка декодирования Groq ответа: %w", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("Groq API вернул ошибку: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) > 0 && groqResp.Choices[0].Message.Content != "" {
		return groqResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("Groq API не вернул ответ (пустое 'choices')")
}
