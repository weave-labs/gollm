package llm

import (
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// GenerateOption is a function type for configuring generation behavior.
type GenerateOption func(*GenerateConfig)

// WithStructuredResponse configures Generate to produce output conforming to the provided schema type.
// The generic type parameter T should be a struct type describing the expected JSON structure.
func WithStructuredResponse[T any]() GenerateOption {
	return func(cfg *GenerateConfig) {
		schema, err := jsonschema.For[T](&jsonschema.ForOptions{
			IgnoreInvalidTypes: true,
			TypeSchemas:        nil,
		})
		if err != nil {
			panic(err)
		}

		var zero T
		rt := reflect.TypeOf(zero)
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}

		cfg.structuredResponseType = reflect.New(rt).Interface()
		cfg.StructuredResponseSchema = schema
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
	RetryStrategy            RetryStrategy
	StructuredResponseSchema *jsonschema.Schema
	StreamBufferSize         int
	structuredResponseType   any
}
