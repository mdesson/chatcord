package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/mdesson/chatcord/discord"
	"github.com/mdesson/chatcord/logger"
	"github.com/mdesson/chatcord/openai"
	"github.com/mdesson/chatcord/util"
	"log/slog"
	"time"
)

type conversation struct {
	channelID   string
	openAIConvo *openai.Conversation
}

type Bot struct {
	conversations []*conversation
	discordClient *discord.Client
	openAIClient  *openai.Client
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

	return &Bot{conversations: make([]*conversation, 0), discordClient: discordClient, openAIClient: openAIClient, l: logger.New(logLevel)}, nil
}

// Start registers discord handlers and then starts the discord session
func (b *Bot) Start() error {
	b.l.Info("starting bot")

	// Handler listening for channel creation
	b.discordClient.Session.AddHandler(func(s *discordgo.Session, event *discordgo.ChannelCreate) {
		// TODO: Use a system default prompt.
		c := conversation{
			channelID:   event.Channel.ID,
			openAIConvo: openai.NewConversation(openai.GPT_4_TURBO, "You are a helpful assistant. If you need to use formatting, send it with discord-flavoured markdown.", b.openAIClient),
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
			done := make(chan bool, 0)
			if event.ChannelID == c.channelID {
				// Set typing while openAI processes API request
				go func() {
					for {
						select {
						case <-done:
							return
						default:
							if err := b.discordClient.Session.ChannelTyping(event.ChannelID); err != nil {
								b.l.Error(err.Error(), "channel_id", event.ChannelID)
								return
							}
							time.Sleep(3 * time.Second)
						}
					}
				}()
				defer func() { done <- true }()

				if msg, err := c.openAIConvo.Chat(event.Content); err != nil {
					b.l.Error(err.Error(), "channel_id", event.ChannelID)
				} else {
					for _, chunk := range util.ChunkText(msg) {
						if err := b.discordClient.SendMessage(chunk, event.ChannelID); err != nil {
							b.l.Error(err.Error(), "channel_id", event.ChannelID)
						}
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
