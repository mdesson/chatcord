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
	Usage        Usage // Will be zeroed out for streaming conversations, as it is not returned, TODO: Is there an OpenAI endpoint to count tokens in a conversation? Do it myself?
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

func (c *Conversation) ChatStream(message string) (chan string, chan error, error) {
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
		return nil, nil, err
	}

	chunks := make(chan string)
	errChan := make(chan error)
	newMessage := Message{Role: ROLE_ASSISTANT, Content: ""}

	go func() {
		defer func() {
			_ = chatResponse.HTTPBody.Close()
			close(chunks)
			close(errChan)
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
				errChan <- err
				break
			}

			if chunk.Choices[0].FinishReason != nil {
				break
			}

			text := chunk.Choices[0].Delta.Content
			c.Messages[len(c.Messages)-1].Content += text
			chunks <- text
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}

		c.Messages = append(c.Messages, newMessage)
	}()

	return chunks, errChan, nil
}
