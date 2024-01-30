package discord

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"time"
)

type Client struct {
	apiToken          string
	Session           *discordgo.Session
	streamBatchWaitMs int64
	GeneralChannel    string
}

func NewClient(streamBatchWaitMs int64) (*Client, error) {
	// Get API token from environment variable, return error if it's not set
	apiToken := os.Getenv("DISCORD_BOT_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("DISCORD_API_TOKEN environment variable not set")
	}

	// TODO: Find general general channel on start, add env var to optionally pass name or id for it
	generalChannel := os.Getenv("GENERAL_CHANNEL_ID")
	if generalChannel == "" {
		return nil, fmt.Errorf("GENERAL_CHANNEL_ID environment variable not set")
	}

	session, err := discordgo.New("Bot " + apiToken)
	if err != nil {
		return nil, err
	}

	return &Client{Session: session, apiToken: apiToken, streamBatchWaitMs: streamBatchWaitMs, GeneralChannel: generalChannel}, nil
}

func (c *Client) SendMessage(message string, channelID string) error {
	// set typing
	if err := c.Session.ChannelTyping(channelID); err != nil {
		return err
	}

	_, err := c.Session.ChannelMessageSend(channelID, message)
	return err
}

func (c *Client) StreamMessage(chunks chan string, channelID string) error {
	// set typing
	if err := c.Session.ChannelTyping(channelID); err != nil {
		return err
	}
	defer func(session *discordgo.Session, channelID string, options ...discordgo.RequestOption) {
		_ = session.ChannelTyping(channelID)
	}(c.Session, channelID)

	// grab the first non-empty string off the channel to create the initial message
	msgText := ""
	for msg := range chunks {
		if msg != "" {
			msgText = msg
			break
		}
	}

	msg, err := c.Session.ChannelMessageSend(channelID, msgText)
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
						if _, err := c.Session.ChannelMessageSend(channelID, buff); err != nil {
							return err
						} else {
							if _, err := c.Session.ChannelMessageEdit(channelID, msg.ID, msgText+buff); err != nil {
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
				msg, err = c.Session.ChannelMessageSend(channelID, msgText)
				if err != nil {
					return err
				}
			} else {
				msgText += buff
				if _, err := c.Session.ChannelMessageEdit(channelID, msg.ID, msgText); err != nil {
					return err
				}
			}
			buff = ""
		}
	}
}
