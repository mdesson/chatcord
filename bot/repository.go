package bot

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdesson/chatcord/openai"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "conversation.db")
	if err != nil {
		return nil, err
	}

	// Adjusting the table schema. Make channel_id the primary STRING key for conversations,
	// and adjust foreign keys in messages and usages to reference channel_id.
	createStmt := `
    CREATE TABLE IF NOT EXISTS conversations (
        channel_id TEXT PRIMARY KEY,
        name TEXT,
        model TEXT,
        temperature REAL,
        total_choices INTEGER,
        system_prompt TEXT
    );
    CREATE TABLE IF NOT EXISTS messages (
        channel_id TEXT,
        role TEXT,
        content TEXT,
        FOREIGN KEY (channel_id) REFERENCES conversations (channel_id)
    );
    CREATE TABLE IF NOT EXISTS usages (
        channel_id TEXT PRIMARY KEY,
        completion_tokens INTEGER,
        prompt_tokens INTEGER,
        total_tokens INTEGER,
        FOREIGN KEY (channel_id) REFERENCES conversations (channel_id)
    );`

	_, err = db.Exec(createStmt)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func insertConversation(b Bot, conv conversation) (err error) {
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}

	// Insert into conversations table
	if _, err := tx.Exec(
		"INSERT INTO conversations(channel_id, name, model, temperature, total_choices, system_prompt) VALUES(?, ?, ?, ?, ?, ?)",
		conv.ChannelID, conv.Name, conv.Model, conv.Temperature, conv.TotalChoices, conv.SystemPrompt,
	); err != nil {
		tx.Rollback()
		return err
	}

	// Insert into messages table
	for _, msg := range conv.Messages {
		_, err = tx.Exec(
			"INSERT INTO messages(channel_id, role, content) VALUES(?, ?, ?)",
			conv.ChannelID, msg.Role, msg.Content,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Insert into usages table
	_, err = tx.Exec(
		"INSERT INTO usages(channel_id, completion_tokens, prompt_tokens, total_tokens) VALUES(?, ?, ?, ?)",
		conv.ChannelID, conv.Usage.CompletionTokens, conv.Usage.PromptTokens, conv.Usage.TotalTokens,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}

func selectAllConversations(b Bot) ([]conversation, error) {
	convoRes, err := b.db.Query(`SELECT * FROM conversations`)
	if err != nil {
		return nil, err
	}

	convos := make([]conversation, 0)
	for convoRes.Next() {
		convo := conversation{Conversation: &openai.Conversation{}}
		if err := convoRes.Scan(&convo.ChannelID, &convo.Name, &convo.Model, &convo.Temperature, &convo.TotalChoices, &convo.SystemPrompt); err != nil {
			return nil, err
		}

		convo.Init(b.openAIClient)

		msgs, err := selectMessagesByChannelID(b, convo.ChannelID)
		if err != nil {
			return nil, err
		}
		convo.Messages = msgs

		usage, err := selectUsageByChannelID(b, convo.ChannelID)
		if err != nil {
			return nil, err
		}
		convo.Usage = usage

		convos = append(convos, convo)

	}

	if err := convoRes.Err(); err != nil {
		return nil, err
	}

	return convos, nil
}

func insertMessage(b Bot, channelID string, m openai.Message) error {
	if _, err := b.db.Exec(`INSERT INTO messages(channel_id, role, content) VALUES (?, ?, ?)`, channelID, m.Role, m.Content); err != nil {
		return err
	}
	return nil
}

func selectMessagesByChannelID(b Bot, channelID string) ([]openai.Message, error) {
	msgRes, err := b.db.Query(`SELECT * FROM messages WHERE channel_id = ?`, channelID)
	if err != nil {
		return nil, err
	}

	msgs := make([]openai.Message, 0)
	for msgRes.Next() {
		msg := struct {
			ChannelID string
			openai.Message
		}{}

		if err := msgRes.Scan(&msg.ChannelID, &msg.Role, &msg.Content); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg.Message)
	}
	if err := msgRes.Err(); err != nil {
		return nil, err
	}

	return msgs, nil
}

func updateUsage(b Bot, channelID string, u openai.Usage) error {
	if _, err := b.db.Exec(`UPDATE usages
    SET completion_tokens = ?, prompt_tokens = ?, total_tokens = ?
    WHERE channel_id = ?`, u.CompletionTokens, u.PromptTokens, u.TotalTokens, channelID); err != nil {
		return err
	}
	return nil
}

func selectUsageByChannelID(b Bot, channelID string) (openai.Usage, error) {
	usage := struct {
		ChannelID string
		openai.Usage
	}{}
	if err := b.db.QueryRow(`SELECT * FROM usages WHERE channel_id = ?`, channelID).Scan(&usage.ChannelID, &usage.CompletionTokens, &usage.PromptTokens, &usage.TotalTokens); err != nil {
		return openai.Usage{}, err
	}

	return usage.Usage, nil
}
