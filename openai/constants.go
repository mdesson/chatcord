package openai

type Model string
type Role string

const base_url = "https://api.openai.com/v1"

// Underlying OpenAI Model
const (
	GPT_4_TURBO  Model = "gpt-4-turbo-preview"
	GPT_4_VISION Model = "gpt-4-vision-preview"
	GPT_3_TURBO  Model = "gpt-3.5-turbo"
)

// Chat participant Role
const (
	ROLE_SYSTEM    Role = "system"
	ROLE_ASSISTANT Role = "assistant"
	ROLE_USER      Role = "user"
)
