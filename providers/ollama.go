// Package providers implements LLM provider interfaces and implementations.
package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/weave-labs/gollm/config"
	"github.com/weave-labs/gollm/internal/logging"
)

// Common parameter keys for Ollama
const (
	ollamaKeyModel    = "model"
	ollamaKeyPrompt   = "prompt"
	ollamaKeyStream   = "stream"
	ollamaKeyMessages = "messages"
)

// OllamaProvider implements the Provider interface for Ollama's API.
// It enables interaction with locally hosted language models through Ollama,
// supporting various open-source models like Llama, Mistral, and others.
type OllamaProvider struct {
	logger       logging.Logger
	extraHeaders map[string]string
	options      map[string]any
	endpoint     string
	model        string
}

// NewOllamaProvider creates a new Ollama provider instance.
// It initializes the provider with the specified endpoint URL and model name.
// Note that Ollama typically doesn't require an API key, so the apiKey parameter is ignored.
//
// Parameters:
//   - endpoint: The Ollama API endpoint URL (e.g., "http://localhost:11434")
//   - model: The model to use (e.g., "llama2", "mistral")
//   - extraHeaders: Additional HTTP headers for requests
//
// Returns:
//   - A configured Ollama Provider instance
func NewOllamaProvider(_ string, model string, extraHeaders map[string]string) *OllamaProvider {
	endpoint := "http://localhost:11434"
	if extraHeaders == nil {
		extraHeaders = make(map[string]string)
	}

	p := &OllamaProvider{
		endpoint:     endpoint,
		model:        model,
		extraHeaders: extraHeaders,
		options:      make(map[string]any),
		logger:       logging.NewLogger(logging.LogLevelInfo),
	}

	// Register capabilities based on model
	p.registerCapabilities()
	return p
}

// Name returns the identifier for this provider ("ollama").
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// registerCapabilities registers capabilities for all known Ollama models
func (p *OllamaProvider) registerCapabilities() {
	registry := GetRegistry()

	// Define common Ollama models (this list is extensive but not exhaustive)
	allModels := []string{
		// Llama models
		"llama3.2", "llama3.2:1b", "llama3.2:3b",
		"llama3.1", "llama3.1:8b", "llama3.1:70b", "llama3.1:405b",
		"llama3", "llama3:8b", "llama3:70b",
		"llama2", "llama2:7b", "llama2:13b", "llama2:70b",
		"llama2-uncensored", "llama2-chinese",

		// Code Llama models
		"codellama", "codellama:7b", "codellama:13b", "codellama:34b",
		"codellama:python", "codellama:instruct",

		// Mistral models
		"mistral", "mistral:7b", "mistral:instruct",
		"mistral-openorca", "mistral-nemo",
		"mixtral", "mixtral:8x7b", "mixtral:8x22b",

		// Gemma models
		"gemma", "gemma:2b", "gemma:7b",
		"gemma2", "gemma2:2b", "gemma2:9b", "gemma2:27b",

		// Phi models
		"phi3", "phi3:mini", "phi3:medium",
		"phi", "phi:2.7b",

		// Qwen models
		"qwen", "qwen:4b", "qwen:7b", "qwen:14b", "qwen:32b", "qwen:72b",
		"qwen2", "qwen2:0.5b", "qwen2:1.5b", "qwen2:7b", "qwen2:72b",
		"qwen2.5", "qwen2.5:0.5b", "qwen2.5:1.5b", "qwen2.5:3b", "qwen2.5:7b", "qwen2.5:14b", "qwen2.5:32b",
		"qwen2.5:72b",

		// Other popular models
		"falcon", "falcon:7b", "falcon:40b",
		"vicuna", "vicuna:7b", "vicuna:13b", "vicuna:33b",
		"orca-mini", "orca2",
		"dolphin-llama3", "dolphin-mistral",
		"neural-chat", "starling-lm",
		"openchat", "zephyr",
		"wizardlm", "wizardcoder",
		"yi", "yi:6b", "yi:9b", "yi:34b",
		"deepseek-coder", "deepseek-llm",
		"solar", "solar:10.7b",
		"tinyllama", "tinydolphin",
		"stablelm2", "starcoder", "starcoder2",

		// Vision models
		"llava", "llava:7b", "llava:13b", "llava:34b",
		"bakllava", "moondream",

		// Embedding models
		"nomic-embed-text", "mxbai-embed-large", "all-minilm",

		// Special/utility models
		"medllama2", "samantha-mistral", "magicoder",
		"orca-mini:3b", "orca-mini:7b", "orca-mini:13b", "orca-mini:70b",
	}

	for _, model := range allModels {
		// Ollama supports streaming for all models
		registry.Register(ProviderOllama, model, CapStreaming, StreamingConfig{
			SupportsSSE:    true,
			BufferSize:     4096,
			ChunkDelimiter: "data: ",
			SupportsUsage:  true,
		})

		// Vision capability for vision models
		visionModels := []string{"llava", "llava:7b", "llava:13b", "llava:34b", "bakllava", "moondream"}
		for _, vm := range visionModels {
			if strings.Contains(model, vm) || model == vm {
				registry.Register(ProviderOllama, model, CapVision, VisionConfig{
					MaxImageSize:        10 * 1024 * 1024,
					SupportedFormats:    []string{"jpeg", "png", "gif", "webp"},
					MaxImagesPerRequest: 1,
				})
				break
			}
		}

		// System prompt support for all models (basic capability)
		registry.Register(ProviderOllama, model, CapSystemPrompt, SystemPromptConfig{
			MaxLength:        8192,
			SupportsMultiple: false,
		})
	}

	// Ollama doesn't support structured responses or function calling natively
	// These capabilities are intentionally not registered
}

// HasCapability checks if a capability is supported
func (p *OllamaProvider) HasCapability(capability Capability, model string) bool {
	targetModel := p.model
	if model != "" {
		targetModel = model
	}
	return GetRegistry().HasCapability(ProviderOllama, targetModel, capability)
}

// Endpoint returns the configured Ollama API endpoint URL.
// This is typically "http://localhost:11434/api/generate".
func (p *OllamaProvider) Endpoint() string {
	return p.endpoint + "/api/generate"
}

// Headers returns the HTTP headers required for Ollama API requests.
// This includes content type and any custom headers.
func (p *OllamaProvider) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// SetExtraHeaders configures additional HTTP headers for API requests.
// This allows for custom headers needed for specific features or requirements.
func (p *OllamaProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.extraHeaders = extraHeaders
}

// SetDefaultOptions configures standard options from the global configuration.
// This includes temperature and other generation parameters.
func (p *OllamaProvider) SetDefaultOptions(cfg *config.Config) {
	p.SetOption("temperature", cfg.Temperature)
	p.SetOption("num_predict", cfg.MaxTokens)
	if cfg.Seed != nil {
		p.SetOption("seed", *cfg.Seed)
	}
	if cfg.OllamaEndpoint != "" {
		p.endpoint = cfg.OllamaEndpoint
	}
	p.SetOption("top_p", cfg.TopP)
	p.SetOption("min_p", cfg.MinP)
	p.SetOption("repeat_penalty", cfg.RepeatPenalty)
	p.SetOption("repeat_last_n", cfg.RepeatLastN)
	p.SetOption("mirostat", cfg.Mirostat)
	p.SetOption("mirostat_eta", cfg.MirostatEta)
	p.SetOption("mirostat_tau", cfg.MirostatTau)
	p.SetOption("tfs_z", cfg.TfsZ)
}

// SetOption sets a model-specific option for the Ollama provider.
// Supported options include:
//   - temperature: Controls randomness (0.0 to 1.0)
//   - num_predict: Maximum number of tokens to generate
//   - top_p: Nucleus sampling parameter
//   - top_k: Top-k sampling parameter
//   - stop: Custom stop sequences
func (p *OllamaProvider) SetOption(key string, value any) {
	p.options[key] = value
	if p.logger != nil {
		p.logger.Debug("Setting option for Ollama", "key", key, "value", value)
	}
}

// SetLogger configures the logger for the Ollama provider.
// This is used for debugging and monitoring API interactions.
func (p *OllamaProvider) SetLogger(logger logging.Logger) {
	p.logger = logger
}

// PrepareRequest creates the request body for an Ollama API call.
// It formats the request according to Ollama's API requirements.
func (p *OllamaProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := map[string]any{
		ollamaKeyModel: model,
	}

	// Convert messages to a single prompt for Ollama
	if len(req.Messages) > 0 {
		var prompt strings.Builder

		// Add system prompt if present
		if req.SystemPrompt != "" {
			prompt.WriteString("System: ")
			prompt.WriteString(req.SystemPrompt)
			prompt.WriteString("\n\n")
		}

		// Add all messages
		for _, msg := range req.Messages {
			prompt.WriteString(msg.Role)
			prompt.WriteString(": ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		}

		requestBody[ollamaKeyPrompt] = strings.TrimSpace(prompt.String())
	}

	// Add remaining options
	for k, v := range options {
		requestBody[k] = v
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// PrepareStreamRequest prepares a request body for streaming
func (p *OllamaProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
	// Ollama doesn't support structured response natively; proceed with standard streaming
	options[ollamaKeyStream] = true
	return p.PrepareRequest(req, options)
}

// ParseResponse extracts the generated text from the Ollama API response.
// It handles Ollama's streaming response format and concatenates the results.
//
// Parameters:
//   - body: Raw API response body
//
// Returns:
//   - Generated text content
//   - Any error encountered during parsing
func (p *OllamaProvider) ParseResponse(body []byte) (*Response, error) {
	var fullText strings.Builder
	var promptEvalCount int64
	var evalCount int64

	decoder := json.NewDecoder(bytes.NewReader(body))

	for decoder.More() {
		var response struct {
			Model           string `json:"model"`
			Response        string `json:"response"`
			Done            bool   `json:"done"`
			PromptEvalCount int64  `json:"prompt_eval_count"`
			EvalCount       int64  `json:"eval_count"`
		}
		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("error parsing Ollama response: %w", err)
		}
		if response.Response != "" {
			fullText.WriteString(response.Response)
		}
		// Capture usage as we see it; typically populated on the final object
		if response.PromptEvalCount > 0 {
			promptEvalCount = response.PromptEvalCount
		}
		if response.EvalCount > 0 {
			evalCount = response.EvalCount
		}
		if response.Done {
			break
		}
	}

	resp := &Response{Content: Text{Value: fullText.String()}}
	// Attach usage if we captured any token counts
	if promptEvalCount > 0 || evalCount > 0 {
		resp.Usage = NewUsage(promptEvalCount, 0, evalCount, 0, 0)
	}
	return resp, nil
}

// ParseStreamResponse parses a single chunk from a streaming response
func (p *OllamaProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
	var response struct {
		Response        string `json:"response"`
		Done            bool   `json:"done"`
		PromptEvalCount int64  `json:"prompt_eval_count"`
		EvalCount       int64  `json:"eval_count"`
	}
	if err := json.Unmarshal(chunk, &response); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}
	// When done=true, no more content; return usage so stream can expose token counts
	if response.Done {
		usage := (*Usage)(nil)
		if response.PromptEvalCount > 0 || response.EvalCount > 0 {
			usage = NewUsage(response.PromptEvalCount, 0, response.EvalCount, 0, 0)
		}
		return &Response{Usage: usage}, nil
	}
	if strings.TrimSpace(response.Response) == "" {
		return nil, errors.New("skip resp")
	}
	return &Response{Content: Text{Value: response.Response}}, nil
}
