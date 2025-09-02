package providers

import (
	"errors"
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/weave-labs/weave-go/weaveapi/llmx/v1"
)

var (
	registry     *CapabilityRegistry
	registryOnce sync.Once
)

// CapabilityRegistry manages capabilities for all providers and models.
type CapabilityRegistry struct {
	models *xsync.Map[string, ModelCapabilities] // key: "provider:model"
}

// GetCapabilityRegistry returns the singleton global capability registry.
func GetCapabilityRegistry() *CapabilityRegistry {
	registryOnce.Do(func() {
		registry = &CapabilityRegistry{
			models: xsync.NewMap[string, ModelCapabilities](),
		}
	})

	return registry
}

// RegisterCapability registers a capability for a specific provider and model.
func (r *CapabilityRegistry) RegisterCapability(
	provider string,
	model string,
	capType llmx.CapabilityType,
	config any,
) {
	modelCaps, _ := r.models.LoadOrStore(makeSlug(provider, model), NewModelCapabilities())
	modelCaps.AddCapability(capType, config)
}

// HasCapability checks if a capability is registered for a specific provider and model.
func (r *CapabilityRegistry) HasCapability(provider string, model string, capType llmx.CapabilityType) bool {
	modelCaps, exists := r.models.Load(makeSlug(provider, model))
	if !exists {
		return false
	}

	return modelCaps.HasCapability(capType)
}

func (r *CapabilityRegistry) GetConfig(provider string, model string, capType llmx.CapabilityType) any {
	modelCaps, exists := r.models.Load(makeSlug(provider, model))
	if !exists {
		return nil
	}

	return modelCaps.GetCapability(capType)
}

// Clear removes all registered capabilities (mainly for testing).
func (r *CapabilityRegistry) Clear() {
	r.models.Clear()
}

// makeSlug creates a unique key for provider and model combination.
func makeSlug(provider string, model string) string {
	return provider + "/" + model
}

// GetCapability is a generic function to get typed capability config
// The type T itself determines which capability to fetch via its Name() method
func GetCapability[T any](provider string, model string) (T, error) {
	zeroVal := *new(T)

	getTyper, ok := any(zeroVal).(interface{ GetType() llmx.CapabilityType })
	if !ok {
		return zeroVal, errors.New("capability type not supported")
	}

	capName := getTyper.GetType()
	config := GetCapabilityRegistry().GetConfig(provider, model, capName)
	if config == nil {
		return zeroVal, fmt.Errorf("capability %s not found for provider %s model %s", capName, provider, model)
	}

	// Type assert to the requested type
	typedConfig, ok := config.(T)
	if !ok {
		return zeroVal, fmt.Errorf("capability config type mismatch: expected %T, got %T", zeroVal, config)
	}

	return typedConfig, nil
}

type ModelCapabilities []any

// NewModelCapabilities creates a new capability registry.
func NewModelCapabilities() ModelCapabilities {
	var maxCapabilityVal int
	for _, value := range llmx.CapabilityType_value {
		if int(value) > maxCapabilityVal {
			maxCapabilityVal = int(value)
		}
	}

	return make([]any, maxCapabilityVal+1)
}

// AddCapability registers a capability configuration for a given type.
// The config should be the actual proto type (e.g., *llmx.StructuredResponseCapability).
func (r ModelCapabilities) AddCapability(capType llmx.CapabilityType, config any) {
	if int(capType) < len(r) {
		r[capType] = config
	}
}

// GetCapability retrieves a capability configuration for a given type.
// Returns nil if not found.
func (r ModelCapabilities) GetCapability(capType llmx.CapabilityType) any {
	if int(capType) < len(r) {
		return r[capType]
	}

	return nil
}

// HasCapability checks if a capability is registered.
func (r ModelCapabilities) HasCapability(capType llmx.CapabilityType) bool {
	if int(capType) < len(r) {
		return r[capType] != nil
	}

	return false
}
