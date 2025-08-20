package providers

import (
	"testing"
)

func TestCapabilities(t *testing.T) {
	// Clear registry before tests to ensure clean state
	GetRegistry().Clear()

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
		{
			name:     "OpenAI o1 does not support structured response",
			provider: NewOpenAIProvider("key", "o1-preview", nil),
			cap:      CapStructuredResponse,
			expected: false,
		},
		{
			name:     "OpenAI GPT-3.5 supports function calling",
			provider: NewOpenAIProvider("key", "gpt-3.5-turbo", nil),
			cap:      CapFunctionCalling,
			expected: true,
		},
		{
			name:     "Cohere command-r supports streaming",
			provider: NewCohereProvider("key", "command-r", nil),
			cap:      CapStreaming,
			expected: true,
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
	// Clear registry before test
	GetRegistry().Clear()

	provider := NewCohereProvider("key", "command-r-plus", nil)

	// Should support structured response
	if !provider.HasCapability(CapStructuredResponse) {
		t.Fatal("expected Cohere to support structured response")
	}

	// Check the config in the registry
	cfg, err := GetCapabilityConfig[StructuredResponseConfig](ProviderCohere, "command-r-plus")
	if err != nil {
		t.Fatal("expected to get config from registry")
	}

	if !cfg.RequiresToolUse {
		t.Error("expected Cohere to require tool use for structured response")
	}

	if cfg.SystemPromptHint == "" {
		t.Error("expected Cohere to have a system prompt hint")
	}
}

func TestOpenAIVisionConfig(t *testing.T) {
	// Clear registry before test
	GetRegistry().Clear()

	provider := NewOpenAIProvider("key", "gpt-4o", nil)

	// Should support vision
	if !provider.HasCapability(CapVision) {
		t.Fatal("expected GPT-4o to support vision")
	}

	// Get vision config from registry
	visionConfig, err := GetCapabilityConfig[VisionConfig](ProviderOpenAI, "gpt-4o")
	if err != nil {
		t.Fatalf("expected to get vision config from registry: %v", err)
	}

	// Check specific settings
	if visionConfig.MaxImageSize != 20*1024*1024 {
		t.Errorf("expected max image size of 20MB, got %d", visionConfig.MaxImageSize)
	}

	if visionConfig.MaxImagesPerRequest != 10 {
		t.Errorf("expected max 10 images per request, got %d", visionConfig.MaxImagesPerRequest)
	}

	if !visionConfig.SupportsVideoFrames {
		t.Error("expected GPT-4o to support video frames")
	}
}

func TestOpenAIModelSpecificCapabilities(t *testing.T) {
	// Clear registry before tests
	GetRegistry().Clear()

	tests := []struct {
		model       string
		capability  Capability
		shouldHave  bool
		description string
	}{
		{"gpt-4o", CapVision, true, "GPT-4o should support vision"},
		{"gpt-4-turbo", CapVision, true, "GPT-4-turbo should support vision"},
		{"gpt-3.5-turbo", CapVision, false, "GPT-3.5 should not support vision"},
		{"o1-preview", CapFunctionCalling, false, "O1 models should not support function calling"},
		{"o1-preview", CapStructuredResponse, false, "O1 models should not support structured response"},
		{"gpt-3.5-turbo-0125", CapStructuredResponse, true, "GPT-3.5-turbo-0125 should support structured response"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			provider := NewOpenAIProvider("key", tt.model, nil)
			got := provider.HasCapability(tt.capability)
			if got != tt.shouldHave {
				t.Errorf("model %s: expected %v for %s, got %v",
					tt.model, tt.shouldHave, tt.capability, got)
			}
		})
	}
}

func TestFunctionCallingConfig(t *testing.T) {
	// Clear registry before test
	GetRegistry().Clear()

	// Test OpenAI GPT-4 function calling config
	gpt4 := NewOpenAIProvider("key", "gpt-4", nil)
	if !gpt4.HasCapability(CapFunctionCalling) {
		t.Fatal("expected GPT-4 to support function calling")
	}

	funcConfig, err := GetCapabilityConfig[FunctionCallingConfig](ProviderOpenAI, "gpt-4")
	if err != nil {
		t.Fatalf("expected GPT-4 to have function calling config in registry: %v", err)
	}

	if funcConfig.MaxFunctions != 128 {
		t.Errorf("expected GPT-4 to support 128 functions, got %d", funcConfig.MaxFunctions)
	}
	if !funcConfig.SupportsParallel {
		t.Error("expected GPT-4 to support parallel function calls")
	}

	// Test OpenAI GPT-3.5 function calling config
	gpt35 := NewOpenAIProvider("key", "gpt-3.5-turbo", nil)
	if !gpt35.HasCapability(CapFunctionCalling) {
		t.Fatal("expected GPT-3.5 to support function calling")
	}

	funcConfig35, err := GetCapabilityConfig[FunctionCallingConfig](ProviderOpenAI, "gpt-3.5-turbo")
	if err != nil {
		t.Fatalf("expected GPT-3.5 to have function calling config in registry: %v", err)
	}

	if funcConfig35.MaxFunctions != 64 {
		t.Errorf("expected GPT-3.5 to support 64 functions, got %d", funcConfig35.MaxFunctions)
	}

	// Test Cohere function calling config
	cohere := NewCohereProvider("key", "command-r", nil)
	if !cohere.HasCapability(CapFunctionCalling) {
		t.Fatal("expected Cohere to support function calling")
	}

	cohereFuncConfig, err := GetCapabilityConfig[FunctionCallingConfig](ProviderCohere, "command-r")
	if err != nil {
		t.Fatalf("expected Cohere to have function calling config in registry: %v", err)
	}

	if cohereFuncConfig.SupportsParallel {
		t.Error("expected Cohere to not support parallel function calls")
	}
}

func TestGlobalRegistrySingleton(t *testing.T) {
	// Clear registry
	GetRegistry().Clear()

	// Get registry multiple times
	registry1 := GetRegistry()
	registry2 := GetRegistry()

	// They should be the same instance
	if registry1 != registry2 {
		t.Error("expected GetRegistry to return the same instance")
	}

	// Register a capability
	registry1.Register(ProviderOpenAI, "test-model", CapStreaming, StreamingConfig{
		SupportsSSE: true,
		BufferSize:  1024,
	})

	// Should be accessible from registry2
	if !registry2.HasCapability(ProviderOpenAI, "test-model", CapStreaming) {
		t.Error("expected capability registered in registry1 to be accessible from registry2")
	}
}
