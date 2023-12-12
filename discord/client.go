package discord

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"time"
)

type Client struct {
	apiToken          string
	session           *discordgo.Session
	streamBatchWaitMs int64
}

func NewClient(streamBatchWaitMs int64) (*Client, error) {
	// Get API token from environment variable, return error if it's not set
	apiToken := os.Getenv("DISCORD_BOT_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("DISCORD_API_TOKEN environment variable not set")
	}

	session, err := discordgo.New("Bot " + apiToken)
	if err != nil {
		return nil, err
	}

	return &Client{session: session, apiToken: apiToken, streamBatchWaitMs: streamBatchWaitMs}, nil
}

func (c *Client) SendMessage(message string, channelID string) error {
	// set typing
	if err := c.session.ChannelTyping(channelID); err != nil {
		return err
	}
	defer func(session *discordgo.Session, channelID string, options ...discordgo.RequestOption) {
		_ = session.ChannelTyping(channelID)
	}(c.session, channelID)

	_, err := c.session.ChannelMessageSend(channelID, message)
	return err
}

func (c *Client) StreamMessage(chunks chan string, channelID string) error {
	// set typing
	if err := c.session.ChannelTyping(channelID); err != nil {
		return err
	}
	defer func(session *discordgo.Session, channelID string, options ...discordgo.RequestOption) {
		_ = session.ChannelTyping(channelID)
	}(c.session, channelID)

	// grab the first non-empty string off the channel to create the initial message
	msgText := ""
	for msg := range chunks {
		if msg != "" {
			msgText = msg
			break
		}
	}

	msg, err := c.session.ChannelMessageSend(channelID, msgText)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(c.streamBatchWaitMs) * time.Millisecond)
	buff := ""
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				if buff != "" {
					if len(msgText)+len(buff) > 2000 {
						if _, err := c.session.ChannelMessageSend(channelID, buff); err != nil {
							return err
						} else {
							if _, err := c.session.ChannelMessageEdit(channelID, msg.ID, msgText+buff); err != nil {
								return err
							}
						}
					}
				}
				return nil
			}
			buff += chunk
		case <-ticker.C:
			if len(msgText)+len(buff) > 2000 {
				msgText = buff
				msg, err = c.session.ChannelMessageSend(channelID, msgText)
				if err != nil {
					return err
				}
			} else {
				msgText += buff
				if _, err := c.session.ChannelMessageEdit(channelID, msg.ID, msgText); err != nil {
					return err
				}
			}
			buff = ""
		}
	}
}
