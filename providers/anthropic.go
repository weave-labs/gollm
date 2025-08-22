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
)

// Common parameter keys
const (
	anthropicKeyMaxTokens     = "max_tokens"
	anthropicKeyStream        = "stream"
	anthropicKeySystemPrompt  = "system_prompt"
	anthropicKeyTools         = "tools"
	anthropicKeyToolChoice    = "tool_choice"
	anthropicKeyEnableCaching = "enable_caching"
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude API.
// It supports Claude models and provides access to Anthropic's language model capabilities,
// including structured output and system prompts.
type AnthropicProvider struct {
	logger       logging.Logger
	extraHeaders map[string]string
	options      map[string]any
	apiKey       string
	model        string
}

// NewAnthropicProvider creates a new Anthropic provider instance.
// It initializes the provider with the given API key, model, and optional headers.
func NewAnthropicProvider(apiKey, model string, extraHeaders map[string]string) *AnthropicProvider {
	p := &AnthropicProvider{
		apiKey:       apiKey,
		model:        model,
		extraHeaders: make(map[string]string),
		options:      make(map[string]any),
		logger:       logging.NewLogger(logging.LogLevelInfo),
	}

	for k, v := range extraHeaders {
		p.extraHeaders[k] = v
	}

	if _, exists := p.extraHeaders["anthropic-beta"]; !exists {
		p.extraHeaders["anthropic-beta"] = "prompt-caching-2024-07-31"
	}

	// Register capabilities based on model
	p.registerCapabilities()
	return p
}

// SetLogger configures the logger for the Anthropic provider.
// This is used for debugging and monitoring API interactions.
func (p *AnthropicProvider) SetLogger(logger logging.Logger) {
	p.logger = logger
}

// SetOption sets a specific option for the Anthropic provider.
// Supported options include:
//   - temperature: Controls randomness (0.0 to 1.0)
//   - max_tokens: Maximum tokens in the response
//   - top_p: Nucleus sampling parameter
//   - top_k: Top-k sampling parameter
//   - stop_sequences: Custom stop sequences
func (p *AnthropicProvider) SetOption(key string, value any) {
	p.options[key] = value
}

// SetDefaultOptions configures standard options from the global configuration.
// This includes temperature, max tokens, and sampling parameters.
func (p *AnthropicProvider) SetDefaultOptions(cfg *config.Config) {
	p.SetOption("temperature", cfg.Temperature)
	p.SetOption(anthropicKeyMaxTokens, cfg.MaxTokens)
	if cfg.Seed != nil {
		p.SetOption("seed", *cfg.Seed)
	}
}

// Name returns "anthropic" as the provider identifier.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// registerCapabilities registers capabilities for all known Anthropic models
func (p *AnthropicProvider) registerCapabilities() {
	registry := GetRegistry()

	// Define all known Anthropic Claude models
	allModels := []string{
		// Claude 3.5 models
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022",

		// Claude 3 models
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",

		// Legacy Claude models
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",

		// Generic model names (latest)
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
	}

	for _, model := range allModels {
		// All Claude models support structured responses
		registry.Register(ProviderAnthropic, model, CapStructuredResponse, StructuredResponseConfig{
			RequiresToolUse:  false,
			MaxSchemaDepth:   15,
			SupportedFormats: []string{"json"},
			SystemPromptHint: "You must respond with a JSON object that strictly adheres to this schema",
			RequiresJSONMode: false,
		})

		// All Claude models support function calling
		registry.Register(ProviderAnthropic, model, CapFunctionCalling, FunctionCallingConfig{
			MaxFunctions:      64,
			SupportsParallel:  true,
			MaxParallelCalls:  10,
			RequiresToolRole:  false,
			SupportsStreaming: true,
		})

		// All Claude models support streaming
		registry.Register(ProviderAnthropic, model, CapStreaming, StreamingConfig{
			SupportsSSE:    true,
			BufferSize:     4096,
			ChunkDelimiter: "data: ",
			SupportsUsage:  true,
		})

		// Claude 3+ models support caching (legacy models have limited support)
		if strings.Contains(model, "claude-3") {
			registry.Register(ProviderAnthropic, model, CapCaching, CachingConfig{
				MaxCacheSize:     1024 * 1024, // 1MB
				CacheTTLSeconds:  3600,        // 1 hour
				CacheKeyStrategy: "ephemeral",
			})
		} else {
			// Legacy models have basic caching
			registry.Register(ProviderAnthropic, model, CapCaching, CachingConfig{
				MaxCacheSize:     512 * 1024, // 512KB
				CacheTTLSeconds:  1800,       // 30 minutes
				CacheKeyStrategy: "ephemeral",
			})
		}

		// Vision capability for Claude 3+ models
		if strings.Contains(model, "claude-3") {
			registry.Register(ProviderAnthropic, model, CapVision, VisionConfig{
				MaxImageSize:        5 * 1024 * 1024, // 5MB
				SupportedFormats:    []string{"jpeg", "png", "gif", "webp"},
				MaxImagesPerRequest: 20,
			})
		}

		// System prompt support for all models
		registry.Register(ProviderAnthropic, model, CapSystemPrompt, SystemPromptConfig{
			MaxLength:        32768,
			SupportsMultiple: true,
		})
	}
}

// HasCapability checks if a capability is supported
func (p *AnthropicProvider) HasCapability(capability Capability, model string) bool {
	targetModel := p.model
	if model != "" {
		targetModel = model
	}
	return GetRegistry().HasCapability(ProviderAnthropic, targetModel, capability)
}

// Endpoint returns the Anthropic API endpoint URL.
// For API version 2024-02-15, this is "https://api.anthropic.com/v1/messages".
func (p *AnthropicProvider) Endpoint() string {
	return "https://api.anthropic.com/v1/messages"
}

// Headers return the required HTTP headers for Anthropic API requests.
// This includes:
//   - x-api-key: API key for authentication
//   - anthropic-version: API version identifier
//   - Content-Type: application/json
//   - Any additional headers specified via SetExtraHeaders
func (p *AnthropicProvider) Headers() map[string]string {
	headers := map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
		"anthropic-beta":    "prompt-caching-2024-07-31",
	}
	return headers
}

// PrepareRequest creates the request body for an Anthropic API call
func (p *AnthropicProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	requestBody := p.initializeRequestBodyWithModel(model)

	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	systemPrompt = p.handleToolsForRequest(requestBody, systemPrompt, options)
	p.addSystemPromptToRequestBody(requestBody, systemPrompt)

	p.addMessagesToRequestBody(requestBody, req.Messages, options)

	if req.ResponseSchema != nil && p.HasCapability(CapStructuredResponse, model) {
		err := p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to add structured response: %w", err)
		}
	}

	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// PrepareStreamRequest creates a request body for streaming API calls
func (p *AnthropicProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
	// Determine which model to use
	model := p.model
	if req.Model != "" {
		model = req.Model
	} else if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	if !p.HasCapability(CapStreaming, model) {
		return nil, errors.New("streaming is not supported by this provider")
	}
	requestBody := p.initializeRequestBodyWithModel(model)
	requestBody[anthropicKeyStream] = true

	systemPrompt := p.extractSystemPromptFromRequest(req, options)
	systemPrompt = p.handleToolsForRequest(requestBody, systemPrompt, options)
	p.addSystemPromptToRequestBody(requestBody, systemPrompt)

	p.addMessagesToRequestBody(requestBody, req.Messages, options)

	if req.ResponseSchema != nil && p.HasCapability(CapStructuredResponse, model) {
		err := p.addStructuredResponseToRequest(requestBody, req.ResponseSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to add structured response: %w", err)
		}
	}

	p.addRemainingOptions(requestBody, options)

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return data, nil
}

// ParseResponse extracts the generated text from the Anthropic API response.
// It handles various response formats and error cases.
func (p *AnthropicProvider) ParseResponse(body []byte) (*Response, error) {
	p.logger.Debug("Raw API anthropicResponse: %s", string(body))

	anthropicResponse := anthropicResponse{}

	if err := json.Unmarshal(body, &anthropicResponse); err != nil {
		return nil, fmt.Errorf("error parsing anthropicResponse: %w", err)
	}
	if len(anthropicResponse.Content) == 0 {
		return nil, errors.New("empty anthropicResponse from LLM")
	}

	p.logger.Debug("Number of content blocks: %d", len(anthropicResponse.Content))
	p.logger.Debug("Stop reason: %s", anthropicResponse.StopReason)

	// Process content blocks
	result, err := p.processAnthropicContent(anthropicResponse.Content)
	if err != nil {
		return nil, err
	}

	p.logger.Debug("Final anthropicResponse: %s", result)

	response := &Response{
		Content: Text{result},
		Usage: NewUsage(
			anthropicResponse.Usage.InputTokens,
			anthropicResponse.Usage.CacheCreationInputTokens,
			anthropicResponse.Usage.OutputTokens,
			0,
			anthropicResponse.Usage.CacheReadInputTokens,
		),
	}

	return response, nil
}

// ParseStreamResponse processes single SSE JSON "data:" payload from Anthropic Messages streaming.
// It returns either a text Content token, a Usage-only token, io.EOF for message_stop, or "skip token".
func (p *AnthropicProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
	// Skip empty lines
	if len(bytes.TrimSpace(chunk)) == 0 {
		return nil, errors.New("empty chunk")
	}
	// [DONE] guard (if your decoder ever passes this through)
	if bytes.Equal(bytes.TrimSpace(chunk), []byte("[DONE]")) {
		return nil, io.EOF
	}

	var ev anthropicEvent
	if err := json.Unmarshal(chunk, &ev); err != nil {
		return nil, fmt.Errorf("malformed event: %w", err)
	}

	switch ev.Type {
	case "content_block_delta":
		// Only emit text deltas as tokens
		if ev.Delta != nil && ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
			return &Response{
				Content: Text{Value: ev.Delta.Text},
			}, nil
		}
		return nil, errors.New("skip token")

	case "message_start":
		// Usage may be present on the embedded message
		if ev.Message != nil && ev.Message.Usage != nil {
			return &Response{
				Usage: NewUsage(
					ev.Message.Usage.InputTokens,
					ev.Message.Usage.CacheCreationInputTokens,
					ev.Message.Usage.OutputTokens,
					0,
					ev.Message.Usage.CacheReadInputTokens,
				),
			}, nil
		}
		return nil, errors.New("skip token")

	case "message_delta":
		// Usage may be present at the top level; counts are cumulative
		if ev.Usage != nil {
			return &Response{
				Usage: NewUsage(
					ev.Usage.InputTokens,
					ev.Usage.CacheCreationInputTokens,
					ev.Usage.OutputTokens,
					0,
					ev.Usage.CacheReadInputTokens,
				),
			}, nil
		}
		return nil, errors.New("skip token")

	case "message_stop":
		return nil, io.EOF

	// Ignore pings, starts/stops of blocks, tool JSON partials, thinking/signature, etc.
	default:
		return nil, errors.New("skip token")
	}
}

// SetExtraHeaders configures additional HTTP headers for API requests.
// This allows for custom headers needed for specific features or requirements.
func (p *AnthropicProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.extraHeaders = extraHeaders
}

// initializeRequestBodyWithModel creates the base request structure with specified model
func (p *AnthropicProvider) initializeRequestBodyWithModel(model string) map[string]any {
	return map[string]any{
		"model":               model,
		anthropicKeyMaxTokens: p.options[anthropicKeyMaxTokens],
		"system":              []map[string]any{},
		"messages":            []map[string]any{},
	}
}

// extractSystemPromptFromRequest gets system prompt from request or options
func (p *AnthropicProvider) extractSystemPromptFromRequest(req *Request, options map[string]any) string {
	// Priority: Request.SystemPrompt > options["system_prompt"]
	if req.SystemPrompt != "" {
		return req.SystemPrompt
	}
	if sp, ok := options["system_prompt"].(string); ok && sp != "" {
		return sp
	}
	return ""
}

// handleToolsForRequest processes tools and updates system prompt if needed
func (p *AnthropicProvider) handleToolsForRequest(
	requestBody map[string]any,
	systemPrompt string,
	options map[string]any,
) string {
	tools, ok := options[anthropicKeyTools].([]models.Tool)
	if !ok || len(tools) == 0 {
		return systemPrompt
	}
	return p.processTools(tools, requestBody, systemPrompt, options)
}

// addSystemPromptToRequestBody adds the system prompt to the request
func (p *AnthropicProvider) addSystemPromptToRequestBody(requestBody map[string]any, systemPrompt string) {
	if systemPrompt == "" {
		return
	}

	parts := splitSystemPrompt(systemPrompt, AnthropicSystemPromptMaxParts)
	for i, part := range parts {
		systemMessage := map[string]any{
			"type": "text",
			"text": part,
		}
		if i > 0 {
			systemMessage["cache_control"] = map[string]string{"type": "ephemeral"}
		}
		if systemArray, ok := requestBody["system"].([]map[string]any); ok {
			requestBody["system"] = append(systemArray, systemMessage)
		}
	}
}

// addStructuredResponseToRequest adds structured response schema to the request
func (p *AnthropicProvider) addStructuredResponseToRequest(requestBody map[string]any, schema any) error {
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// For Anthropic, we add the schema to the system prompt
	systemInstruction := fmt.Sprintf(
		"You must respond with a JSON object that strictly adheres to this schema:\n%s\nDo not include any explanatory text, only output valid JSON.",
		string(schemaJSON),
	)

	// Append to existing system prompt if present
	if existing, ok := requestBody["system"].([]map[string]any); ok && len(existing) > 0 {
		existing = append(existing, map[string]any{
			"type": "text",
			"text": systemInstruction,
		})
		requestBody["system"] = existing
	} else {
		requestBody["system"] = []map[string]any{
			{
				"type": "text",
				"text": systemInstruction,
			},
		}
	}

	return nil
}

// addMessagesToRequestBody converts and adds messages to the request
func (p *AnthropicProvider) addMessagesToRequestBody(
	requestBody map[string]any,
	messages []Message,
	options map[string]any,
) {
	anthropicMessages := make([]map[string]any, 0, len(messages))

	for i := range messages {
		anthropicMsg := p.convertMessageToAnthropicFormat(&messages[i], options)
		anthropicMessages = append(anthropicMessages, anthropicMsg)
	}

	requestBody["messages"] = anthropicMessages
}

// convertMessageToAnthropicFormat converts a Message to Anthropic's format
func (p *AnthropicProvider) convertMessageToAnthropicFormat(msg *Message, options map[string]any) map[string]any {
	// Create content array
	content := []map[string]any{
		{
			"type": "text",
			"text": msg.Content,
		},
	}

	// Add cache control if specified
	if msg.CacheType != "" || p.shouldEnableCaching(options) {
		cacheType := string(msg.CacheType)
		if cacheType == "" {
			cacheType = "ephemeral"
		}
		content[0]["cache_control"] = map[string]string{"type": cacheType}
	}

	// Handle tool calls if present
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    toolCall.ID,
				"name":  toolCall.Function.Name,
				"input": toolCall.Function.Arguments,
			})
		}
	}

	anthropicMsg := map[string]any{
		"role":    msg.Role,
		"content": content,
	}

	// Add name if present
	if msg.Name != "" {
		anthropicMsg["name"] = msg.Name
	}

	return anthropicMsg
}

// shouldEnableCaching checks if caching should be enabled
func (p *AnthropicProvider) shouldEnableCaching(options map[string]any) bool {
	if caching, ok := options["enable_caching"].(bool); ok {
		return caching
	}
	return false
}

// addRemainingOptions adds non-handled options to the request
func (p *AnthropicProvider) addRemainingOptions(requestBody map[string]any, options map[string]any) {
	for k, v := range options {
		if p.isGlobalOption(k) {
			continue
		}
		requestBody[k] = v
	}
}

// isGlobalOption checks if an option is already handled
func (p *AnthropicProvider) isGlobalOption(key string) bool {
	return key == anthropicKeySystemPrompt ||
		key == anthropicKeyMaxTokens ||
		key == anthropicKeyTools ||
		key == anthropicKeyToolChoice ||
		key == anthropicKeyEnableCaching
}

// processTools handles tool configuration and updates system prompt
func (p *AnthropicProvider) processTools(
	tools []models.Tool,
	requestBody map[string]any,
	systemPrompt string,
	options map[string]any,
) string {
	anthropicTools := make([]map[string]any, len(tools))
	for i, tool := range tools {
		anthropicTools[i] = map[string]any{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
	}
	requestBody[anthropicKeyTools] = anthropicTools

	// Add tool usage instructions to system prompt for multiple tools
	if len(tools) > 1 {
		toolUsagePrompt := "When multiple tools are needed to answer a question, you should identify all required tools upfront and use them all at once in your response, rather than using them sequentially. Do not wait for tool results before calling other tools."
		if systemPrompt != "" {
			systemPrompt = toolUsagePrompt + "\n\n" + systemPrompt
		} else {
			systemPrompt = toolUsagePrompt
		}
	}

	// Set tool choice
	if toolChoice, ok := options["tool_choice"].(string); ok {
		requestBody["tool_choice"] = map[string]any{
			"type": toolChoice,
		}
	} else {
		// Default to auto for tool choice when tools are provided
		requestBody["tool_choice"] = map[string]any{
			"type": "auto",
		}
	}

	return systemPrompt
}

// Helper function to split the system prompt into a maximum of n parts
func splitSystemPrompt(prompt string, n int) []string {
	if n <= 1 {
		return []string{prompt}
	}

	// Split the prompt into paragraphs
	paragraphs := strings.Split(prompt, "\n\n")

	if len(paragraphs) <= n {
		return paragraphs
	}

	// If we have more paragraphs than allowed parts, we need to combine some
	result := make([]string, n)
	paragraphsPerPart := len(paragraphs) / n
	extraParagraphs := len(paragraphs) % n

	currentIndex := 0
	for i := range n {
		end := currentIndex + paragraphsPerPart
		if i < extraParagraphs {
			end++
		}
		result[i] = strings.Join(paragraphs[currentIndex:end], "\n\n")
		currentIndex = end
	}

	return result
}

// processAnthropicContent processes the content blocks from Anthropic response
func (p *AnthropicProvider) processAnthropicContent(contents []anthropicContent) (string, error) {
	var finalResponse strings.Builder
	var functionCalls []string
	var pendingText strings.Builder
	var lastType string

	// First pass: collect all function calls and text
	for i, content := range contents {
		p.logger.Debug("Processing content block %d: type=%s", i, content.Type)

		switch content.Type {
		case "text":
			p.processTextContent(&pendingText, content.Text, lastType)
			p.logger.Debug("Added text content: %s", content.Text)

		case "tool_use", "tool_calls":
			// Transfer pending text to final response
			p.transferPendingText(&finalResponse, &pendingText)

			// Process function call
			functionCall, err := p.processFunctionCall(&content)
			if err != nil {
				return "", err
			}
			functionCalls = append(functionCalls, functionCall)
			p.logger.Debug("Added function call: %s", functionCall)
		}
		lastType = content.Type
	}

	// Add any remaining pending text
	p.transferPendingText(&finalResponse, &pendingText)

	p.logger.Debug("Number of function calls collected: %d", len(functionCalls))
	for i, call := range functionCalls {
		p.logger.Debug("Function call %d: %s", i, call)
	}

	// Add all function calls at the end
	if len(functionCalls) > 0 {
		if finalResponse.Len() > 0 {
			finalResponse.WriteString("\n")
		}
		finalResponse.WriteString(strings.Join(functionCalls, "\n"))
	}

	return finalResponse.String(), nil
}

// processTextContent handles text content blocks
func (p *AnthropicProvider) processTextContent(pendingText *strings.Builder, text string, lastType string) {
	// If we have pending text and this is also text, add a space
	if lastType == "text" && pendingText.Len() > 0 {
		pendingText.WriteString(" ")
	}
	pendingText.WriteString(text)
}

// transferPendingText transfers pending text to final response
func (p *AnthropicProvider) transferPendingText(finalResponse, pendingText *strings.Builder) {
	if pendingText.Len() > 0 {
		if finalResponse.Len() > 0 {
			finalResponse.WriteString("\n")
		}
		finalResponse.WriteString(pendingText.String())
		pendingText.Reset()
	}
}

// processFunctionCall processes a function call content block
func (p *AnthropicProvider) processFunctionCall(content *anthropicContent) (string, error) {
	// Parse input as raw JSON to preserve the exact format
	var args any
	if err := json.Unmarshal(content.Input, &args); err != nil {
		p.logger.Debug("Error parsing tool input: %v, raw input: %s", err, string(content.Input))
		return "", fmt.Errorf("error parsing tool input: %w", err)
	}

	functionCall, err := FormatFunctionCall(content.Name, args)
	if err != nil {
		p.logger.Debug("Error formatting function call: %v", err)
		return "", fmt.Errorf("error formatting function call: %w", err)
	}

	return functionCall, nil
}

// anthropicResponse represents the structure of a response from the Anthropic API.
// nolint: tagliatelle // These types are specific to the Anthropic API response structure
type anthropicResponse struct {
	StopSeq    *string            `json:"stop_sequence"`
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Content    []anthropicContent `json:"content"`
	Usage      anthropicUsage     `json:"usage"`
}

// anthropicContent represents a single content block in an Anthropic response.
type anthropicContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicEvent struct {
	Index   *int              `json:"index,omitempty"`
	Delta   *anthropicDelta   `json:"delta,omitempty"`
	Usage   *anthropicUsage   `json:"usage,omitempty"`
	Message *anthropicMessage `json:"message,omitempty"`
	Type    string            `json:"type"`
}

type anthropicMessage struct {
	StopReason   *string         `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        *anthropicUsage `json:"usage,omitempty"`
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Model        string          `json:"model"`
	Content      []any           `json:"content"`
}

type anthropicDelta struct {
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
	Type         string  `json:"type,omitempty"`
	Text         string  `json:"text,omitempty"`
	PartialJSON  string  `json:"partial_json,omitempty"`
	Thinking     string  `json:"thinking,omitempty"`
	Signature    string  `json:"signature,omitempty"`
}

type anthropicUsage struct {
	InputTokens              int64 `json:"input_tokens,omitempty"`
	OutputTokens             int64 `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
}
