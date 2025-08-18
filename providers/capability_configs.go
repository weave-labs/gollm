// Package providers implements various Language Learning Model (LLM) provider interfaces
package providers

// CapabilityConfig is a sealed interface - only types in this package can implement it
type CapabilityConfig interface {
	// Private method makes this interface sealed to this package
	isCapabilityConfig()
	// Name returns the capability this config is for
	Name() Capability
}

// StructuredResponseConfig defines how structured responses work
type StructuredResponseConfig struct {
	SystemPromptHint string
	SupportedFormats []string
	MaxSchemaDepth   int
	RequiresToolUse  bool
	RequiresJSONMode bool
}

// Implement sealed interface
func (StructuredResponseConfig) isCapabilityConfig() {}
func (StructuredResponseConfig) Name() Capability    { return CapStructuredResponse }

// FunctionCallingConfig defines function calling capabilities
type FunctionCallingConfig struct {
	MaxFunctions      int
	MaxParallelCalls  int
	SupportsParallel  bool
	RequiresToolRole  bool
	SupportsStreaming bool
}

// Implement sealed interface
func (FunctionCallingConfig) isCapabilityConfig() {}
func (FunctionCallingConfig) Name() Capability    { return CapFunctionCalling }

// StreamingConfig defines streaming behavior
type StreamingConfig struct {
	ChunkDelimiter string
	BufferSize     int
	SupportsSSE    bool
	SupportsUsage  bool
}

// Implement sealed interface
func (StreamingConfig) isCapabilityConfig() {}
func (StreamingConfig) Name() Capability    { return CapStreaming }

// VisionConfig defines image handling capabilities
type VisionConfig struct {
	SupportedFormats    []string
	MaxImageSize        int64
	MaxImagesPerRequest int
	SupportsImageGen    bool
	SupportsVideoFrames bool
}

// Implement sealed interface
func (VisionConfig) isCapabilityConfig() {}
func (VisionConfig) Name() Capability    { return CapVision }

// CachingConfig defines caching capabilities
type CachingConfig struct {
	CacheKeyStrategy string
	MaxCacheSize     int64
	CacheTTLSeconds  int
}

// Implement sealed interface
func (CachingConfig) isCapabilityConfig() {}
func (CachingConfig) Name() Capability    { return CapCaching }

// SystemPromptConfig defines system prompt capabilities
type SystemPromptConfig struct {
	MaxLength        int
	SupportsMultiple bool // Whether multiple system prompts are supported
}

// Implement sealed interface
func (SystemPromptConfig) isCapabilityConfig() {}
func (SystemPromptConfig) Name() Capability    { return CapSystemPrompt }
