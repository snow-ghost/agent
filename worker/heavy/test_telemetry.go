package heavy

import (
	"sync"

	"github.com/snow-ghost/agent/worker/telemetry"
)

// Global telemetry instance for all tests to avoid duplicate metrics registration
var (
	globalTelemetry *telemetry.Telemetry
	globalOnce      sync.Once
)

// GetGlobalTelemetry returns a shared telemetry instance for all tests
func GetGlobalTelemetry() *telemetry.Telemetry {
	globalOnce.Do(func() {
		globalTelemetry = telemetry.NewTelemetry()
	})
	return globalTelemetry
}

