package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/weave-labs/gollm/config"
	"github.com/weave-labs/gollm/internal/logging"
	"github.com/weave-labs/weave-go/weaveapi/llmx/v1"
)

const (
	cohereKeyText           = "text"
	cohereKeySystemPrompt   = "system_prompt"
	cohereKeyPreamble       = "preamble"
	cohereKeyMessages       = "messages"
	cohereKeyResponseFormat = "response_format"
	cohereKeyStream         = "stream"
)

// CohereProvider implements the Provider interface for Cohere's API.
// It supports Cohere's language models and provides access to their capabilities,
// including chat completion and structured output
type CohereProvider struct {
	logger       logging.Logger
	extraHeaders map[string]string
	options      map[string]any
	apiKey       string
	model        string
}

// NewCohereProvider creates a new Cohere provider instance.
// It initializes the provider with the given API key, model, and optional headers.
func NewCohereProvider(apiKey, model string, extraHeaders map[string]string) *CohereProvider {
	if extraHeaders == nil {
		extraHeaders = make(map[string]string)
	}

	p := &CohereProvider{
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

// Name returns "cohere" as the provider identifier.
func (p *CohereProvider) Name() string {
	return "cohere"
}

// registerCapabilities registers capabilities for all known Cohere models
func (p *CohereProvider) registerCapabilities() {
	registry := GetCapabilityRegistry()

	// Define all known Cohere models
	allModels := []string{
		// Command A models
		"command-a-03-2025",

		// Command R Plus models
		"command-r-plus-08-2024",
		"command-r-plus-04-2024",
		"command-r-plus",

		// Command R models
		"command-r-08-2024",
		"command-r-03-2024",
		"command-r",

		// Legacy Command models
		"command",
		"command-light",
		"command-nightly",
		"command-light-nightly",
	}

	for _, model := range allModels {
		// Structured response support for newer models
		structuredResponseModels := []string{
			"command-a-03-2025",
			"command-r-plus-08-2024",
			"command-r-plus-04-2024",
			"command-r-plus",
			"command-r-08-2024",
			"command-r-03-2024",
			"command-r",
		}

		if slices.Contains(structuredResponseModels, model) {
			// IMPORTANT: Cohere quirk - structured response only via tool calling
			registry.RegisterCapability(ProviderCohere, model,
				llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, &llmx.StructuredResponse{
					RequiresToolUse:  true, // THE COHERE QUIRK!
					MaxSchemaDepth:   5,
					SupportedFormats: []llmx.DataFormat{llmx.DataFormat_DATA_FORMAT_JSON},
					SystemPromptHint: "You must use the provided tool to structure your response",
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

		// Function calling support
		if strings.Contains(model, "command-r") {
			registry.RegisterCapability(ProviderCohere, model, llmx.CapabilityType_CAPABILITY_TYPE_FUNCTION_CALLING,
				&llmx.FunctionCalling{
					MaxFunctions:      50,
					SupportsParallel:  false,
					RequiresToolRole:  true,
					SupportsStreaming: true,
					MaxParallelCalls:  1,
					SupportedParameterTypes: []llmx.JsonSchemaType{
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxNestingDepth: 5,
				})
		} else if strings.Contains(model, "command") {
			registry.RegisterCapability(ProviderCohere, model, llmx.CapabilityType_CAPABILITY_TYPE_FUNCTION_CALLING,
				&llmx.FunctionCalling{
					MaxFunctions:      20,
					SupportsParallel:  false,
					RequiresToolRole:  true,
					SupportsStreaming: false,
					MaxParallelCalls:  1,
					SupportedParameterTypes: []llmx.JsonSchemaType{
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						llmx.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxNestingDepth: 5,
				})
		}

		// All Cohere models support streaming
		registry.RegisterCapability(ProviderCohere, model, llmx.CapabilityType_CAPABILITY_TYPE_STREAMING,
			&llmx.Streaming{
				SupportsSse:    true,
				BufferSize:     8192,
				ChunkDelimiter: "\n",
				SupportsUsage:  false,
			})
	}
}

// HasCapability checks if a capability is supported
func (p *CohereProvider) HasCapability(capability llmx.CapabilityType, model string) bool {
	targetModel := p.model
	if model != "" {
		targetModel = model
	}
	return GetCapabilityRegistry().HasCapability(ProviderCohere, targetModel, capability)
}

// Endpoint returns the base URL for the Cohere API.
// This is "https://api.cohere.com/v2/chat".
func (p *CohereProvider) Endpoint() string {
	return "https://api.cohere.com/v2/chat"
}

// Headers return the required HTTP headers for Cohere API requests.
// This includes:
//   - Content-type: application/json
//   - Authorization: Bearer token using the API key
//   - Any additional headers specified via SetExtraHeaders
func (p *CohereProvider) Headers() map[string]string {
	headers := map[string]string{
		"Content-type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}

	for k, v := range p.extraHeaders {
		headers[k] = v
	}
	return headers
}

// SetExtraHeaders configures additional HTTP headers for API requests.
// This allows for custom headers needed for specific features or requirements.
func (p *CohereProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.extraHeaders = extraHeaders
	p.logger.Debug("Extra headers set", "headers", extraHeaders)
}

// SetDefaultOptions configures standard options from the global configuration.
// This includes temperature, max tokens, and sampling parameters.
func (p *CohereProvider) SetDefaultOptions(cfg *config.Config) {
	p.SetOption("temperature", cfg.Temperature)
	p.SetOption("max_tokens", cfg.MaxTokens)
	p.SetOption(cohereKeyStream, false)
	if cfg.Seed != nil {
		p.SetOption("seed", *cfg.Seed)
	}
}

// SetOption sets a specific option for the Cohere provider.
// Support options include:
//   - temperature: Controls randomness
//   - max_tokens: Maximum tokens in the response
//   - p: Total probability mass (0.01 to 0.99)
//   - k: Top k most likely tokens are considered
//   - strict_tools: If set to true, follow tool definition strictly
func (p *CohereProvider) SetOption(key string, value any) {
	p.options[key] = value
	if p.logger != nil {
		p.logger.Debug("Setting option for Cohere", "key", key, "value", value)
	}
}

// SetLogger configures the logger for the Cohere provider.
// This is used for debugging and monitoring API interactions.
func (p *CohereProvider) SetLogger(logger logging.Logger) {
	p.logger = logger
}

// PrepareRequest creates the request body for a Cohere API call
func (p *CohereProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeRequestBodyWithModel(model)

	p.addMessagesToRequestBody(requestBody, req.Messages)

	if req.SystemPrompt != "" {
		requestBody[cohereKeyPreamble] = req.SystemPrompt
	} else if systemPrompt, ok := options[cohereKeySystemPrompt].(string); ok && systemPrompt != "" {
		requestBody[cohereKeyPreamble] = systemPrompt
	}

	if req.ResponseSchema != nil && p.HasCapability(llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
		p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
	}

	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// ParseResponse extracts the generated text from the Cohere API response.
// It handles various response formats and error cases
func (p *CohereProvider) ParseResponse(body []byte) (*Response, error) {
	var response struct {
		Message struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(response.Message.Content) == 0 {
		return nil, errors.New("empty response from API")
	}

	var finalResponse strings.Builder

	for _, content := range response.Message.Content {
		if content.Type == cohereKeyText {
			finalResponse.WriteString(content.Text)
			p.logger.Debug("Text content: %s", content.Text)
		}
	}

	for _, toolCall := range response.Message.ToolCalls {
		var args any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
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

	p.logger.Debug("Final response: %s", finalResponse.String())
	return &Response{Content: Text{Value: finalResponse.String()}}, nil
}

// PrepareStreamRequest prepares a request body for streaming
func (p *CohereProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeRequestBodyWithModel(model)
	requestBody[cohereKeyStream] = true

	p.addMessagesToRequestBody(requestBody, req.Messages)

	if req.SystemPrompt != "" {
		requestBody[cohereKeyPreamble] = req.SystemPrompt
	} else if systemPrompt, ok := options[cohereKeySystemPrompt].(string); ok && systemPrompt != "" {
		requestBody[cohereKeyPreamble] = systemPrompt
	}

	if req.ResponseSchema != nil && p.HasCapability(llmx.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
		p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
	}

	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// ParseStreamResponse parses a single chunk from a streaming response
func (p *CohereProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
	var response struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(chunk, &response); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}
	if response.Text == "" {
		return nil, errors.New("skip resp")
	}
	return &Response{Content: Text{Value: response.Text}}, nil
}

// initializeRequestBodyWithModel creates the base request structure with specified model
func (p *CohereProvider) initializeRequestBodyWithModel(model string) map[string]any {
	return map[string]any{
		"model":           model,
		cohereKeyMessages: []map[string]any{},
	}
}

// addMessagesToRequestBody converts and adds messages to the request
func (p *CohereProvider) addMessagesToRequestBody(
	requestBody map[string]any,
	messages []Message,
) {
	cohereMessages := make([]map[string]any, 0, len(messages))

	for i := range messages {
		cohereMsg := p.convertMessageToCohereFormat(&messages[i])
		cohereMessages = append(cohereMessages, cohereMsg)
	}

	requestBody[cohereKeyMessages] = cohereMessages
}

// convertMessageToCohereFormat converts a Message to Cohere's format
func (p *CohereProvider) convertMessageToCohereFormat(msg *Message) map[string]any {
	cohereMsg := map[string]any{
		"role":    msg.Role,
		"content": msg.Content,
	}

	if msg.Name != "" {
		cohereMsg["name"] = msg.Name
	}

	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, len(msg.ToolCalls))
		for i, toolCall := range msg.ToolCalls {
			toolCalls[i] = map[string]any{
				"id":   toolCall.ID,
				"type": toolCall.Type,
				"function": map[string]any{
					"name":      toolCall.Function.Name,
					"arguments": string(toolCall.Function.Arguments),
				},
			}
		}
		cohereMsg["tool_calls"] = toolCalls
	}

	return cohereMsg
}

// addStructuredResponseToRequest adds structured response schema to the request
func (p *CohereProvider) addStructuredResponseToRequest(requestBody map[string]any, schema any) {
	requestBody[cohereKeyResponseFormat] = map[string]any{
		"type":        "json_object",
		"json_schema": schema,
	}
}

// addRemainingOptions adds non-handled options to the request
func (p *CohereProvider) addRemainingOptions(requestBody map[string]any, options map[string]any) {
	// First, add default options
	for k, v := range p.options {
		if !p.isGlobalOption(k) {
			requestBody[k] = v
		}
	}

	// Then, add any additional options (which may override defaults)
	for k, v := range options {
		if !p.isGlobalOption(k) {
			requestBody[k] = v
		}
	}
}

// isGlobalOption checks if an option is already handled
func (p *CohereProvider) isGlobalOption(key string) bool {
	return key == cohereKeySystemPrompt ||
		key == cohereKeyPreamble ||
		key == cohereKeyMessages ||
		key == cohereKeyResponseFormat ||
		key == cohereKeyStream
}
