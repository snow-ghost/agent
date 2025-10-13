package prompts

import (
	"encoding/json"
	"testing"
)

func TestBuildUserTaskJSON(t *testing.T) {
	// Тестовые данные
	id := "test-task-001"
	domain := "algorithms"
	desc := "Sort an array of integers"
	inSchema := `{"type": "array", "items": {"type": "integer"}}`
	outSchema := `{"type": "array", "items": {"type": "integer"}}`
	examples := []map[string]string{
		{"input": "[3, 1, 4, 1, 5]", "output": "[1, 1, 3, 4, 5]"},
		{"input": "[5, 2, 8, 1]", "output": "[1, 2, 5, 8]"},
	}
	opts := BuildOpts{
		TimeoutMS:     5000,
		MemMB:         128,
		MaxComplexity: "O(n log n)",
	}

	// Вызываем функцию
	result := BuildUserTaskJSON(id, domain, desc, inSchema, outSchema, examples, opts)

	// Проверяем, что результат не пустой
	if result == "" {
		t.Fatal("BuildUserTaskJSON returned empty string")
	}

	// Проверяем, что результат является валидным JSON
	var taskData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &taskData); err != nil {
		t.Fatalf("BuildUserTaskJSON returned invalid JSON: %v", err)
	}

	// Проверяем наличие обязательных полей
	task, ok := taskData["task"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing 'task' field in JSON")
	}

	// Проверяем поля task
	requiredFields := []string{"id", "domain", "description", "constraints", "input_schema", "output_schema", "examples"}
	for _, field := range requiredFields {
		if _, exists := task[field]; !exists {
			t.Errorf("Missing required field 'task.%s'", field)
		}
	}

	// Проверяем конкретные значения
	if task["id"] != id {
		t.Errorf("Expected task.id = %s, got %v", id, task["id"])
	}
	if task["domain"] != domain {
		t.Errorf("Expected task.domain = %s, got %v", domain, task["domain"])
	}
	if task["description"] != desc {
		t.Errorf("Expected task.description = %s, got %v", desc, task["description"])
	}

	// Проверяем constraints
	constraints, ok := task["constraints"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing 'constraints' field in task")
	}

	expectedConstraints := map[string]interface{}{
		"timeout_ms":     float64(opts.TimeoutMS),
		"mem_mb":         float64(opts.MemMB),
		"max_complexity": opts.MaxComplexity,
	}

	for key, expectedValue := range expectedConstraints {
		if constraints[key] != expectedValue {
			t.Errorf("Expected task.constraints.%s = %v, got %v", key, expectedValue, constraints[key])
		}
	}

	// Проверяем examples
	examplesData, ok := task["examples"].([]interface{})
	if !ok {
		t.Fatal("Missing or invalid 'examples' field in task")
	}

	if len(examplesData) != len(examples) {
		t.Errorf("Expected %d examples, got %d", len(examples), len(examplesData))
	}

	// Проверяем, что JSON можно распарсить обратно в структуру TaskJSON
	var parsedTask TaskJSON
	if err := json.Unmarshal([]byte(result), &parsedTask); err != nil {
		t.Fatalf("Failed to parse JSON back to TaskJSON struct: %v", err)
	}

	// Проверяем, что значения совпадают
	if parsedTask.Task.ID != id {
		t.Errorf("Parsed task ID mismatch: expected %s, got %s", id, parsedTask.Task.ID)
	}
	if parsedTask.Task.Domain != domain {
		t.Errorf("Parsed task domain mismatch: expected %s, got %s", domain, parsedTask.Task.Domain)
	}
	if parsedTask.Task.Description != desc {
		t.Errorf("Parsed task description mismatch: expected %s, got %s", desc, parsedTask.Task.Description)
	}
}

func TestBuildUserTaskJSONEmptyExamples(t *testing.T) {
	// Тест с пустыми примерами
	result := BuildUserTaskJSON("test", "domain", "desc", "{}", "{}", []map[string]string{}, BuildOpts{})

	if result == "" {
		t.Fatal("BuildUserTaskJSON returned empty string")
	}

	var taskData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &taskData); err != nil {
		t.Fatalf("BuildUserTaskJSON returned invalid JSON: %v", err)
	}

	task := taskData["task"].(map[string]interface{})
	examples := task["examples"].([]interface{})
	if len(examples) != 0 {
		t.Errorf("Expected empty examples array, got %d items", len(examples))
	}
}
