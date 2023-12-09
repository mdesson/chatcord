package main

import (
	"fmt"
	openai "github.com/mdesson/chatcord/openai"
)

func main() {
	client, err := openai.NewClient(openai.GPT_4_TURBO)
	if err != nil {
		panic(err)
	}

	systemPrompt := "You're a helpful assistant that loves to say Hello World! as much as humanly possible."
	conversation := openai.NewConversation(openai.GPT_4_TURBO, systemPrompt, client)

	chunks, err := conversation.ChatStream("What's the best phrase in the world and why!")
	if err != nil {
		panic(err)
	}

	for chunk := range chunks {
		fmt.Printf("%s", chunk)
	}
}
