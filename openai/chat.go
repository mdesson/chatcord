package openai

import (
	"io"
)

type Message struct {
	Index   int    `json:"-"`
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatRequest struct {
	Model        Model     `json:"model"`
	Messages     []Message `json:"messages"`
	Temperature  *float64  `json:"temperature,omitempty"` // Between 0 and 2
	Stream       bool      `json:"stream,omitempty"`
	TotalChoices int       `json:"n,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
		Message      Message `json:"message"`
	} `json:"choices"`
	Created  int    `json:"created"`
	ID       string `json:"id"`
	Model    Model  `json:"model"`
	Object   string `json:"object"`
	Usage    Usage  `json:"usage"`
	HTTPBody io.ReadCloser
}

type Chunk struct {
	Id                string `json:"id"`
	Object            string `json:"object"`
	Created           int    `json:"created"`
	Model             Model  `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Choices           []struct {
		Delta        *Message `json:"delta"`
		FinishReason *string  `json:"finish_reason,omitempty"`
	} `json:"choices"`
}
