package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/mdesson/chatcord/openai"
	"github.com/mdesson/chatcord/util"
	"time"
)

func MakeChannelCreateHandler(b *Bot) func(s *discordgo.Session, event *discordgo.ChannelCreate) {
	return func(s *discordgo.Session, event *discordgo.ChannelCreate) {
		b.l.Debug("called", "channel_id", event.ID, "handler", "channel_create")

		// TODO: Use a system default prompt.
		c := conversation{
			ChannelID:    event.Channel.ID,
			Conversation: openai.NewConversation(openai.GPT_4_TURBO, "You are a helpful assistant. If you need to use formatting, send it with discord-flavoured markdown.", b.openAIClient),
		}

		if err := createConvoMessageUsage(*b, c); err != nil {
			b.l.Error(err.Error(), "handler", "channel_create")
			return
		}

		b.conversations[event.ID] = &c
	}
}

func MakeMessageCreateHandler(b *Bot) func(s *discordgo.Session, event *discordgo.MessageCreate) {
	return func(s *discordgo.Session, event *discordgo.MessageCreate) {
		b.l.Debug("called", "channel_id", event, "handler", "message_create")
		
		// Ignore messages sent by the bot
		if event.Author.Bot {
			return
		}

		c, ok := b.conversations[event.ChannelID]

		// ignore conversations we are not watching
		if !ok {
			return
		}

		// Set typing while openAI processes API request
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					if err := b.discordClient.Session.ChannelTyping(event.ChannelID); err != nil {
						b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
						return
					}
					time.Sleep(3 * time.Second)
				}
			}
		}()

		if msg, err := c.Chat(event.Content); err != nil {
			done <- true
			b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
		} else {
			done <- true
			for _, chunk := range util.ChunkText(msg) {
				if err := b.discordClient.SendMessage(chunk, event.ChannelID); err != nil {
					b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
				}
			}
			// After successfully sending message, update db with user message and bot response
			// TODO: This should really be done as a transaction.
			userMsg := c.Messages[len(c.Messages)-2]
			botMsg := c.Messages[len(c.Messages)-1]

			if err := insertMessage(*b, event.ChannelID, userMsg); err != nil {
				b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
			}

			if err := insertMessage(*b, event.ChannelID, botMsg); err != nil {
				b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
			}

			if err := updateUsage(*b, event.ChannelID, c.Usage); err != nil {
				b.l.Error(err.Error(), "handler", "message_create", "channel_id", event.ChannelID)
			}
		}
	}
}

func MakeChannelDeleteHandler(b *Bot) func(s *discordgo.Session, event *discordgo.ChannelDelete) {
	return func(s *discordgo.Session, event *discordgo.ChannelDelete) {
		b.l.Debug("called", "channel_id", event, "handler", "channel_delete")

		// Remove from database
		if err := deleteConvoMessageUsage(*b, event.ID); err != nil {
			b.l.Error(err.Error(), "handler", "channel_delete", "channel_id", event.ID)
		} else {
			// On success, remove from memory
			delete(b.conversations, event.ID)
		}

	}
}
