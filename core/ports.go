package core

import "context"

type Skill interface {
	Name() string
	Domain() string
	CanSolve(task Task) (bool, float64) // match + confidence
	Execute(ctx context.Context, task Task) (Result, error)
	Tests() []TestCase
}

type KnowledgeBase interface {
	Find(task Task) []Skill // sorted by confidence
	SaveHypothesis(ctx context.Context, h Hypothesis, quality float64) error
}

type LLMClient interface {
	Propose(ctx context.Context, task Task) (algo string, tests []TestCase, criteria []string, err error)
}

type Interpreter interface {
	Execute(ctx context.Context, h Hypothesis, task Task) (Result, error)
}

type PolicyGuard interface {
	Wrap(ctx context.Context, b Budget, run func(ctx context.Context) error) error
	AllowTool(name string) bool
}

type TestCase struct {
	Name   string
	Input  []byte
	Oracle []byte   // expected answer if any
	Checks []string // properties/metamorphic checks
	Weight float64
}

type TestRunner interface {
	Run(ctx context.Context, h Hypothesis, cases []TestCase, exec Interpreter) (metrics map[string]float64, pass bool, err error)
}

type FitnessEvaluator interface {
	Score(task Task, metrics map[string]float64, sizeBytes int) float64
	Passed(score float64, threshold float64) bool
}

type Critic interface {
	Accept(task Task, metrics map[string]float64) (bool, string)
}

type Mutator interface {
	Mutate(base Hypothesis) []Hypothesis // набор кандидатов
}
