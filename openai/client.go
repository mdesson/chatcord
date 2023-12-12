package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	apiToken string
}

func NewClient() (*Client, error) {
	// Get API token from environment variable, return error if it's not set
	apiToken := os.Getenv("OPENAI_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("OPENAI_API_TOKEN environment variable not set")
	}
	return &Client{apiToken: apiToken}, nil
}

func (c *Client) sendChat(request ChatRequest) (ChatResponse, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return ChatResponse{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/chat/completions", base_url), bytes.NewBuffer(reqBody))
	if err != nil {
		return ChatResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ChatResponse{}, err
	}

	if request.Stream {
		return ChatResponse{HTTPBody: resp.Body}, nil
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var chatResponse ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return ChatResponse{}, err
	}
	chatResponse.HTTPBody = resp.Body

	return chatResponse, nil
}
