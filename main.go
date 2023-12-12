package main

import (
	"github.com/mdesson/chatcord/discord"
	openai "github.com/mdesson/chatcord/openai"
)

func main() {
	openAIClient, err := openai.NewClient()
	if err != nil {
		panic(err)
	}

	systemPrompt := "You're a helpful assistant that loves to say Hello World! as much as humanly possible."
	conversation := openai.NewConversation(openai.GPT_4_TURBO, systemPrompt, openAIClient)

	chunks, err := conversation.ChatStream("What are the five best things about Montreal? Your response must be at least 2001 characters.")
	if err != nil {
		panic(err)
	}

	discordClient, err := discord.NewClient(500)
	if err != nil {
		panic(err)
	}

	if err := discordClient.StreamMessage(chunks, "1183176110150791179"); err != nil {
		panic(err)
	}
}
