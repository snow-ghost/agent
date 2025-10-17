package design

import (
	"strings"
	"testing"
)

// Helper function to create a valid design for testing
func createValidDesignForValidation() HypothesisDesign {
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
			Name: "test-algorithm",
			Idea: "Test algorithm",
			Complexity: struct {
				Time  string `json:"time"`
				Space string `json:"space"`
			}{
				Time:  "O(n)",
				Space: "O(1)",
			},
		},
		Code: struct {
			Lang  string        `json:"lang"`
			Entry string        `json:"entry"`
			Src   string        `json:"src"`
			AST   *AFDSLProgram `json:"ast,omitempty"`
		}{
			Lang:  "af-dsl",
			Entry: "program",
			Src:   "(let x input (return x))",
		},
		Evaluation: struct {
			Metrics       []string `json:"metrics"`
			Fitness       string   `json:"fitness"`
			PassThreshold float64  `json:"pass_threshold"`
		}{
			Metrics:       []string{"correctness"},
			Fitness:       "correctness*1.0",
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
					Name:   "test_1",
					Input:  `{"test": "input"}`,
					Oracle: `{"test": "output"}`,
					Weight: 1.0,
				},
			},
		},
	}
}

func TestValidateAFDSLSecurity_ValidCode(t *testing.T) {
	hd := createValidDesignForValidation()

	err := Validate(hd)
	if err != nil {
		t.Errorf("Expected no error for valid AF-DSL code, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_TooLongIdentifier(t *testing.T) {
	longIdentifier := make([]byte, MaxIdentifierLength+1)
	for i := range longIdentifier {
		longIdentifier[i] = 'a'
	}

	hd := createValidDesignForValidation()
	hd.Code.Src = "(let " + string(longIdentifier) + " input (return " + string(longIdentifier) + "))"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for too long identifier, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "identifier too long: 101 characters (max 100)") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_TooLongString(t *testing.T) {
	longString := make([]byte, MaxStringLength+1)
	for i := range longString {
		longString[i] = 'a'
	}

	hd := createValidDesignForValidation()
	hd.Code.Src = `(let x "` + string(longString) + `" (return x))`

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for too long string, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "string literal too long: 10003 characters (max 10000)") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_InvalidIdentifier(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(let 123invalid input (return 123invalid))"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for invalid identifier, got nil")
	}
}

func TestValidateAFDSLSecurity_UnknownFunctionCall(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(call (evil_function input) (return result))"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for unknown function call, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "unknown function call: evil_function (not in allowed list)") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_AllowedFunctionCall(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(call (len input) (return result))"

	err := Validate(hd)
	if err != nil {
		t.Errorf("Expected no error for allowed function call, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_UnbalancedParentheses(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(let x input (return x)"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for unbalanced parentheses, got nil")
	}
}

func TestValidateAFDSLSecurity_InvalidUTF8(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = string([]byte{0xff, 0xfe, 0xfd}) + "(let x input (return x))"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for invalid UTF-8, got nil")
	}
}

func TestValidateAFDSLSecurity_ControlCharacters(t *testing.T) {
	hd := createValidDesignForValidation()
	// Use actual null byte instead of escape sequence
	hd.Code.Src = `(let x "string with ` + string([]byte{0x00}) + ` null byte" (return x))`

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for control characters, got nil")
	}
}

func TestValidateAFDSLSecurity_ComplexValidCode(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = `(let numbers input (if (call (len numbers) 0) (return "empty") (call (sorted? numbers) (return "sorted") (return "unsorted"))))`

	err := Validate(hd)
	if err != nil {
		t.Errorf("Expected no error for complex valid code, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_TooLongNumber(t *testing.T) {
	longNumber := "1" + string(make([]byte, MaxLiteralLength))

	hd := createValidDesignForValidation()
	hd.Code.Src = "(let x " + longNumber + " (return x))"

	err := Validate(hd)
	if err == nil {
		t.Error("Expected error for too long number, got nil")
	}
}

func TestValidateAFDSLSecurity_ComparisonOperators(t *testing.T) {
	// Test that comparison operators are now allowed
	testCases := []string{
		"(let x input (if (call (<= x 5) (return \"small\") (return \"large\"))))",
		"(let x input (if (call (>= x 10) (return \"big\") (return \"small\"))))",
		"(let x input (if (call (< x 0) (return \"negative\") (return \"positive\"))))",
		"(let x input (if (call (> x 100) (return \"huge\") (return \"normal\"))))",
		"(let x input (if (call (= x 42) (return \"answer\") (return \"not-answer\"))))",
		"(let x input (if (call (!= x 0) (return \"non-zero\") (return \"zero\"))))",
		"(let x input (if (call (== x 1) (return \"one\") (return \"not-one\"))))",
	}

	for i, code := range testCases {
		hd := createValidDesignForValidation()
		hd.Code.Src = code

		err := Validate(hd)
		if err != nil {
			t.Errorf("Test case %d: Expected no error for comparison operator code, got: %v", i+1, err)
		}
	}
}
func TestValidateAFDSLSecurity_SimpleComparison(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(<= x 5)"

	err := Validate(hd)
	if err != nil {
		t.Errorf("Expected no error for comparison operator code, got: %v", err)
	}
}

func TestValidateAFDSLSecurity_ComplexComparison(t *testing.T) {
	hd := createValidDesignForValidation()
	hd.Code.Src = "(let x input (if (call (<= x 5) (return \"small\") (return \"large\"))))"

	err := Validate(hd)
	if err != nil {
		t.Errorf("Expected no error for complex comparison operator code, got: %v", err)
	}
}
