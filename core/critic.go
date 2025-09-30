package core

// SimpleCritic accepts if required checks are satisfied per metrics or logs.
// For MVP: if metrics contain pass=true or checks list is empty, accept.
type SimpleCritic struct{}

func NewSimpleCritic() *SimpleCritic { return &SimpleCritic{} }

// Accept returns ok=true if task's success criteria are met.
// MVP rule: if there are SuccessCriteria, require metrics["cases_failed"]==0 when present.
// If no criteria, accept.
func (c *SimpleCritic) Accept(task Task, metrics map[string]float64) (bool, string) {
	if len(task.Spec.SuccessCriteria) == 0 {
		return true, "no criteria"
	}
	if failed, ok := metrics["cases_failed"]; ok {
		if failed == 0 {
			return true, "all tests passed"
		}
		return false, "some tests failed"
	}
	// If no metric provided, be conservative.
	return false, "no test metrics"
}
