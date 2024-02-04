package bot

import (
	"database/sql"
	"github.com/mdesson/chatcord/discord"
	"github.com/mdesson/chatcord/logger"
	"github.com/mdesson/chatcord/openai"
	"log/slog"
)

type conversation struct {
	ChannelID string
	*openai.Conversation
}

type Bot struct {
	conversations map[string]*conversation
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

	b := &Bot{conversations: make(map[string]*conversation), discordClient: discordClient, openAIClient: openAIClient, db: db, l: logger.New(logLevel)}

	conversations, err := selectAllConversations(*b)
	if err != nil {
		return nil, err
	}
	for _, c := range conversations {
		c := c
		b.conversations[c.ChannelID] = &c
	}

	return b, nil
}

// Start registers discord handlers and then starts the discord session
func (b *Bot) Start() error {
	b.l.Info("starting bot")

	// Handler listening for channel creation
	b.discordClient.Session.AddHandler(MakeChannelCreateHandler(b))

	// Handler to respond to messages in conversations
	b.discordClient.Session.AddHandler(MakeMessageCreateHandler(b))

	// Handler clear out db on channel delete
	b.discordClient.Session.AddHandler(MakeChannelDeleteHandler(b))

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
