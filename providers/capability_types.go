// Package providers implements various Language Learning Model (LLM) provider interfaces
package providers

// ProviderName ensures we use correct provider identifiers
type ProviderName string

const (
	ProviderOpenAI     ProviderName = "openai"
	ProviderGemini     ProviderName = "gemini"
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderCohere     ProviderName = "cohere"
	ProviderOllama     ProviderName = "ollama"
	ProviderGroq       ProviderName = "groq"
	ProviderMistral    ProviderName = "mistral"
	ProviderDeepSeek   ProviderName = "deepseek"
	ProviderOpenRouter ProviderName = "openrouter"
)

// Capability represents a feature that a provider/model may support
type Capability string

const (
	CapStructuredResponse Capability = "structured_response"
	CapStreaming          Capability = "streaming"
	CapFunctionCalling    Capability = "function_calling"
	CapVision             Capability = "vision"
	CapToolUse            Capability = "tool_use"
	CapSystemPrompt       Capability = "system_prompt"
	CapCaching            Capability = "caching"
)
