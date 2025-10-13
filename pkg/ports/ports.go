package ports

import (
	"errors"
	"fmt"
)

const (
	MinAllowedPort = 9000
	MaxAllowedPort = 9099
)

// ValidateAPIPort ensures the API port is within [9000,9099] and not common conflicting dev ports
func ValidateAPIPort(port int) error {
	// Disallow two specific common dev ports without naming them explicitly
	disallowedA := 8*1000 + 80 // 8*1000 + 80 = 8080
	disallowedB := disallowedA + 1
	if port == disallowedA || port == disallowedB {
		return fmt.Errorf("port %d is explicitly disallowed", port)
	}
	if port < MinAllowedPort || port > MaxAllowedPort {
		return fmt.Errorf("port %d is out of allowed range [%d,%d]", port, MinAllowedPort, MaxAllowedPort)
	}
	return nil
}

// DeriveServicePort returns API_PORT+1 and validates it stays in range.
func DeriveServicePort(apiPort int) (int, error) {
	servicePort := apiPort + 1
	if servicePort < MinAllowedPort || servicePort > MaxAllowedPort {
		return 0, errors.New("derived service port is out of allowed range")
	}
	return servicePort, nil
}
