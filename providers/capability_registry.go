// Package providers implements various Language Learning Model (LLM) provider interfaces
package providers

import (
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v4"
)

// globalRegistry is the singleton instance of the capability registry
var (
	globalRegistry *CapabilityRegistry
	registryOnce   sync.Once
)

// CapabilityRegistry is a global registry for provider capabilities
type CapabilityRegistry struct {
	// Using xsync.MapOf for lock-free concurrent access
	// Key is provider:model string, value is map of capabilities
	capabilities *xsync.Map[string, map[Capability]CapabilityConfig]
}

// GetRegistry returns the singleton instance of the CapabilityRegistry
func GetRegistry() *CapabilityRegistry {
	registryOnce.Do(func() {
		globalRegistry = &CapabilityRegistry{
			capabilities: xsync.NewMap[string, map[Capability]CapabilityConfig](),
		}
	})
	return globalRegistry
}

// Register adds a capability for a specific provider and model
func (r *CapabilityRegistry) Register(
	provider ProviderName,
	model string,
	capability Capability,
	config CapabilityConfig,
) {
	if config == nil {
		return
	}

	key := r.makeKey(provider, model)

	// Load or create the capability map for this provider:model
	caps, _ := r.capabilities.LoadOrStore(key, make(map[Capability]CapabilityConfig))

	// Update the capability map
	// Note: This is a simple assignment, not thread-safe for the inner map
	// If multiple goroutines register capabilities for the same provider:model simultaneously,
	// we need to synchronize access to the inner map
	// For now, we assume registration happens during initialization (single-threaded)
	caps[capability] = config
}

// HasCapability checks if a provider/model combination has a specific capability
func (r *CapabilityRegistry) HasCapability(provider ProviderName, model string, capability Capability) bool {
	key := r.makeKey(provider, model)

	caps, exists := r.capabilities.Load(key)
	if !exists {
		return false
	}

	_, hasCapability := caps[capability]
	return hasCapability
}

// GetConfig retrieves the configuration for a specific capability
func (r *CapabilityRegistry) GetConfig(provider ProviderName, model string, cap Capability) CapabilityConfig {
	key := r.makeKey(provider, model)

	caps, exists := r.capabilities.Load(key)
	if !exists {
		return nil
	}

	return caps[cap]
}

// makeKey creates a unique key for provider/model combination
func (r *CapabilityRegistry) makeKey(provider ProviderName, model string) string {
	return string(provider) + ":" + model
}

// Clear removes all registered capabilities (useful for testing)
func (r *CapabilityRegistry) Clear() {
	r.capabilities.Clear()
}

// GetCapabilityConfig is a generic function to get typed capability config
// The type T itself determines which capability to fetch via its Name() method
func GetCapabilityConfig[T CapabilityConfig](provider ProviderName, model string) (T, error) {
	var zero T

	// Create a zero value to get the capability name
	capName := zero.Name()

	// Get the config from registry
	config := GetRegistry().GetConfig(provider, model, capName)
	if config == nil {
		return zero, fmt.Errorf("capability %s not found for provider %s model %s", capName, provider, model)
	}

	// Type assert to the requested type
	typedConfig, ok := config.(T)
	if !ok {
		return zero, fmt.Errorf("capability config type mismatch: expected %T, got %T", zero, config)
	}

	return typedConfig, nil
}
