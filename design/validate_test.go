package design

import (
	"encoding/base64"
	"testing"
)

func TestValidate_ValidCases(t *testing.T) {
	tests := []struct {
		name string
		h    HypothesisDesign
	}{
		{
			name: "valid af-dsl design",
			h: HypothesisDesign{
				Status: "ok",
				Algorithm: struct {
					Name       string `json:"name"`
					Idea       string `json:"idea"`
					Complexity struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					} `json:"complexity"`
				}{
					Name: "QuickSort",
					Idea: "Divide and conquer sorting",
					Complexity: struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					}{
						Time:  "O(n log n)",
						Space: "O(log n)",
					},
				},
				Code: struct {
					Lang  string `json:"lang"`
					Entry string `json:"entry"`
					Src   string `json:"src"`
				}{
					Lang:  "af-dsl",
					Entry: "program",
					Src:   "(program (let x 5) (return x))",
				},
				Evaluation: struct {
					Metrics       []string `json:"metrics"`
					Fitness       string   `json:"fitness"`
					PassThreshold float64  `json:"pass_threshold"`
				}{
					Metrics:       []string{"correctness", "time", "size"},
					Fitness:       "score = 0.8*correctness + 0.15*time + 0.05*size",
					PassThreshold: 0.95,
				},
				Tests: struct {
					Unit []struct {
						Name   string  `json:"name"`
						Input  string  `json:"input"`
						Oracle string  `json:"oracle"`
						Weight float64 `json:"weight"`
					} `json:"unit"`
					Property []struct {
						Name      string   `json:"name"`
						Generator string   `json:"generator"`
						Checks    []string `json:"checks"`
					} `json:"property"`
				}{
					Unit: []struct {
						Name   string  `json:"name"`
						Input  string  `json:"input"`
						Oracle string  `json:"oracle"`
						Weight float64 `json:"weight"`
					}{
						{
							Name:   "test1",
							Input:  "[3,1,4]",
							Oracle: "[1,3,4]",
							Weight: 1.0,
						},
					},
					Property: []struct {
						Name      string   `json:"name"`
						Generator string   `json:"generator"`
						Checks    []string `json:"checks"`
					}{
						{
							Name:      "prop1",
							Generator: "list<int>(n<=100)",
							Checks:    []string{"sorted?", "permutes?"},
						},
					},
				},
			},
		},
		{
			name: "valid wasm design",
			h: HypothesisDesign{
				Status: "ok",
				Algorithm: struct {
					Name       string `json:"name"`
					Idea       string `json:"idea"`
					Complexity struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					} `json:"complexity"`
				}{
					Name: "BubbleSort",
					Idea: "Simple comparison sort",
					Complexity: struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					}{
						Time:  "O(n²)",
						Space: "O(1)",
					},
				},
				Code: struct {
					Lang  string `json:"lang"`
					Entry string `json:"entry"`
					Src   string `json:"src"`
				}{
					Lang:  "wasm",
					Entry: "program",
					Src:   base64.StdEncoding.EncodeToString([]byte("dummy wasm binary")),
				},
				Evaluation: struct {
					Metrics       []string `json:"metrics"`
					Fitness       string   `json:"fitness"`
					PassThreshold float64  `json:"pass_threshold"`
				}{
					Metrics:       []string{"correctness", "time"},
					Fitness:       "score = 0.9*correctness + 0.1*time",
					PassThreshold: 0.8,
				},
				Tests: struct {
					Unit []struct {
						Name   string  `json:"name"`
						Input  string  `json:"input"`
						Oracle string  `json:"oracle"`
						Weight float64 `json:"weight"`
					} `json:"unit"`
					Property []struct {
						Name      string   `json:"name"`
						Generator string   `json:"generator"`
						Checks    []string `json:"checks"`
					} `json:"property"`
				}{
					Unit: []struct {
						Name   string  `json:"name"`
						Input  string  `json:"input"`
						Oracle string  `json:"oracle"`
						Weight float64 `json:"weight"`
					}{
						{
							Name:   "test1",
							Input:  "[5,2,8]",
							Oracle: "[2,5,8]",
							Weight: 0.5,
						},
						{
							Name:   "test2",
							Input:  "[1]",
							Oracle: "[1]",
							Weight: 0.5,
						},
					},
				},
			},
		},
		{
			name: "valid cannot_solve",
			h: HypothesisDesign{
				Status: "cannot_solve",
				Reason: "Task is underspecified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.h); err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestValidate_InvalidCases(t *testing.T) {
	tests := []struct {
		name string
		h    HypothesisDesign
		want string
	}{
		{
			name: "invalid status",
			h: HypothesisDesign{
				Status: "invalid_status",
			},
			want: "status must be 'ok' or 'cannot_solve'",
		},
		{
			name: "cannot_solve without reason",
			h: HypothesisDesign{
				Status: "cannot_solve",
			},
			want: "reason is required when status is 'cannot_solve'",
		},
		{
			name: "empty algorithm name",
			h: HypothesisDesign{
				Status: "ok",
				Algorithm: struct {
					Name       string `json:"name"`
					Idea       string `json:"idea"`
					Complexity struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					} `json:"complexity"`
				}{
					Name: "",
					Idea: "test",
					Complexity: struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					}{
						Time:  "O(n)",
						Space: "O(1)",
					},
				},
			},
			want: "algorithm name cannot be empty",
		},
		{
			name: "empty algorithm idea",
			h: HypothesisDesign{
				Status: "ok",
				Algorithm: struct {
					Name       string `json:"name"`
					Idea       string `json:"idea"`
					Complexity struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					} `json:"complexity"`
				}{
					Name: "test",
					Idea: "",
					Complexity: struct {
						Time  string `json:"time"`
						Space string `json:"space"`
					}{
						Time:  "O(n)",
						Space: "O(1)",
					},
				},
			},
			want: "algorithm idea cannot be empty",
		},
		{
			name: "invalid code lang",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Lang = "python"
				return h
			}(),
			want: "code lang must be 'af-dsl' or 'wasm'",
		},
		{
			name: "invalid code entry",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Entry = "main"
				return h
			}(),
			want: "code entry must be 'program'",
		},
		{
			name: "empty code src",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Src = ""
				return h
			}(),
			want: "code src cannot be empty",
		},
		{
			name: "invalid wasm base64",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Lang = "wasm"
				h.Code.Src = "invalid base64!"
				return h
			}(),
			want: "wasm code must be valid base64 encoded",
		},
		{
			name: "wasm with forbidden path",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Lang = "wasm"
				h.Code.Src = "file:///etc/passwd" // Not base64 encoded to test pattern detection
				return h
			}(),
			want: "wasm code must be valid base64 encoded",
		},
		{
			name: "wasm with forbidden pattern in base64",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Code.Lang = "wasm"
				// Base64 encode a string that contains forbidden patterns
				h.Code.Src = base64.StdEncoding.EncodeToString([]byte("http://localhost:8080"))
				return h
			}(),
			want: "wasm code contains forbidden pattern: http://",
		},
		{
			name: "invalid pass threshold too high",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Evaluation.PassThreshold = 1.5
				return h
			}(),
			want: "pass_threshold must be between 0 and 1",
		},
		{
			name: "invalid pass threshold too low",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Evaluation.PassThreshold = -0.1
				return h
			}(),
			want: "pass_threshold must be between 0 and 1",
		},
		{
			name: "missing correctness metric",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Evaluation.Metrics = []string{"time", "size"}
				return h
			}(),
			want: "metrics must include 'correctness'",
		},
		{
			name: "empty fitness formula",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Evaluation.Fitness = ""
				return h
			}(),
			want: "fitness formula cannot be empty",
		},
		{
			name: "zero total unit test weight",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Unit = []struct {
					Name   string  `json:"name"`
					Input  string  `json:"input"`
					Oracle string  `json:"oracle"`
					Weight float64 `json:"weight"`
				}{
					{
						Name:   "test1",
						Input:  "[1,2,3]",
						Oracle: "[1,2,3]",
						Weight: 0.0,
					},
				}
				return h
			}(),
			want: "unit tests must have total weight > 0",
		},
		{
			name: "negative unit test weight",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Unit = []struct {
					Name   string  `json:"name"`
					Input  string  `json:"input"`
					Oracle string  `json:"oracle"`
					Weight float64 `json:"weight"`
				}{
					{
						Name:   "test1",
						Input:  "[1,2,3]",
						Oracle: "[1,2,3]",
						Weight: -1.0,
					},
				}
				return h
			}(),
			want: "unit test weight cannot be negative",
		},
		{
			name: "empty unit test name",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Unit = []struct {
					Name   string  `json:"name"`
					Input  string  `json:"input"`
					Oracle string  `json:"oracle"`
					Weight float64 `json:"weight"`
				}{
					{
						Name:   "",
						Input:  "[1,2,3]",
						Oracle: "[1,2,3]",
						Weight: 1.0,
					},
				}
				return h
			}(),
			want: "unit test name cannot be empty",
		},
		{
			name: "empty property test name",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Property = []struct {
					Name      string   `json:"name"`
					Generator string   `json:"generator"`
					Checks    []string `json:"checks"`
				}{
					{
						Name:      "",
						Generator: "list<int>(n<=100)",
						Checks:    []string{"sorted?"},
					},
				}
				return h
			}(),
			want: "property test name cannot be empty",
		},
		{
			name: "empty property test generator",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Property = []struct {
					Name      string   `json:"name"`
					Generator string   `json:"generator"`
					Checks    []string `json:"checks"`
				}{
					{
						Name:      "prop1",
						Generator: "",
						Checks:    []string{"sorted?"},
					},
				}
				return h
			}(),
			want: "property test generator cannot be empty",
		},
		{
			name: "empty property test checks",
			h: func() HypothesisDesign {
				h := createValidDesign()
				h.Tests.Property = []struct {
					Name      string   `json:"name"`
					Generator string   `json:"generator"`
					Checks    []string `json:"checks"`
				}{
					{
						Name:      "prop1",
						Generator: "list<int>(n<=100)",
						Checks:    []string{},
					},
				}
				return h
			}(),
			want: "property test must have at least one check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.h)
			if err == nil {
				t.Errorf("Validate() error = nil, want error containing %q", tt.want)
				return
			}
			if !contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

// Helper function to create a valid design for testing
func createValidDesign() HypothesisDesign {
	return HypothesisDesign{
		Status: "ok",
		Algorithm: struct {
			Name       string `json:"name"`
			Idea       string `json:"idea"`
			Complexity struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			} `json:"complexity"`
		}{
			Name: "TestAlgorithm",
			Idea: "Test idea",
			Complexity: struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			}{
				Time:  "O(n)",
				Space: "O(1)",
			},
		},
		Code: struct {
			Lang  string `json:"lang"`
			Entry string `json:"entry"`
			Src   string `json:"src"`
		}{
			Lang:  "af-dsl",
			Entry: "program",
			Src:   "(program (return 42))",
		},
		Evaluation: struct {
			Metrics       []string `json:"metrics"`
			Fitness       string   `json:"fitness"`
			PassThreshold float64  `json:"pass_threshold"`
		}{
			Metrics:       []string{"correctness", "time"},
			Fitness:       "score = 0.8*correctness + 0.2*time",
			PassThreshold: 0.9,
		},
		Tests: struct {
			Unit []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			} `json:"unit"`
			Property []struct {
				Name      string   `json:"name"`
				Generator string   `json:"generator"`
				Checks    []string `json:"checks"`
			} `json:"property"`
		}{
			Unit: []struct {
				Name   string  `json:"name"`
				Input  string  `json:"input"`
				Oracle string  `json:"oracle"`
				Weight float64 `json:"weight"`
			}{
				{
					Name:   "test1",
					Input:  "[1,2,3]",
					Oracle: "[1,2,3]",
					Weight: 1.0,
				},
			},
			Property: []struct {
				Name      string   `json:"name"`
				Generator string   `json:"generator"`
				Checks    []string `json:"checks"`
			}{
				{
					Name:      "prop1",
					Generator: "list<int>(n<=100)",
					Checks:    []string{"sorted?"},
				},
			},
		},
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr))))
}
