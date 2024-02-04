package bot

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"github.com/mdesson/chatcord/discord"
	"github.com/mdesson/chatcord/logger"
	"github.com/mdesson/chatcord/openai"
	"github.com/mdesson/chatcord/util"
	"log/slog"
	"time"
)

type conversation struct {
	ChannelID string
	*openai.Conversation
}

type Bot struct {
	conversations []*conversation
	discordClient *discord.Client
	openAIClient  *openai.Client
	db            *sql.DB
	l             *logger.Logger
}

// New creates a new bot, which has access to a discord client and an OpenAI client
func New(logLevel slog.Level) (*Bot, error) {
	discordClient, err := discord.NewClient(500)
	if err != nil {
		return nil, err
	}

	openAIClient, err := openai.NewClient()
	if err != nil {
		return nil, err
	}

	db, err := initDB()
	if err != nil {
		return nil, err
	}

	b := &Bot{conversations: make([]*conversation, 0), discordClient: discordClient, openAIClient: openAIClient, db: db, l: logger.New(logLevel)}

	conversations, err := selectAllConversations(*b)
	if err != nil {
		return nil, err
	}
	for _, c := range conversations {
		c := c
		b.conversations = append(b.conversations, &c)
	}

	return b, nil
}

// Start registers discord handlers and then starts the discord session
func (b *Bot) Start() error {
	b.l.Info("starting bot")

	// Handler listening for channel creation
	b.discordClient.Session.AddHandler(func(s *discordgo.Session, event *discordgo.ChannelCreate) {
		// TODO: Use a system default prompt.
		c := conversation{
			ChannelID:    event.Channel.ID,
			Conversation: openai.NewConversation(openai.GPT_4_TURBO, "You are a helpful assistant. If you need to use formatting, send it with discord-flavoured markdown.", b.openAIClient),
		}

		if err := createConvoMessageUsage(*b, c); err != nil {
			b.l.Error(err.Error(), "handler", "channel_create")
			return
		}

		b.conversations = append(b.conversations, &c)
	})

	// Handler to respond to messages in conversations
	b.discordClient.Session.AddHandler(func(s *discordgo.Session, event *discordgo.MessageCreate) {
		// Ignore messages sent by the bot
		if event.Author.Bot {
			return
		}

		for _, c := range b.conversations {
			done := make(chan bool)
			if event.ChannelID == c.ChannelID {
				// Set typing while openAI processes API request
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
				break
			}
		}
	})

	// Open the session, it is now listening for events
	if err := b.discordClient.Session.Open(); err != nil {
		return err
	}

	// TODO: Swap to user-friendly init message
	if err := b.discordClient.SendMessage("online", b.discordClient.GeneralChannel); err != nil {
		return err
	}

	b.l.Info("bot init complete")

	return nil
}

func (b *Bot) Stop() {
	b.l.Info("stopping")

	if err := b.db.Close(); err != nil {
		b.l.Error(err.Error())
	}
	if err := b.discordClient.Session.Close(); err != nil {
		b.l.Error(err.Error())
	}

	b.l.Info("stopped")
}
