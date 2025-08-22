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
	"github.com/weave-labs/gollm/internal/models"
	modexv1 "github.com/weave-labs/weave-go/weaveapi/modex/v1"
)

const (
	openAIKeyMaxTokens           = "max_tokens"
	openAIKeyToolChoice          = "tool_choice"
	openAIKeySystemPrompt        = "system_prompt"
	openAIKeyTools               = "tools"
	openAIKeyStructuredMessages  = "structured_messages"
	openAIKeyMaxCompletionTokens = "max_completion_tokens"
	openAIKeyStream              = "stream"
)

// OpenAIProvider implements the Provider interface for OpenAI's API.
// It supports GPT models and provides access to OpenAI's language model capabilities,
// including function calling, JSON mode, and structured output validation.
type OpenAIProvider struct {
	logger       logging.Logger
	extraHeaders map[string]string
	options      map[string]any
	apiKey       string
	model        string
}

// NewOpenAIProvider creates a new OpenAI provider instance.
// It initializes the provider with the given API key, model, and optional headers.
func NewOpenAIProvider(apiKey, model string, extraHeaders map[string]string) *OpenAIProvider {
	if extraHeaders == nil {
		extraHeaders = make(map[string]string)
	}

	p := &OpenAIProvider{
		apiKey:       apiKey,
		model:        model,
		extraHeaders: extraHeaders,
		options:      make(map[string]any),
		logger:       logging.NewLogger(logging.LogLevelInfo),
	}

	// AddCapability capabilities with the global registry
	p.registerCapabilities()
	return p
}

// SetLogger configures the logger for the OpenAI provider.
// This is used for debugging and monitoring API interactions.
func (p *OpenAIProvider) SetLogger(logger logging.Logger) {
	p.logger = logger
}

// SetOption sets a specific option for the OpenAI provider.
// Supported options include:
//   - temperature: Controls randomness (0.0 to 2.0)
//   - max_tokens: Maximum tokens in the response (automatically converted to max_completion_tokens for "o" models)
//   - top_p: Nucleus sampling parameter
//   - frequency_penalty: Repetition reduction
//   - presence_penalty: Topic steering
//   - seed: Deterministic sampling seed
func (p *OpenAIProvider) SetOption(key string, value any) {
	// Handle max_tokens conversion for "o" models
	switch key {
	case openAIKeyMaxTokens:
		if p.needsMaxCompletionTokens() {
			// For models requiring max_completion_tokens, use that instead
			key = "max_completion_tokens"
			// Delete max_tokens if it was previously set
			delete(p.options, openAIKeyMaxTokens)
		} else {
			// For models using max_tokens, make sure max_completion_tokens is not set
			delete(p.options, "max_completion_tokens")
		}
	case "max_completion_tokens":
		// If explicitly setting max_completion_tokens, remove max_tokens to avoid conflicts
		delete(p.options, openAIKeyMaxTokens)
	}

	p.options[key] = value
	p.logger.Debug("Option set", "key", key, "value", value)
}

// SetDefaultOptions configures standard options from the global configuration.
// This includes temperature, max tokens, and sampling parameters.
func (p *OpenAIProvider) SetDefaultOptions(cfg *config.Config) {
	p.SetOption("temperature", cfg.Temperature)
	p.SetOption(openAIKeyMaxTokens, cfg.MaxTokens)
	if cfg.Seed != nil {
		p.SetOption("seed", *cfg.Seed)
	}
	p.logger.Debug(
		"Default options set",
		"temperature",
		cfg.Temperature,
		openAIKeyMaxTokens,
		cfg.MaxTokens,
		"seed",
		cfg.Seed,
	)
}

// Name returns "openai" as the provider identifier.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// registerCapabilities registers capabilities for all known OpenAI models
func (p *OpenAIProvider) registerCapabilities() {
	registry := GetCapabilityRegistry()

	// Define all known OpenAI models
	allModels := []string{
		// GPT-4o models
		"gpt-4o", "gpt-4o-mini", "gpt-4o-2024-11-20", "gpt-4o-2024-08-06", "gpt-4o-2024-05-13",
		"gpt-4o-mini-2024-07-18",

		// GPT-4 Turbo models
		"gpt-4-turbo", "gpt-4-turbo-2024-04-09", "gpt-4-turbo-preview", "gpt-4-0125-preview",
		"gpt-4-1106-preview", "gpt-4-turbo-vision-preview",

		// GPT-4 models
		"gpt-4", "gpt-4-0613", "gpt-4-0314", "gpt-4-vision-preview",

		// GPT-3.5 Turbo models
		"gpt-3.5-turbo", "gpt-3.5-turbo-0125", "gpt-3.5-turbo-1106", "gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k", "gpt-3.5-turbo-16k-0613",

		// O1 models
		"o1-preview", "o1-mini", "o1-preview-2024-09-12", "o1-mini-2024-09-12",
	}

	for _, model := range allModels {
		// O1 models have limited capabilities
		if strings.HasPrefix(model, "o1") {
			// Only register streaming for O1 models
			// Streaming registration handled below
			continue
		}

		// GPT-4o models - advanced structured response
		if strings.HasPrefix(model, "gpt-4o") || strings.HasPrefix(model, "gpt-4-turbo") {
			registry.RegisterCapability(ProviderOpenAI, model,
				modexv1.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, &modexv1.StructuredResponse{
					RequiresToolUse:  false,
					MaxSchemaDepth:   15,
					SupportedFormats: []modexv1.DataFormat{modexv1.DataFormat_DATA_FORMAT_JSON},
					RequiresJsonMode: true,
				})
		} else if strings.HasPrefix(model, "gpt-4") {
			// Regular GPT-4 models
			registry.RegisterCapability(ProviderOpenAI, model,
				modexv1.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, &modexv1.StructuredResponse{
					RequiresToolUse:  false,
					MaxSchemaDepth:   15,
					SupportedFormats: []modexv1.DataFormat{modexv1.DataFormat_DATA_FORMAT_JSON},
					RequiresJsonMode: true,
				})
		} else if strings.HasPrefix(model, "gpt-3.5-turbo") {
			// GPT-3.5 models - limited structured response support
			if model == "gpt-3.5-turbo-0125" || model == "gpt-3.5-turbo-1106" {
				registry.RegisterCapability(ProviderOpenAI, model,
					modexv1.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, &modexv1.StructuredResponse{
						RequiresToolUse:  false,
						MaxSchemaDepth:   10,
						SupportedFormats: []modexv1.DataFormat{modexv1.DataFormat_DATA_FORMAT_JSON},
						RequiresJsonMode: true,
					})
			}
		}

		// Function calling
		if strings.HasPrefix(model, "gpt-4") {
			registry.RegisterCapability(ProviderOpenAI, model, modexv1.CapabilityType_CAPABILITY_TYPE_FUNCTION_CALLING,
				&modexv1.FunctionCalling{
					MaxFunctions:      128,
					SupportsParallel:  true,
					MaxParallelCalls:  10,
					SupportsStreaming: true,
					RequiresToolRole:  false,
					SupportedParameterTypes: []modexv1.JsonSchemaType{
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxNestingDepth: 10,
				})
		} else if strings.HasPrefix(model, "gpt-3.5-turbo") {
			registry.RegisterCapability(ProviderOpenAI, model, modexv1.CapabilityType_CAPABILITY_TYPE_FUNCTION_CALLING,
				&modexv1.FunctionCalling{
					MaxFunctions:      64,
					SupportsParallel:  true,
					MaxParallelCalls:  5,
					SupportsStreaming: false,
					RequiresToolRole:  false,
					SupportedParameterTypes: []modexv1.JsonSchemaType{
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_OBJECT,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_ARRAY,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_STRING,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_NUMBER,
						modexv1.JsonSchemaType_JSON_SCHEMA_TYPE_BOOLEAN,
					},
					MaxNestingDepth: 10,
				})
		}

		// All OpenAI models support streaming (including O1 which was handled above)
		if !strings.HasPrefix(model, "o1") {
			registry.RegisterCapability(ProviderOpenAI, model, modexv1.CapabilityType_CAPABILITY_TYPE_STREAMING,
				&modexv1.Streaming{
					SupportsSse:    true,
					BufferSize:     4096,
					ChunkDelimiter: "data: ",
					SupportsUsage:  strings.HasPrefix(model, "gpt-4"),
				})
		}

		// Vision for specific models
		visionModels := []string{"gpt-4o", "gpt-4-turbo", "gpt-4-vision"}
		for _, vm := range visionModels {
			if strings.HasPrefix(model, vm) {
				registry.RegisterCapability(ProviderOpenAI, model, modexv1.CapabilityType_CAPABILITY_TYPE_VISION,
					&modexv1.Vision{
						MaxImageSizeBytes: 20 * 1024 * 1024,
						SupportedFormats: []modexv1.ImageFormat{
							modexv1.ImageFormat_IMAGE_FORMAT_JPEG, modexv1.ImageFormat_IMAGE_FORMAT_PNG,
							modexv1.ImageFormat_IMAGE_FORMAT_GIF, modexv1.ImageFormat_IMAGE_FORMAT_WEBP,
						},
						MaxImagesPerRequest:     10,
						SupportsVideoFrames:     strings.Contains(model, "4o"),
						SupportsImageGeneration: false,
						SupportsOcr:             true,
						SupportsObjectDetection: false,
					})
				break
			}
		}
	}
}

// HasCapability checks if a capability is supported
func (p *OpenAIProvider) HasCapability(capability modexv1.CapabilityType, model string) bool {
	targetModel := p.model
	if model != "" {
		targetModel = model
	}
	return GetCapabilityRegistry().HasCapability(ProviderOpenAI, targetModel, capability)
}

// Endpoint returns the OpenAI API endpoint URL.
// For API version 1, this is "https://api.openai.com/v1/chat/completions".
func (p *OpenAIProvider) Endpoint() string {
	return "https://api.openai.com/v1/chat/completions"
}

// Headers returns the required HTTP headers for OpenAI API requests.
// This includes:
//   - Authorization: Bearer token using the API key
//   - Content-Type: application/json
//   - Any additional headers specified via SetExtraHeaders
func (p *OpenAIProvider) Headers() map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}

	for key, value := range p.extraHeaders {
		headers[key] = value
	}

	p.logger.Debug("Headers prepared", "headers", headers)
	return headers
}

// PrepareRequest creates the request body for an OpenAI API call
func (p *OpenAIProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeOpenAIRequestWithModel(model)

	// Handle system prompt from Request or options
	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	if systemPrompt != "" {
		p.addSystemPromptToRequestBody(requestBody, systemPrompt)
	}

	// Add messages from the Request
	p.addMessagesToRequestBody(requestBody, req.Messages, options)

	// Handle tools if present in options
	p.handleToolsForRequest(requestBody, options)

	// Handle structured response schema
	if req.ResponseSchema != nil && p.HasCapability(modexv1.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
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

// ParseResponse extracts the generated text from the OpenAI API response.
// It handles various response formats and error cases.
func (p *OpenAIProvider) ParseResponse(body []byte) (*Response, error) {
	response := openAIResponse{}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("empty response from API")
	}

	usage := &Usage{}

	if response.Usage != nil && response.Usage.PromptTokensDetails != nil {
		usage = NewUsage(
			response.Usage.PromptTokens,
			response.Usage.PromptTokensDetails.CacheTokens,
			response.Usage.CompletionTokens,
			0,
			0, // ReasoningTokens
		)
	}

	message := response.Choices[0].Message
	if message.Content != "" {
		return &Response{
			Content: Text{message.Content},
			Usage:   usage,
		}, nil
	}

	if len(message.ToolCalls) > 0 {
		var functionCalls []string
		for _, call := range message.ToolCalls {
			// Parse arguments as raw JSON to preserve the exact format
			var args any
			if err := json.Unmarshal(call.Function.Arguments, &args); err != nil {
				return nil, fmt.Errorf("error parsing function arguments: %w", err)
			}

			functionCall, err := FormatFunctionCall(call.Function.Name, args)
			if err != nil {
				return nil, fmt.Errorf("error formatting function call: %w", err)
			}
			functionCalls = append(functionCalls, functionCall)
		}

		return &Response{
			Content: Text{strings.Join(functionCalls, "\n")},
			Usage:   usage,
		}, nil
	}

	return nil, errors.New("no content or tool calls in response")
}

// SetExtraHeaders configures additional HTTP headers for API requests.
// This allows for custom headers needed for specific features or requirements.
func (p *OpenAIProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.extraHeaders = extraHeaders
	p.logger.Debug("Extra headers set", "headers", extraHeaders)
}

// PrepareStreamRequest creates a request body for streaming API calls
func (p *OpenAIProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	if !p.HasCapability(modexv1.CapabilityType_CAPABILITY_TYPE_STREAMING, model) {
		return nil, errors.New("streaming is not supported by this provider")
	}

	requestBody := p.initializeOpenAIRequestWithModel(model)
	requestBody[openAIKeyStream] = true
	requestBody["stream_options"] = map[string]bool{"include_usage": true}

	// Handle system prompt from Request or options
	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	if systemPrompt != "" {
		p.addSystemPromptToRequestBody(requestBody, systemPrompt)
	}

	// Add messages from the Request
	p.addMessagesToRequestBody(requestBody, req.Messages, options)

	// Handle tools if present in options
	p.handleToolsForRequest(requestBody, options)

	// Handle structured response schema
	if req.ResponseSchema != nil && p.HasCapability(modexv1.CapabilityType_CAPABILITY_TYPE_STRUCTURED_RESPONSE, model) {
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

// ParseStreamResponse processes a single chunk from a streaming response
func (p *OpenAIProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
	// Skip empty lines
	if len(bytes.TrimSpace(chunk)) == 0 {
		return nil, errors.New("empty chunk")
	}

	// Check for [DONE] marker
	if bytes.Equal(bytes.TrimSpace(chunk), []byte("[DONE]")) {
		return nil, io.EOF
	}

	// Parse the chunk
	response := openAIStreamResponse{}

	if err := json.Unmarshal(chunk, &response); err != nil {
		return nil, fmt.Errorf("malformed response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	// Handle finish reason
	if response.Choices[0].FinishReason != "" {
		return nil, io.EOF
	}

	// Skip role-only messages
	if response.Choices[0].Delta.Role != "" && response.Choices[0].Delta.Content == "" {
		return nil, errors.New("skip token")
	}

	usage := &Usage{}

	if response.Usage != nil && response.Usage.PromptTokensDetails != nil {
		usage = NewUsage(
			response.Usage.PromptTokens,
			response.Usage.PromptTokensDetails.CacheTokens,
			response.Usage.CompletionTokens,
			0,
			0, // ReasoningTokens
		)
	}

	return &Response{
		Content: Text{
			response.Choices[0].Delta.Content,
		},
		Usage: usage,
	}, nil
}

// needsMaxCompletionTokens checks if the model requires max_completion_tokens instead of max_tokens
func (p *OpenAIProvider) needsMaxCompletionTokens() bool {
	if strings.HasPrefix(p.model, "o") {
		return true
	}

	if strings.Contains(p.model, "4o") || strings.Contains(p.model, "-o") {
		return true
	}

	return false
}

// initializeOpenAIRequestWithModel creates the base request structure with specified model
func (p *OpenAIProvider) initializeOpenAIRequestWithModel(model string) map[string]any {
	return map[string]any{
		"model":    model,
		"messages": []map[string]any{},
	}
}

// extractSystemPromptFromRequest gets system prompt from request or options
func (p *OpenAIProvider) extractSystemPromptFromRequest(req *Request, options map[string]any) string {
	// Priority: Request.SystemPrompt > options["system_prompt"]
	if req.SystemPrompt != "" {
		return req.SystemPrompt
	}
	if sp, ok := options[openAIKeySystemPrompt].(string); ok && sp != "" {
		return sp
	}
	return ""
}

// addSystemPromptToRequestBody adds the system prompt to the request
func (p *OpenAIProvider) addSystemPromptToRequestBody(requestBody map[string]any, systemPrompt string) {
	if systemPrompt == "" {
		return
	}

	if messages, ok := requestBody["messages"].([]map[string]any); ok {
		systemMessage := map[string]any{
			"role":    "system",
			"content": systemPrompt,
		}
		requestBody["messages"] = append([]map[string]any{systemMessage}, messages...)
	}
}

// addMessagesToRequestBody converts and adds messages to the request
func (p *OpenAIProvider) addMessagesToRequestBody(
	requestBody map[string]any,
	messages []Message,
	options map[string]any,
) {
	if len(messages) == 0 {
		return
	}

	openAIMessages := make([]map[string]any, 0, len(messages))
	for i := range messages {
		openAIMsg := p.convertMessageToOpenAIFormat(&messages[i], options)
		openAIMessages = append(openAIMessages, openAIMsg)
	}

	if existingMessages, ok := requestBody["messages"].([]map[string]any); ok {
		requestBody["messages"] = append(existingMessages, openAIMessages...)
	} else {
		requestBody["messages"] = openAIMessages
	}
}

// convertMessageToOpenAIFormat converts a Message to OpenAI's format
func (p *OpenAIProvider) convertMessageToOpenAIFormat(msg *Message, _ map[string]any) map[string]any {
	openAIMsg := map[string]any{
		"role":    msg.Role,
		"content": msg.Content,
	}

	// Add name if present
	if msg.Name != "" {
		openAIMsg["name"] = msg.Name
	}

	// Add tool_call_id if present
	if msg.ToolCallID != "" {
		openAIMsg["tool_call_id"] = msg.ToolCallID
	}

	// Handle tool calls if present
	if len(msg.ToolCalls) > 0 {
		openAIToolCalls := make([]map[string]any, len(msg.ToolCalls))
		for i, toolCall := range msg.ToolCalls {
			openAIToolCalls[i] = map[string]any{
				"id":   toolCall.ID,
				"type": toolCall.Type,
				"function": map[string]any{
					"name":      toolCall.Function.Name,
					"arguments": string(toolCall.Function.Arguments),
				},
			}
		}
		openAIMsg["tool_calls"] = openAIToolCalls
	}

	return openAIMsg
}

// handleToolsForRequest processes tools and adds them to the request
func (p *OpenAIProvider) handleToolsForRequest(requestBody map[string]any, options map[string]any) {
	tools, ok := options[openAIKeyTools].([]models.Tool)
	if !ok || len(tools) == 0 {
		return
	}

	// Convert tools to OpenAI format
	openAITools := make([]map[string]any, len(tools))
	for i, tool := range tools {
		openAITools[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
			"strict": true,
		}
	}

	requestBody["tools"] = openAITools

	// Handle tool_choice if specified
	if toolChoice, ok := options[openAIKeyToolChoice].(string); ok {
		requestBody["tool_choice"] = toolChoice
	}
}

// addStructuredResponseToRequest adds structured response schema to the request
func (p *OpenAIProvider) addStructuredResponseToRequest(requestBody map[string]any, schema any) {
	// For OpenAI, we use response_format with JSON schema
	requestBody["response_format"] = map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "response",
			"schema": schema,
			"strict": true,
		},
	}
}

// addRemainingOptions adds any remaining options to the request body
func (p *OpenAIProvider) addRemainingOptions(requestBody map[string]any, options map[string]any) {
	// Create merged options excluding handled keys
	mergedOptions := make(map[string]any)

	// Add provider options first
	for k, v := range p.options {
		if !p.isGlobalOption(k) {
			mergedOptions[k] = v
		}
	}

	// Add request options (may override provider options)
	for k, v := range options {
		if !p.isGlobalOption(k) {
			mergedOptions[k] = v
		}
	}

	// Handle token parameters
	p.handleTokenParameters(mergedOptions)

	// Add all merged options to request body
	for k, v := range mergedOptions {
		requestBody[k] = v
	}
}

// isGlobalOption checks if a key is handled globally and should be excluded from options
func (p *OpenAIProvider) isGlobalOption(key string) bool {
	return key == openAIKeySystemPrompt ||
		key == openAIKeyTools ||
		key == openAIKeyToolChoice ||
		key == openAIKeyStructuredMessages ||
		key == openAIKeyStream
}

// handleTokenParameters handles max_tokens/max_completion_tokens conflict
func (p *OpenAIProvider) handleTokenParameters(mergedOptions map[string]any) {
	// For models that need max_completion_tokens, ensure we use that and not max_tokens
	if p.needsMaxCompletionTokens() {
		if _, hasMaxTokens := mergedOptions[openAIKeyMaxTokens]; hasMaxTokens {
			// Move max_tokens value to max_completion_tokens
			mergedOptions["max_completion_tokens"] = mergedOptions[openAIKeyMaxTokens]
			delete(mergedOptions, openAIKeyMaxTokens)
		}
	} else {
		// For other models, ensure we use max_tokens and not max_completion_tokens
		if _, hasMaxCompletionTokens := mergedOptions["max_completion_tokens"]; hasMaxCompletionTokens {
			// Move max_completion_tokens value to max_tokens
			mergedOptions[openAIKeyMaxTokens] = mergedOptions["max_completion_tokens"]
			delete(mergedOptions, "max_completion_tokens")
		}
	}
}

type openAIResponse struct {
	Usage   *openAIUsage   `json:"usage"`
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message *openAIMessage `json:"message"`
}

type openAIMessage struct {
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

type openAIToolCall struct {
	Function *openAIFunction `json:"function"`
	ID       string          `json:"id"`
	Type     string          `json:"type"`
}

type openAIFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
type openAIUsage struct {
	PromptTokensDetails     *openAIPromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails *openAICompletionTokensDetails `json:"completion_tokens_details"`
	PromptTokens            int64                          `json:"prompt_tokens"`
	CompletionTokens        int64                          `json:"completion_tokens"`
	TotalTokens             int64                          `json:"total_tokens"`
}

type openAIPromptTokensDetails struct {
	CacheTokens int64 `json:"cache_tokens"`
	AudioTokens int64 `json:"audio_tokens"`
}

type openAICompletionTokensDetails struct {
	ReasoningTokens          int64 `json:"reasoning_tokens"`
	AudioTokens              int64 `json:"audio_tokens"`
	AcceptedPredictionTokens int64 `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int64 `json:"rejected_prediction_tokens"`
}

type openAIStreamResponse struct {
	Usage   *openAIUsage         `json:"usage,omitempty"`
	Choices []openAIStreamChoice `json:"choices"`
}

type openAIStreamChoice struct {
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
