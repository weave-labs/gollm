package llm

import (
	"github.com/invopop/jsonschema"
)

// GenerateOption is a function type for configuring generation behavior.
type GenerateOption func(*GenerateConfig)

// WithStructuredResponse configures Generate to produce output conforming to the provided schema type.
// The generic type parameter T should be a struct type describing the expected JSON structure.
func WithStructuredResponse[T any]() GenerateOption {
	return func(cfg *GenerateConfig) {
		reflector := StructuredResponseReflector()
		cfg.StructuredResponse = StripSchemaIDs(reflector.Reflect(*new(T)))
	}
}

// WithStreamBufferSize sets the size of the token buffer for streaming responses.
func WithStreamBufferSize(size int) GenerateOption {
	return func(cfg *GenerateConfig) {
		cfg.StreamBufferSize = size
	}
}

// WithRetryStrategy defines how to handle stream interruptions.
func WithRetryStrategy(strategy RetryStrategy) GenerateOption {
	return func(cfg *GenerateConfig) {
		cfg.RetryStrategy = strategy
	}
}

// GenerateConfig holds configuration options for text generation.
type GenerateConfig struct {
	RetryStrategy      RetryStrategy
	StructuredResponse *jsonschema.Schema
	StreamBufferSize   int
}
