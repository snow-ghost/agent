package worker

import "github.com/snow-ghost/agent/worker/capabilities"

// Re-export types from capabilities package for convenience
type Capabilities = capabilities.Capabilities
type WorkerWithCapabilities = capabilities.WorkerWithCapabilities

// DefaultCapabilities returns default capabilities for different worker types
func DefaultCapabilities(workerType WorkerType) Capabilities {
	switch workerType {
	case WorkerTypeLight:
		return capabilities.DefaultCapabilities("light")
	case WorkerTypeHeavy:
		return capabilities.DefaultCapabilities("heavy")
	default:
		return capabilities.DefaultCapabilities("light")
	}
}
