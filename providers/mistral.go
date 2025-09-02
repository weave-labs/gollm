// Package providers implements LLM provider interfaces and implementations.
package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/weave-labs/gollm/config"
	"github.com/weave-labs/gollm/internal/logging"
	"github.com/weave-labs/weave-go/weaveapi/llmx/v1"
)

// Common parameter keys
const (
	mistralKeyMaxTokens      = "max_tokens"
	mistralKeyStream         = "stream"
	mistralKeyModel          = "model"
	mistralKeyMessages       = "messages"
	mistralKeySystemPrompt   = "system_prompt"
	mistralKeyTools          = "tools"
	mistralKeyToolChoice     = "tool_choice"
	mistralKeyResponseFormat = "response_format"
	mistralKeyStrict         = "strict"
	mistralKeyTemperature    = "temperature"
	mistralKeySeed           = "seed"
)

// MistralProvider implements the Provider interface for Mistral AI's API.
// It supports Mistral's language models and provides access to their capabilities,
// including chat completion and structured output.
type MistralProvider struct {
	logger       logging.Logger
	extraHeaders map[string]string
	options      map[string]any
	apiKey       string
	model        string
}

// NewMistralProvider creates a new Mistral provider instance.
// It initializes the provider with the given API key, model, and optional headers.
//
// Parameters:
//   - apiKey: Mistral API key for authentication
//   - model: The model to use (e.g., "mistral-large", "mistral-medium")
//   - extraHeaders: Additional HTTP headers for requests
//
// Returns:
//   - A configured Mistral Provider instance
func NewMistralProvider(apiKey, model string, extraHeaders map[string]string) *MistralProvider {
	if extraHeaders == nil {
		extraHeaders = make(map[string]string)
	}

	p := &MistralProvider{
		apiKey:       apiKey,
		model:        model,
		extraHeaders: extraHeaders,
		options:      make(map[string]any),
		logger:       logging.NewLogger(logging.LogLevelInfo),
	}

	// AddCapability capabilities based on model
	p.registerCapabilities()
	return p
}

// SetLogger configures the logger for the Mistral provider.
// This is used for debugging and monitoring API interactions.
func (p *MistralProvider) SetLogger(logger logging.Logger) {
	p.logger = logger
}

// SetOption sets a specific option for the Mistral provider.
// Supported options include:
//   - temperature: Controls randomness (0.0 to 1.0)
//   - max_tokens: Maximum tokens in the response
//   - top_p: Nucleus sampling parameter
//   - random_seed: Random seed for deterministic sampling
func (p *MistralProvider) SetOption(key string, value any) {
	p.options[key] = value
}

// SetDefaultOptions configures standard options from the global configuration.
// This includes temperature, max tokens, and sampling parameters.
func (p *MistralProvider) SetDefaultOptions(cfg *config.Config) {
	p.SetOption(mistralKeyTemperature, cfg.Temperature)
	p.SetOption(mistralKeyMaxTokens, cfg.MaxTokens)
	if cfg.Seed != nil {
		p.SetOption(mistralKeySeed, *cfg.Seed)
	}
}

// Name returns "mistral" as the provider identifier.
func (p *MistralProvider) Name() string {
	return "mistral"
}

// registerCapabilities registers capabilities for all known Mistral models
func (p *MistralProvider) registerCapabilities() {
	registry := GetCapabilityRegistry()

	// Define all known Mistral models
	allModels := []string{
		// Current latest models
		"mistral-large-latest",
		"mistral-medium-latest",
		"mistral-small-latest",
		"devstral-small-latest",
		"codestral-latest",
		"ministral-8b-latest",
		"ministral-3b-latest",
		"pixtral-12b-latest",
		"pixtral-large-latest",

		// Versioned models
		"mistral-large-2411",
		"mistral-large-2407",
		"mistral-medium-2312",
		"mistral-small-2312",
		"mistral-small-2402",
		"codestral-2405",
		"ministral-8b-2410",
		"ministral-3b-2410",
		"pixtral-12b-2409",

		// Other models
		"open-mistral-nemo",
		"open-mistral-7b",
		"open-mixtral-8x7b",
		"open-mixtral-8x22b",
		"codestral-mamba",
		"mistral-embed",
	}

	for _, model := range allModels {
		// Structured response - all models except codestral-mamba
		if model != "codestral-mamba" && model != "mistral-embed" {
			registry.RegisterCapability(ProviderMistral, model,
				llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, &llmx.StructuredResponse{
					MaxSchemaDepth:   10,
					SupportedFormats: []llmx.DataFormat{llmx.DataFormat_DATA_FORMAT_JSON},
					RequiresJsonMode: true,
					SupportedTypes: []llmx.JsonSchemaType{
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxProperties: 100,
				})
		}

		// Function calling - specific models
		functionCallingSupportedModels := map[string]bool{
			"mistral-large-latest":  true,
			"mistral-large-2411":    true,
			"mistral-large-2407":    true,
			"mistral-medium-latest": true,
			"mistral-medium-2312":   true,
			"mistral-small-latest":  true,
			"mistral-small-2312":    true,
			"mistral-small-2402":    true,
			"devstral-small-latest": true,
			"codestral-latest":      true,
			"codestral-2405":        true,
			"ministral-8b-latest":   true,
			"ministral-8b-2410":     true,
			"ministral-3b-latest":   true,
			"ministral-3b-2410":     true,
			"pixtral-12b-latest":    true,
			"pixtral-12b-2409":      true,
			"pixtral-large-latest":  true,
			"open-mistral-nemo":     true,
		}

		if functionCallingSupportedModels[model] {
			registry.RegisterCapability(ProviderMistral, model, llmx.CapabilityType_CAPABILITY_TYPE_FUNCTION_CALLING,
				&llmx.FunctionCalling{
					MaxFunctions:      100,
					SupportsParallel:  true,
					MaxParallelCalls:  10,
					SupportsStreaming: true,
					RequiresToolRole:  false,
					SupportedParameterTypes: []llmx.JsonSchemaType{
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxNestingDepth: 10,
				})
		}

		// All Mistral models support streaming (except embed)
		if model != "mistral-embed" {
			registry.RegisterCapability(ProviderMistral, model, llmx.CapabilityType_CAPABILITY_TYPE_STREAMING,
				&llmx.Streaming{
					SupportsSse:    true,
					BufferSize:     4096,
					ChunkDelimiter: "data: ",
					SupportsUsage:  true,
				})
		}

		// Vision for pixtral models
		if strings.Contains(model, "pixtral") {
			registry.RegisterCapability(ProviderMistral, model, llmx.CapabilityType_CAPABILITY_TYPE_VISION,
				&llmx.Vision{
					MaxImageSizeBytes: 10 * 1024 * 1024,
					SupportedFormats: []llmx.ImageFormat{
						llmx.ImageFormat_IMAGE_FORMAT_JPEG,
						llmx.ImageFormat_IMAGE_FORMAT_PNG,
						llmx.ImageFormat_IMAGE_FORMAT_WEBP,
					},
					MaxImagesPerRequest:     5,
					SupportsVideoFrames:     false,
					SupportsOcr:             true,
					SupportsObjectDetection: false,
				})
		}
	}
}

// HasCapability checks if a capability is supported
func (p *MistralProvider) HasCapability(capability llmx.CapabilityType, model string) bool {
	targetModel := p.model
	if model != "" {
		targetModel = model
	}
	return GetCapabilityRegistry().HasCapability(ProviderMistral, targetModel, capability)
}

// Endpoint returns the Mistral API endpoint URL.
// This is "https://api.mistral.ai/v1/chat/completions".
func (p *MistralProvider) Endpoint() string {
	return "https://api.mistral.ai/v1/chat/completions"
}

// Headers returns the required HTTP headers for Mistral API requests.
// This includes:
//   - Authorization: Bearer token using the API key
//   - Content-Type: application/json
//   - Any additional headers specified via SetExtraHeaders
func (p *MistralProvider) Headers() map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}

	for key, value := range p.extraHeaders {
		headers[key] = value
	}

	return headers
}

// PrepareRequest creates the request body for a Mistral API call using the new Request structure.
func (p *MistralProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeRequestBodyWithModel(model)

	// Add system prompt if present
	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	if systemPrompt != "" {
		p.addSystemPromptToRequestBody(requestBody, systemPrompt)
	}

	// Add messages
	p.addMessagesToRequestBody(requestBody, req.Messages)

	// Add structured response if supported
	if req.ResponseSchema != nil && p.HasCapability(llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
		p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
	}

	// Add remaining options
	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// ParseResponse extracts the generated text from the Mistral API response.
// It handles various response formats and error cases.
//
// Parameters:
//   - body: Raw API response body
//
// Returns:
//   - Generated text content
//   - Any error encountered during parsing
func (p *MistralProvider) ParseResponse(body []byte) (*Response, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return nil, errors.New("empty response from API")
	}

	// Combine content and tool calls
	var finalResponse strings.Builder
	finalResponse.WriteString(response.Choices[0].Message.Content)

	// Process tool calls if present
	for _, toolCall := range response.Choices[0].Message.ToolCalls {
		// Parse arguments as raw JSON to preserve the exact format
		var args any
		if err := json.Unmarshal(toolCall.Function.Arguments, &args); err != nil {
			return nil, fmt.Errorf("error parsing function arguments: %w", err)
		}

		functionCall, err := FormatFunctionCall(toolCall.Function.Name, args)
		if err != nil {
			return nil, fmt.Errorf("error formatting function call: %w", err)
		}
		if finalResponse.Len() > 0 {
			finalResponse.WriteString("\n")
		}
		finalResponse.WriteString(functionCall)
	}

	return &Response{Content: Text{Value: finalResponse.String()}}, nil
}

// SetExtraHeaders configures additional HTTP headers for API requests.
// This allows for custom headers needed for specific features or requirements.
func (p *MistralProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.extraHeaders = extraHeaders
}

// PrepareStreamRequest creates a request body for streaming API calls
func (p *MistralProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeRequestBodyWithModel(model)
	requestBody[mistralKeyStream] = true

	// Add system prompt if present
	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	if systemPrompt != "" {
		p.addSystemPromptToRequestBody(requestBody, systemPrompt)
	}

	// Add messages
	p.addMessagesToRequestBody(requestBody, req.Messages)

	// Add structured response if supported
	if req.ResponseSchema != nil && p.HasCapability(llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
		p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
	}

	// Add remaining options
	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// ParseStreamResponse parses a single chunk from a streaming response
func (p *MistralProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
	// Skip empty lines
	if len(bytes.TrimSpace(chunk)) == 0 {
		return nil, errors.New("empty chunk")
	}
	// [DONE] guard
	if bytes.Equal(bytes.TrimSpace(chunk), []byte("[DONE]")) {
		return nil, io.EOF
	}

	var response struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(chunk, &response); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Delta.Content == "" {
		return nil, errors.New("skip token")
	}

	return &Response{Content: Text{Value: response.Choices[0].Delta.Content}}, nil
}

// initializeRequestBodyWithModel creates the base request structure with specified model
func (p *MistralProvider) initializeRequestBodyWithModel(model string) map[string]any {
	return map[string]any{
		mistralKeyModel:     model,
		mistralKeyMaxTokens: p.options[mistralKeyMaxTokens],
		mistralKeyMessages:  []map[string]any{},
	}
}

// extractSystemPromptFromRequest gets system prompt from request or options
func (p *MistralProvider) extractSystemPromptFromRequest(req *Request, options map[string]any) string {
	// Priority: Request.SystemPrompt > options["system_prompt"]
	if req.SystemPrompt != "" {
		return req.SystemPrompt
	}
	if sp, ok := options[mistralKeySystemPrompt].(string); ok && sp != "" {
		return sp
	}
	return ""
}

// addSystemPromptToRequestBody adds the system prompt to the request
func (p *MistralProvider) addSystemPromptToRequestBody(requestBody map[string]any, systemPrompt string) {
	if systemPrompt == "" {
		return
	}

	if messagesArray, ok := requestBody[mistralKeyMessages].([]map[string]any); ok {
		systemMessage := map[string]any{
			"role":    "system",
			"content": systemPrompt,
		}
		requestBody[mistralKeyMessages] = append(messagesArray, systemMessage)
	}
}

// addMessagesToRequestBody converts Request messages to Mistral format
func (p *MistralProvider) addMessagesToRequestBody(requestBody map[string]any, messages []Message) {
	if messagesArray, ok := requestBody[mistralKeyMessages].([]map[string]any); ok {
		for _, msg := range messages {
			mistralMessage := map[string]any{
				"role":    msg.Role,
				"content": msg.Content,
			}
			if msg.Name != "" {
				mistralMessage["name"] = msg.Name
			}
			if len(msg.ToolCalls) > 0 {
				mistralMessage["tool_calls"] = msg.ToolCalls
			}
			if msg.ToolCallID != "" {
				mistralMessage["tool_call_id"] = msg.ToolCallID
			}
			messagesArray = append(messagesArray, mistralMessage)
		}
		requestBody[mistralKeyMessages] = messagesArray
	}
}

// addStructuredResponseToRequest adds structured response schema to the request
func (p *MistralProvider) addStructuredResponseToRequest(requestBody map[string]any, schema any) {
	requestBody[mistralKeyResponseFormat] = map[string]any{
		"type":   "json_schema",
		"schema": schema,
	}
}

// addRemainingOptions adds provider options and additional options to the request
func (p *MistralProvider) addRemainingOptions(requestBody map[string]any, options map[string]any) {
	// Add provider options first
	for k, v := range p.options {
		if k != mistralKeyMaxTokens { // Already added in initialize
			requestBody[k] = v
		}
	}

	// Add additional options (may override provider options)
	for k, v := range options {
		if k != mistralKeySystemPrompt { // Already handled
			requestBody[k] = v
		}
	}
}
