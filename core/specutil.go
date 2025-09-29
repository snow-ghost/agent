package core

// ValidateCriteria checks whether a task's spec meets criteria. Stub returns true.
func ValidateCriteria(task Task) (bool, []string) {
	return true, nil
}

// ValidateMetrics checks metrics against thresholds. Stub returns true.
func ValidateMetrics(result Result) (bool, []string) {
	return true, nil
}
