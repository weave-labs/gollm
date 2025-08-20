Absolutely! Let's redesign this with composition and make capability checking a first-class citizen of the Provider interface:

## Redesigned Option 3 with Composition

### 1. Core Types and Provider Interface

```go
// provider_interface.go
package providers

// Provider interface with built-in capability checking
type Provider interface {
    // Core identification
    Name() ProviderName
    Model() string
    
    // Core methods
    PrepareRequest(req *Request, options map[string]any) ([]byte, error)
    PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error)
    ParseResponse(body []byte) (*Response, error)
    ParseStreamResponse(chunk []byte) (*Response, error)
    
    // Capability checking - first class citizens!
    HasCapability(cap Capability) bool
    GetStructuredResponseConfig() (StructuredResponseConfig, bool)
    GetFunctionCallingConfig() (FunctionCallingConfig, bool)
    GetStreamingConfig() (StreamingConfig, bool)
    GetVisionConfig() (VisionConfig, bool)
    GetCachingConfig() (CachingConfig, bool)
}

// ProviderName ensures we use correct provider identifiers
type ProviderName string

const (
    ProviderOpenAI    ProviderName = "openai"
    ProviderGemini    ProviderName = "gemini"
    ProviderAnthropic ProviderName = "anthropic"
    ProviderCohere    ProviderName = "cohere"
    ProviderOllama    ProviderName = "ollama"
    ProviderOpenRouter ProviderName = "openrouter"
)

// Capability represents a feature that a provider/model may support
type Capability string

const (
    CapStructuredResponse Capability = "structured_response"
    CapStreaming         Capability = "streaming"
    CapFunctionCalling   Capability = "function_calling"
    CapVision           Capability = "vision"
    CapToolUse          Capability = "tool_use"
    CapSystemPrompt     Capability = "system_prompt"
    CapCaching          Capability = "caching"
)
```

### 2. Capability Checker Component

```go
// capability_checker.go
package providers

// CapabilityChecker is a component that provides capability checking
// This can be composed into any provider implementation
type CapabilityChecker struct {
    provider ProviderName
    model    string
}

// NewCapabilityChecker creates a new capability checker
func NewCapabilityChecker(provider ProviderName, model string) *CapabilityChecker {
    return &CapabilityChecker{
        provider: provider,
        model:    model,
    }
}

// HasCapability checks if a capability is supported
func (cc *CapabilityChecker) HasCapability(cap Capability) bool {
    return Registry.HasCapability(cc.provider, cc.model, cap)
}

// GetStructuredResponseConfig returns typed structured response config
func (cc *CapabilityChecker) GetStructuredResponseConfig() (StructuredResponseConfig, bool) {
    return GetTypedConfig[StructuredResponseConfig](cc.provider, cc.model, CapStructuredResponse)
}

// GetFunctionCallingConfig returns typed function calling config
func (cc *CapabilityChecker) GetFunctionCallingConfig() (FunctionCallingConfig, bool) {
    return GetTypedConfig[FunctionCallingConfig](cc.provider, cc.model, CapFunctionCalling)
}

// GetStreamingConfig returns typed streaming config
func (cc *CapabilityChecker) GetStreamingConfig() (StreamingConfig, bool) {
    return GetTypedConfig[StreamingConfig](cc.provider, cc.model, CapStreaming)
}

// GetVisionConfig returns typed vision config
func (cc *CapabilityChecker) GetVisionConfig() (VisionConfig, bool) {
    return GetTypedConfig[VisionConfig](cc.provider, cc.model, CapVision)
}

// GetCachingConfig returns typed caching config
func (cc *CapabilityChecker) GetCachingConfig() (CachingConfig, bool) {
    return GetTypedConfig[CachingConfig](cc.provider, cc.model, CapCaching)
}
```

### 3. Provider Implementations with Composition

```go
// provider_cohere.go
package providers

import (
    "encoding/json"
    "fmt"
)

type CohereProvider struct {
    // Composed capability checker
    capabilities *CapabilityChecker
    
    // Provider-specific fields
    apiKey   string
    model    string
    endpoint string
    headers  map[string]string
}

func NewCohereProvider(apiKey, model string, extraHeaders map[string]string) *CohereProvider {
    return &CohereProvider{
        capabilities: NewCapabilityChecker(ProviderCohere, model),
        apiKey:      apiKey,
        model:       model,
        endpoint:    "https://api.cohere.ai/v1",
        headers:     extraHeaders,
    }
}

// Core identification
func (p *CohereProvider) Name() ProviderName {
    return ProviderCohere
}

func (p *CohereProvider) Model() string {
    return p.model
}

// Delegate capability checking to the composed checker
func (p *CohereProvider) HasCapability(cap Capability) bool {
    return p.capabilities.HasCapability(cap)
}

func (p *CohereProvider) GetStructuredResponseConfig() (StructuredResponseConfig, bool) {
    return p.capabilities.GetStructuredResponseConfig()
}

func (p *CohereProvider) GetFunctionCallingConfig() (FunctionCallingConfig, bool) {
    return p.capabilities.GetFunctionCallingConfig()
}

func (p *CohereProvider) GetStreamingConfig() (StreamingConfig, bool) {
    return p.capabilities.GetStreamingConfig()
}

func (p *CohereProvider) GetVisionConfig() (VisionConfig, bool) {
    return p.capabilities.GetVisionConfig()
}

func (p *CohereProvider) GetCachingConfig() (CachingConfig, bool) {
    return p.capabilities.GetCachingConfig()
}

// Core provider methods
func (p *CohereProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
    // Handle structured response with Cohere's quirk
    if req.ResponseFormat != nil {
        if structConfig, ok := p.GetStructuredResponseConfig(); ok {
            if structConfig.RequiresToolUse {
                // Convert to tool calling format for Cohere
                return p.prepareToolCallRequest(req, options)
            }
        } else {
            return nil, fmt.Errorf("model %s does not support structured responses", p.model)
        }
    }
    
    // Handle function calling
    if len(req.Functions) > 0 {
        if funcConfig, ok := p.GetFunctionCallingConfig(); ok {
            if len(req.Functions) > funcConfig.MaxFunctions {
                return nil, fmt.Errorf("model %s supports max %d functions, got %d", 
                    p.model, funcConfig.MaxFunctions, len(req.Functions))
            }
            
            if funcConfig.RequiresToolRole {
                // Format functions with Cohere's specific role requirements
                return p.prepareFormattedToolRequest(req, options)
            }
        } else {
            return nil, fmt.Errorf("model %s does not support function calling", p.model)
        }
    }
    
    // Normal request
    return p.prepareStandardRequest(req, options)
}
```

```go
// provider_openai.go
package providers

import (
    "encoding/json"
    "fmt"
)

type OpenAIProvider struct {
    // Composed capability checker
    capabilities *CapabilityChecker
    
    // Provider-specific fields
    apiKey   string
    model    string
    endpoint string
    headers  map[string]string
}

func NewOpenAIProvider(apiKey, model string, extraHeaders map[string]string) *OpenAIProvider {
    return &OpenAIProvider{
        capabilities: NewCapabilityChecker(ProviderOpenAI, model),
        apiKey:      apiKey,
        model:       model,
        endpoint:    "https://api.openai.com/v1",
        headers:     extraHeaders,
    }
}

// Core identification
func (p *OpenAIProvider) Name() ProviderName {
    return ProviderOpenAI
}

func (p *OpenAIProvider) Model() string {
    return p.model
}

// Delegate capability checking to the composed checker
func (p *OpenAIProvider) HasCapability(cap Capability) bool {
    return p.capabilities.HasCapability(cap)
}

func (p *OpenAIProvider) GetStructuredResponseConfig() (StructuredResponseConfig, bool) {
    return p.capabilities.GetStructuredResponseConfig()
}

func (p *OpenAIProvider) GetFunctionCallingConfig() (FunctionCallingConfig, bool) {
    return p.capabilities.GetFunctionCallingConfig()
}

func (p *OpenAIProvider) GetStreamingConfig() (StreamingConfig, bool) {
    return p.capabilities.GetStreamingConfig()
}

func (p *OpenAIProvider) GetVisionConfig() (VisionConfig, bool) {
    return p.capabilities.GetVisionConfig()
}

func (p *OpenAIProvider) GetCachingConfig() (CachingConfig, bool) {
    return p.capabilities.GetCachingConfig()
}

func (p *OpenAIProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
    openAIReq := map[string]interface{}{
        "model":    p.model,
        "messages": req.Messages,
    }
    
    // Handle structured response
    if req.ResponseFormat != nil {
        if structConfig, ok := p.GetStructuredResponseConfig(); ok {
            if structConfig.RequiresJSONMode {
                openAIReq["response_format"] = map[string]string{
                    "type": "json_object",
                }
            }
            
            // Add the schema if supported
            if contains(structConfig.SupportedFormats, "json_schema") {
                openAIReq["response_format"] = map[string]interface{}{
                    "type":   "json_schema",
                    "schema": req.ResponseFormat.Schema,
                }
            }
        } else {
            return nil, fmt.Errorf("model %s does not support structured responses", p.model)
        }
    }
    
    // Handle vision
    if req.Images != nil && len(req.Images) > 0 {
        if visionConfig, ok := p.GetVisionConfig(); ok {
            if len(req.Images) > visionConfig.MaxImagesPerRequest {
                return nil, fmt.Errorf("too many images: %d (max: %d)", 
                    len(req.Images), visionConfig.MaxImagesPerRequest)
            }
            // Process images...
        } else {
            return nil, fmt.Errorf("model %s does not support vision", p.model)
        }
    }
    
    return json.Marshal(openAIReq)
}
```

### 4. Optional Helper Functions for Providers

```go
// provider_helpers.go
package providers

// For providers that want even simpler capability delegation
// They can compose this instead of CapabilityChecker
type CapabilityDelegate struct {
    *CapabilityChecker
}

func NewCapabilityDelegate(provider ProviderName, model string) CapabilityDelegate {
    return CapabilityDelegate{
        CapabilityChecker: NewCapabilityChecker(provider, model),
    }
}

// Now a provider can embed this and get all methods automatically
// But you said no embedding, so let's make it a helper function instead

// CreateProviderWithCapabilities helps create providers with capability checking
func CreateProviderWithCapabilities(provider ProviderName, model string) *CapabilityChecker {
    return NewCapabilityChecker(provider, model)
}
```

### 5. Usage Examples - Clean and Direct

```go
// example_usage.go
package main

import (
    "fmt"
    "log"
    "your-package/providers"
)

func main() {
    // Create a provider
    provider := providers.NewCohereProvider("api-key", "command-r-plus", nil)
    
    // Example 1: Direct capability check - no GetChecker() needed!
    if provider.HasCapability(providers.CapStructuredResponse) {
        fmt.Println("✓ Supports structured response")
    }
    
    // Example 2: Get configuration directly from provider
    if structConfig, ok := provider.GetStructuredResponseConfig(); ok {
        if structConfig.RequiresToolUse {
            fmt.Println("⚠️  This model requires tool use for structured responses")
        }
        fmt.Printf("Max schema depth: %d\n", structConfig.MaxSchemaDepth)
        fmt.Printf("Supported formats: %v\n", structConfig.SupportedFormats)
    }
    
    // Example 3: Check capabilities in a clean way
    printCapabilities(provider)
    
    // Example 4: Work with any provider through the interface
    validateProvider(provider)
}

func printCapabilities(p providers.Provider) {
    fmt.Printf("\nCapabilities for %s/%s:\n", p.Name(), p.Model())
    
    capabilities := []providers.Capability{
        providers.CapStructuredResponse,
        providers.CapFunctionCalling,
        providers.CapVision,
        providers.CapStreaming,
        providers.CapCaching,
    }
    
    for _, cap := range capabilities {
        if p.HasCapability(cap) {
            fmt.Printf("  ✓ %s", cap)
            
            // Show specific details for each capability
            switch cap {
            case providers.CapStructuredResponse:
                if cfg, ok := p.GetStructuredResponseConfig(); ok {
                    if cfg.RequiresToolUse {
                        fmt.Printf(" (via tools only)")
                    }
                }
            case providers.CapFunctionCalling:
                if cfg, ok := p.GetFunctionCallingConfig(); ok {
                    fmt.Printf(" (max: %d)", cfg.MaxFunctions)
                }
            case providers.CapVision:
                if cfg, ok := p.GetVisionConfig(); ok {
                    fmt.Printf(" (max images: %d)", cfg.MaxImagesPerRequest)
                }
            }
            fmt.Println()
        } else {
            fmt.Printf("  ✗ %s\n", cap)
        }
    }
}

func validateProvider(p providers.Provider) error {
    // Works with any provider implementation
    if !p.HasCapability(providers.CapStreaming) {
        return fmt.Errorf("provider %s with model %s does not support streaming", 
            p.Name(), p.Model())
    }
    
    if funcConfig, ok := p.GetFunctionCallingConfig(); ok {
        fmt.Printf("Provider supports up to %d functions\n", funcConfig.MaxFunctions)
        if funcConfig.SupportsParallel {
            fmt.Println("Provider can execute functions in parallel")
        }
    }
    
    return nil
}

// Example of building requests with capability validation
func buildRequestWithValidation(p providers.Provider, userMessage string) (*providers.Request, error) {
    req := &providers.Request{
        Messages: []providers.Message{
            {Role: "user", Content: userMessage},
        },
    }
    
    // Want structured response?
    if structConfig, ok := p.GetStructuredResponseConfig(); ok {
        req.ResponseFormat = &providers.ResponseFormat{
            Type: "json",
            Schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "answer": map[string]string{"type": "string"},
                },
            },
        }
        
        // Handle provider quirks
        if structConfig.RequiresToolUse {
            fmt.Println("Note: This provider will use tool calling for structured output")
        }
        
        if structConfig.SystemPromptHint != "" {
            // Prepend system message
            req.Messages = append([]providers.Message{
                {Role: "system", Content: structConfig.SystemPromptHint},
            }, req.Messages...)
        }
    } else {
        return nil, fmt.Errorf("structured response not supported by %s", p.Model())
    }
    
    return req, nil
}
```

### 6. Testing

```go
// provider_test.go
package providers

import (
    "testing"
)

func TestProviderCapabilities(t *testing.T) {
    tests := []struct {
        name     string
        provider Provider
        cap      Capability
        expected bool
    }{
        {
            name:     "Cohere supports structured response",
            provider: NewCohereProvider("key", "command-r-plus", nil),
            cap:      CapStructuredResponse,
            expected: true,
        },
        {
            name:     "OpenAI GPT-4o supports vision",
            provider: NewOpenAIProvider("key", "gpt-4o", nil),
            cap:      CapVision,
            expected: true,
        },
        {
            name:     "OpenAI o1 does not support function calling",
            provider: NewOpenAIProvider("key", "o1-preview", nil),
            cap:      CapFunctionCalling,
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.provider.HasCapability(tt.cap)
            if got != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, got)
            }
        })
    }
}

func TestCohereStructuredResponseQuirk(t *testing.T) {
    provider := NewCohereProvider("key", "command-r-plus", nil)
    
    // Should support structured response
    if !provider.HasCapability(CapStructuredResponse) {
        t.Fatal("expected Cohere to support structured response")
    }
    
    // Should require tool use
    cfg, ok := provider.GetStructuredResponseConfig()
    if !ok {
        t.Fatal("expected to get structured response config")
    }
    
    if !cfg.RequiresToolUse {
        t.Error("expected Cohere to require tool use for structured response")
    }
}

func TestProviderInterface(t *testing.T) {
    // Test that all providers implement the interface correctly
    var providers []Provider = []Provider{
        NewOpenAIProvider("key", "gpt-4", nil),
        NewCohereProvider("key", "command-r", nil),
        NewGeminiProvider("key", "gemini-pro", nil),
    }
    
    for _, p := range providers {
        // All providers should implement these methods
        _ = p.Name()
        _ = p.Model()
        _ = p.HasCapability(CapStreaming)
        _, _ = p.GetStreamingConfig()
        
        // This ensures interface compliance
    }
}
```

### 7. Mock Provider for Testing

```go
// provider_mock.go
package providers

// MockProvider for testing - shows how easy it is to implement
type MockProvider struct {
    capabilities *CapabilityChecker
    name         ProviderName
    model        string
}

func NewMockProvider(name ProviderName, model string) *MockProvider {
    return &MockProvider{
        capabilities: NewCapabilityChecker(name, model),
        name:        name,
        model:       model,
    }
}

func (m *MockProvider) Name() ProviderName { return m.name }
func (m *MockProvider) Model() string { return m.model }

// Delegate all capability methods
func (m *MockProvider) HasCapability(cap Capability) bool {
    return m.capabilities.HasCapability(cap)
}

func (m *MockProvider) GetStructuredResponseConfig() (StructuredResponseConfig, bool) {
    return m.capabilities.GetStructuredResponseConfig()
}

func (m *MockProvider) GetFunctionCallingConfig() (FunctionCallingConfig, bool) {
    return m.capabilities.GetFunctionCallingConfig()
}

func (m *MockProvider) GetStreamingConfig() (StreamingConfig, bool) {
    return m.capabilities.GetStreamingConfig()
}

func (m *MockProvider) GetVisionConfig() (VisionConfig, bool) {
    return m.capabilities.GetVisionConfig()
}

func (m *MockProvider) GetCachingConfig() (CachingConfig, bool) {
    return m.capabilities.GetCachingConfig()
}

// Mock implementations
func (m *MockProvider) PrepareRequest(req *Request, options map[string]any) ([]byte, error) {
    return []byte("mock request"), nil
}

func (m *MockProvider) PrepareStreamRequest(req *Request, options map[string]any) ([]byte, error) {
    return []byte("mock stream request"), nil
}

func (m *MockProvider) ParseResponse(body []byte) (*Response, error) {
    return &Response{}, nil
}

func (m *MockProvider) ParseStreamResponse(chunk []byte) (*Response, error) {
    return &Response{}, nil
}
```

## Key Improvements:

1. **Pure Composition**: No embedding, providers compose the `CapabilityChecker`
2. **Direct Access**: No need to call `GetChecker()`, capability methods are on the Provider interface
3. **Clean Interface**: Provider interface clearly shows all capability-related methods
4. **Type Safety**: All capability checks return typed configs
5. **Simple Implementation**: Each provider just delegates to its composed checker
6. **Testable**: Easy to mock and test

The user experience is now very clean:
- `provider.HasCapability(cap)` - direct check
- `provider.GetStructuredResponseConfig()` - direct typed config access
- No intermediate objects or extra method calls needed!