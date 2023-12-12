package openai

import (
	"bufio"
	"encoding/json"
	"strings"
	"sync"
)

type Conversation struct {
	Name         string
	Model        Model
	Messages     []Message
	Temperature  *float64
	TotalChoices int
	SystemPrompt string
	Usage        Usage // Will be zeroed out for streaming conversations
	client       *Client
	mu           sync.Mutex
}

func NewConversation(model Model, systemPrompt string, client *Client) *Conversation {
	return &Conversation{
		Name:  "temp", // TODO: Conversation name generation
		Model: model,
		Messages: []Message{
			{Role: ROLE_SYSTEM, Content: systemPrompt},
		},
		Temperature:  nil,
		TotalChoices: 1,
		SystemPrompt: systemPrompt,
		Usage: Usage{
			CompletionTokens: 0,
			PromptTokens:     0,
			TotalTokens:      0,
		},
		client: client,
	}
}

func (c *Conversation) UpdatePrompt(prompt string) {
	c.Messages[0].Content = prompt
	c.SystemPrompt = prompt
}

// Chat send a message to the OpenAPI backend and get the entire response in a single message.
func (c *Conversation) Chat(message string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Messages = append(c.Messages, Message{Role: ROLE_USER, Content: message})
	chatResponse, err := c.client.sendChat(ChatRequest{
		Model:        c.Model,
		Messages:     c.Messages,
		Temperature:  c.Temperature,
		Stream:       false,
		TotalChoices: c.TotalChoices,
	})
	if err != nil {
		return "", err
	}

	c.Messages = append(c.Messages, chatResponse.Choices[0].Message)
	c.Usage = chatResponse.Usage
	return chatResponse.Choices[0].Message.Content, nil
}

func (c *Conversation) ChatStream(message string) (chan string, error) {
	c.mu.Lock()

	c.Messages = append(c.Messages, Message{Role: ROLE_USER, Content: message})
	chatResponse, err := c.client.sendChat(ChatRequest{
		Model:        c.Model,
		Messages:     c.Messages,
		Temperature:  c.Temperature,
		Stream:       true,
		TotalChoices: c.TotalChoices,
	})
	if err != nil {
		return nil, err
	}

	chunks := make(chan string)
	newMessage := Message{Role: ROLE_ASSISTANT, Content: ""}

	go func() {
		defer func() {
			_ = chatResponse.HTTPBody.Close()
			close(chunks)
			c.mu.Unlock()
		}()

		scanner := bufio.NewScanner(chatResponse.HTTPBody)
		for scanner.Scan() {
			newText := scanner.Text()
			newMessage.Content += newText
			rawChunk := strings.TrimPrefix(newText, "data: ")

			if rawChunk == "" {
				continue
			}

			var chunk Chunk
			if err := json.Unmarshal([]byte(rawChunk), &chunk); err != nil {
				// TODO: Gracefully handle error once logging is set up
				chunks <- err.Error()
				break
			}

			if chunk.Choices[0].FinishReason != nil {
				break
			}

			text := chunk.Choices[0].Delta.Content
			c.Messages[len(c.Messages)-1].Content += text
			chunks <- text
		}

		// TODO: Log errors once proper logging is set up
		if err := scanner.Err(); err != nil {
			chunks <- err.Error()
		}

		c.Messages = append(c.Messages, newMessage)
	}()

	return chunks, nil
}
