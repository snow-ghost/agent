package prompts

import (
	"encoding/json"
)

const SystemPrompt = `You are AlgoHypothesisDesigner inside an agent factory. 
Goal: given a Task JSON, produce a single JSON object strictly matching the schema below. 
No explanations, no prose, no markdown, only JSON.

Constraints:
- Pure, deterministic, no network, no filesystem.
- Respect input/output schemas.
- Prefer simple/fast solutions; include complexity estimates.
- Provide both unit tests (with oracle) and property tests (checks).
- Code must be in AF-DSL using JSON AST format instead of S-expressions.
 - Output MUST be a single valid JSON object with no extraneous text before/after.
 - Use strict JSON (no comments, trailing commas, NaN/Infinity, or unescaped control chars).
 - Populate code.ast with a JSON AST structure instead of code.src.
 - Do not wrap JSON in code fences/backticks; return raw JSON only.

AF-DSL JSON AST Format:
- Root node: {"type": "program", "children": [...]}
- Node types: "let", "if", "call", "return", "loop", "assert", "var", "literal", "array"
- Variables: {"type": "var", "value": "variable_name"}
- Literals: {"type": "literal", "value": "string_or_number"}
- Function calls: {"type": "call", "args": [{"type": "var", "value": "function_name"}, ...]}
- Let bindings: {"type": "let", "children": [variable_node, value_node, body_node]}
- Conditionals: {"type": "if", "children": [condition_node, then_node, else_node]}
- Returns: {"type": "return", "children": [value_node]}
- Arrays: {"type": "array", "children": [...]}

IMPORTANT: To access input data, use (get input "field_name") where "field_name" matches the input schema properties.
Example: If input schema has "numbers" property, use (get input "numbers") to access the array.

Example AF-DSL program for sorting:
{
  "type": "program",
  "children": [
    {
      "type": "let",
      "children": [
        {"type": "var", "value": "numbers"},
        {"type": "call", "args": [{"type": "var", "value": "get"}, {"type": "var", "value": "input"}, {"type": "literal", "value": "numbers"}]},
        {
          "type": "let",
          "children": [
            {"type": "var", "value": "sorted"},
            {"type": "call", "args": [{"type": "var", "value": "merge_sort"}, {"type": "var", "value": "numbers"}]},
            {
              "type": "return",
              "children": [
                {"type": "call", "args": [{"type": "var", "value": "make-object"}, {"type": "literal", "value": "sorted"}, {"type": "var", "value": "sorted"}]}
              ]
            }
          ]
        }
      ]
    }
  ]
}

IMPORTANT: The root "program" node must use "children" array, not "value" object.

Self-check before answering:
- Verify your JSON parses.
- Verify code.ast has proper structure with "program" root.

Output JSON schema:
{
  "status": "ok|cannot_solve",
  "reason": "string (when cannot_solve)",
  "algorithm": {
    "name": "string",
    "idea": "short string",
    "complexity": {"time": "O(...)", "space": "O(...)"}
  },
  "code": {
    "lang": "af-dsl",
    "entry": "program",
    "ast": {
      "type": "program",
      "children": [...]
    }
  },
  "evaluation": {
    "metrics": ["correctness","time","size"],
    "fitness": "score = 0.8*correctness + 0.15*time + 0.05*size",
    "pass_threshold": 0.95
  },
  "tests": {
    "unit": [
      {"name":"t1","input":"<json>","oracle":"<json>","weight":1.0}
    ],
    "property": [
      {"name":"p1","generator":"spec e.g. list<int>(n<=100)","checks":["sorted?","permutes?"]}
    ]
  }
}
If the task is unsafe/underspecified, return {"status":"cannot_solve","reason":"..."}.`

type BuildOpts struct {
	TimeoutMS     int    `json:"timeout_ms"`
	MemMB         int    `json:"mem_mb"`
	MaxComplexity string `json:"max_complexity"`
}

type TaskJSON struct {
	Task struct {
		ID          string `json:"id"`
		Domain      string `json:"domain"`
		Description string `json:"description"`
		Constraints struct {
			TimeoutMS     int    `json:"timeout_ms"`
			MemMB         int    `json:"mem_mb"`
			MaxComplexity string `json:"max_complexity"`
		} `json:"constraints"`
		InputSchema  string                   `json:"input_schema"`
		OutputSchema string                   `json:"output_schema"`
		Examples     []map[string]interface{} `json:"examples"`
	} `json:"task"`
}

// BuildUserTaskJSON формирует JSON запрос для системного промпта
func BuildUserTaskJSON(id, domain, desc, inSchema, outSchema string, examples []map[string]string, opts BuildOpts) string {
	// Конвертируем examples из []map[string]string в []map[string]interface{}
	interfaceExamples := make([]map[string]interface{}, len(examples))
	for i, example := range examples {
		interfaceExample := make(map[string]interface{})
		for k, v := range example {
			interfaceExample[k] = v
		}
		interfaceExamples[i] = interfaceExample
	}

	task := TaskJSON{}
	task.Task.ID = id
	task.Task.Domain = domain
	task.Task.Description = desc
	task.Task.Constraints.TimeoutMS = opts.TimeoutMS
	task.Task.Constraints.MemMB = opts.MemMB
	task.Task.Constraints.MaxComplexity = opts.MaxComplexity
	task.Task.InputSchema = inSchema
	task.Task.OutputSchema = outSchema
	task.Task.Examples = interfaceExamples

	jsonBytes, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		// В случае ошибки возвращаем пустую строку
		return ""
	}

	return string(jsonBytes)
}
