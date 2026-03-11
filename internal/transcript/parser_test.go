package transcript

import (
	"encoding/json"
	"strings"
	"testing"
)

func testConverter(t *testing.T) *Converter {
	t.Helper()
	c, err := NewConverter("default", "")
	if err != nil {
		t.Fatalf("creating test converter: %v", err)
	}
	return c
}

func mustJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return data
}

// --- Standalone function tests ---

func TestIsSystemNoise(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"local command caveat", "<local-command-caveat>content here", true},
		{"local command stdout", "<local-command-stdout>output", true},
		{"exit command", "<command-name>/exit", true},
		{"plugin command", "<command-name>/plugin stuff", true},
		{"help command", "<command-name>/help", true},
		{"clear command", "<command-name>/clear", true},
		{"normal user text", "Hello world", false},
		{"empty string", "", false},
		{"partial prefix", "<local-command", false},
		{"leading whitespace is trimmed", "  <local-command-caveat>stuff", true},
		{"similar but non-matching prefix", "<local-command-warning>stuff", false},
		{"code that looks like XML", "<div>Hello</div>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSystemNoise(tt.input); got != tt.want {
				t.Errorf("isSystemNoise(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"RFC3339 UTC", "2026-03-10T14:30:00Z", " _14:30:00_"},
		{"RFC3339Nano", "2026-03-10T14:30:00.123456789Z", " _14:30:00_"},
		{"RFC3339 with timezone offset", "2026-03-10T14:30:00+05:00", " _14:30:00_"},
		{"empty string returns empty", "", ""},
		{"invalid format returns empty", "not-a-timestamp", ""},
		{"date only returns empty", "2026-03-10", ""},
		{"unix timestamp returns empty", "1741612200", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTimestamp(tt.input); got != tt.want {
				t.Errorf("formatTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"under limit passes through", "hello", 10, "hello"},
		{"at exact limit passes through", "hello", 5, "hello"},
		{"over limit is cut with ellipsis", "hello world", 5, "hello…"},
		{"empty string passes through", "", 10, ""},
		{"single char over limit", "ab", 1, "a…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.input, tt.maxLen); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestExtractToolResultText(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
		got := extractToolResultText(json.RawMessage(`"file contents here"`))
		if got != "file contents here" {
			t.Errorf("got %q, want %q", got, "file contents here")
		}
	})

	t.Run("array of text blocks", func(t *testing.T) {
		raw := json.RawMessage(`[{"type":"text","text":"line one"},{"type":"text","text":"line two"}]`)
		got := extractToolResultText(raw)
		if got != "line one\nline two" {
			t.Errorf("got %q, want %q", got, "line one\nline two")
		}
	})

	t.Run("non-text block types are ignored", func(t *testing.T) {
		raw := json.RawMessage(`[{"type":"image","text":"ignored"},{"type":"text","text":"kept"}]`)
		got := extractToolResultText(raw)
		if got != "kept" {
			t.Errorf("got %q, want %q", got, "kept")
		}
	})

	t.Run("nil returns empty", func(t *testing.T) {
		got := extractToolResultText(nil)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("empty raw message returns empty", func(t *testing.T) {
		got := extractToolResultText(json.RawMessage{})
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("unparseable JSON falls back to raw string", func(t *testing.T) {
		got := extractToolResultText(json.RawMessage(`not json at all`))
		if got != "not json at all" {
			t.Errorf("got %q, want raw fallback", got)
		}
	})
}

func TestPrepareInput(t *testing.T) {
	t.Run("Write content is truncated to 500 chars", func(t *testing.T) {
		input := map[string]interface{}{
			"file_path": "/foo.go",
			"content":   strings.Repeat("x", 600),
		}
		got := prepareInput(input, "Write")
		content := got["content"].(string)
		if len([]rune(content)) > 501 { // 500 + 1 rune for "…"
			t.Errorf("Write content not truncated: rune len=%d", len([]rune(content)))
		}
		if !strings.HasSuffix(content, "…") {
			t.Error("truncated content should end with …")
		}
	})

	t.Run("Write content under limit is unchanged", func(t *testing.T) {
		input := map[string]interface{}{
			"file_path": "/foo.go",
			"content":   "short content",
		}
		got := prepareInput(input, "Write")
		if got["content"] != "short content" {
			t.Error("short content should not be modified")
		}
	})

	t.Run("Edit old_string and new_string are truncated to 200 chars", func(t *testing.T) {
		input := map[string]interface{}{
			"file_path":  "/foo.go",
			"old_string": strings.Repeat("a", 300),
			"new_string": strings.Repeat("b", 300),
		}
		got := prepareInput(input, "Edit")
		for _, field := range []string{"old_string", "new_string"} {
			s := got[field].(string)
			if len([]rune(s)) > 201 { // 200 + 1 rune for "…"
				t.Errorf("Edit %s not truncated: rune len=%d", field, len([]rune(s)))
			}
			if !strings.HasSuffix(s, "…") {
				t.Errorf("Edit %s should end with …", field)
			}
		}
	})

	t.Run("non-truncated fields are preserved", func(t *testing.T) {
		input := map[string]interface{}{
			"file_path": "/important/path.go",
			"content":   strings.Repeat("x", 600),
		}
		got := prepareInput(input, "Write")
		if got["file_path"] != "/important/path.go" {
			t.Error("file_path should not be modified")
		}
	})

	t.Run("other tools are not truncated", func(t *testing.T) {
		long := strings.Repeat("x", 5000)
		input := map[string]interface{}{"command": long}
		got := prepareInput(input, "Bash")
		if got["command"] != long {
			t.Error("Bash command should not be truncated")
		}
	})

	t.Run("does not mutate the original map", func(t *testing.T) {
		original := strings.Repeat("x", 600)
		input := map[string]interface{}{
			"file_path": "/foo.go",
			"content":   original,
		}
		_ = prepareInput(input, "Write")
		if input["content"] != original {
			t.Error("prepareInput mutated the original map")
		}
	})

	t.Run("non-string fields are preserved", func(t *testing.T) {
		input := map[string]interface{}{
			"file_path": "/foo.go",
			"content":   42, // not a string
		}
		got := prepareInput(input, "Write")
		if got["content"] != 42 {
			t.Error("non-string content should pass through unchanged")
		}
	})
}

// --- Converter method tests ---

func TestClassifyUserMessage(t *testing.T) {
	conv := testConverter(t)

	t.Run("plain string is human input", func(t *testing.T) {
		kind, text := conv.classifyUserMessage(mustJSON(t, "Hello world"))
		if kind != "human" {
			t.Errorf("kind = %q, want %q", kind, "human")
		}
		if text != "Hello world" {
			t.Errorf("text = %q, want %q", text, "Hello world")
		}
	})

	t.Run("system reminder is filtered", func(t *testing.T) {
		kind, text := conv.classifyUserMessage(mustJSON(t, "<system-reminder>You are Claude...</system-reminder>"))
		if kind != "system" {
			t.Errorf("kind = %q, want %q", kind, "system")
		}
		if text != "" {
			t.Errorf("text should be empty for system, got %q", text)
		}
	})

	// Verify all registered noise prefixes are caught
	for _, prefix := range noisePrefixes {
		t.Run("noise prefix: "+prefix, func(t *testing.T) {
			kind, _ := conv.classifyUserMessage(mustJSON(t, prefix+"trailing content"))
			if kind != "noise" {
				t.Errorf("kind = %q, want %q for prefix %q", kind, "noise", prefix)
			}
		})
	}

	t.Run("content blocks with only tool_results", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "tool_result", "tool_use_id": "t1", "content": "result one"},
			{"type": "tool_result", "tool_use_id": "t2", "content": "result two"},
		}
		kind, text := conv.classifyUserMessage(mustJSON(t, blocks))
		if kind != "tool_result" {
			t.Errorf("kind = %q, want %q", kind, "tool_result")
		}
		if !strings.Contains(text, "result one") || !strings.Contains(text, "result two") {
			t.Errorf("tool results not in text: %q", text)
		}
	})

	t.Run("content blocks with only text", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "first part"},
			{"type": "text", "text": "second part"},
		}
		kind, text := conv.classifyUserMessage(mustJSON(t, blocks))
		if kind != "human" {
			t.Errorf("kind = %q, want %q", kind, "human")
		}
		if !strings.Contains(text, "first part") || !strings.Contains(text, "second part") {
			t.Errorf("text blocks not combined: %q", text)
		}
	})

	t.Run("text-only blocks that are noise", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "<local-command-caveat>system stuff"},
		}
		kind, _ := conv.classifyUserMessage(mustJSON(t, blocks))
		if kind != "noise" {
			t.Errorf("kind = %q, want %q", kind, "noise")
		}
	})

	t.Run("mixed text and tool_result", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "user context"},
			{"type": "tool_result", "tool_use_id": "t1", "content": "tool output"},
		}
		kind, text := conv.classifyUserMessage(mustJSON(t, blocks))
		if kind != "mixed" {
			t.Errorf("kind = %q, want %q", kind, "mixed")
		}
		if !strings.Contains(text, "user context") {
			t.Error("mixed should include text blocks")
		}
		if !strings.Contains(text, "tool output") {
			t.Error("mixed should include tool results")
		}
	})

	t.Run("mixed content filters noise from text blocks", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "<local-command-stdout>ignored"},
			{"type": "text", "text": "kept"},
			{"type": "tool_result", "tool_use_id": "t1", "content": "result"},
		}
		_, text := conv.classifyUserMessage(mustJSON(t, blocks))
		if strings.Contains(text, "ignored") {
			t.Error("noise text blocks should be filtered in mixed content")
		}
		if !strings.Contains(text, "kept") {
			t.Error("non-noise text should be kept in mixed content")
		}
	})

	t.Run("invalid JSON returns noise", func(t *testing.T) {
		kind, _ := conv.classifyUserMessage(json.RawMessage(`{bad`))
		if kind != "noise" {
			t.Errorf("kind = %q, want %q for invalid JSON", kind, "noise")
		}
	})
}

func TestFormatAssistantContent(t *testing.T) {
	conv := testConverter(t)

	t.Run("plain string", func(t *testing.T) {
		got := conv.formatAssistantContent(mustJSON(t, "Hello from Claude"))
		if got != "Hello from Claude" {
			t.Errorf("got %q, want plain string", got)
		}
	})

	t.Run("text blocks are joined", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "First paragraph"},
			{"type": "text", "text": "Second paragraph"},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if !strings.Contains(got, "First paragraph") || !strings.Contains(got, "Second paragraph") {
			t.Errorf("text blocks not rendered: %q", got)
		}
	})

	t.Run("empty text blocks are filtered", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": ""},
			{"type": "text", "text": "   "},
			{"type": "text", "text": "visible"},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if strings.TrimSpace(got) != "visible" {
			t.Errorf("empty blocks not filtered, got: %q", got)
		}
	})

	t.Run("tool_use renders with tool template", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "tool_use", "name": "Read", "input": map[string]interface{}{"file_path": "/foo/bar.go"}},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if !strings.Contains(got, "Tool: Read") {
			t.Errorf("tool use not rendered: %q", got)
		}
		if !strings.Contains(got, "/foo/bar.go") {
			t.Errorf("tool input not rendered: %q", got)
		}
	})

	t.Run("thinking block renders", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "thinking", "thinking": "Let me consider this..."},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if !strings.Contains(got, "Thinking") {
			t.Errorf("thinking not rendered: %q", got)
		}
		if !strings.Contains(got, "Let me consider this...") {
			t.Errorf("thinking content missing: %q", got)
		}
	})

	t.Run("empty thinking is skipped", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "thinking", "thinking": ""},
			{"type": "text", "text": "actual response"},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if strings.Contains(got, "Thinking") {
			t.Error("empty thinking should not render")
		}
		if !strings.Contains(got, "actual response") {
			t.Error("text after empty thinking should render")
		}
	})

	t.Run("unknown tool falls back to default template", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "tool_use", "name": "FutureTool", "input": map[string]interface{}{}},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if !strings.Contains(got, "Tool: FutureTool") {
			t.Errorf("unknown tool not rendered with fallback: %q", got)
		}
	})

	t.Run("mixed content types", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "I'll check that."},
			{"type": "tool_use", "name": "Bash", "input": map[string]interface{}{"command": "ls", "description": "list"}},
			{"type": "text", "text": "Here are the results."},
		}
		got := conv.formatAssistantContent(mustJSON(t, blocks))
		if !strings.Contains(got, "I'll check that.") {
			t.Error("first text block missing")
		}
		if !strings.Contains(got, "Tool: Bash") {
			t.Error("tool use missing")
		}
		if !strings.Contains(got, "Here are the results.") {
			t.Error("second text block missing")
		}
	})
}

func TestFormatToolUse(t *testing.T) {
	conv := testConverter(t)

	t.Run("Bash tool with description", func(t *testing.T) {
		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Bash",
			Input: map[string]interface{}{"command": "go test ./...", "description": "Run tests"},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "Tool: Bash") {
			t.Errorf("missing tool name: %q", got)
		}
		if !strings.Contains(got, "go test ./...") {
			t.Errorf("missing command: %q", got)
		}
		if !strings.Contains(got, "Run tests") {
			t.Errorf("missing description: %q", got)
		}
	})

	t.Run("Read tool shows file path", func(t *testing.T) {
		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Read",
			Input: map[string]interface{}{"file_path": "/src/main.go"},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "/src/main.go") {
			t.Errorf("file path missing: %q", got)
		}
	})

	t.Run("Edit tool shows diff-style output", func(t *testing.T) {
		block := ContentBlock{
			Type: "tool_use",
			Name: "Edit",
			Input: map[string]interface{}{
				"file_path":  "/foo.go",
				"old_string": "old line",
				"new_string": "new line",
			},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "- old line") || !strings.Contains(got, "+ new line") {
			t.Errorf("diff format missing: %q", got)
		}
		if !strings.Contains(got, "/foo.go") {
			t.Errorf("file path missing: %q", got)
		}
	})

	t.Run("Grep with explicit path", func(t *testing.T) {
		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Grep",
			Input: map[string]interface{}{"pattern": "TODO", "path": "/src"},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "TODO") || !strings.Contains(got, "/src") {
			t.Errorf("Grep not rendered correctly: %q", got)
		}
	})

	t.Run("Grep defaults path to dot", func(t *testing.T) {
		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Grep",
			Input: map[string]interface{}{"pattern": "TODO"},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "path=`.`") {
			t.Errorf("Grep should default path to '.': %q", got)
		}
	})

	t.Run("Agent tool shows subagent type and description", func(t *testing.T) {
		block := ContentBlock{
			Type: "tool_use",
			Name: "Agent",
			Input: map[string]interface{}{
				"subagent_type": "Explore",
				"description":   "Find config files",
			},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "Explore") || !strings.Contains(got, "Find config files") {
			t.Errorf("Agent not rendered correctly: %q", got)
		}
	})

	t.Run("Write tool shows truncated content", func(t *testing.T) {
		block := ContentBlock{
			Type: "tool_use",
			Name: "Write",
			Input: map[string]interface{}{
				"file_path": "/out.go",
				"content":   strings.Repeat("x", 600),
			},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "/out.go") {
			t.Errorf("file path missing: %q", got)
		}
		// Content should be truncated
		if strings.Contains(got, strings.Repeat("x", 600)) {
			t.Error("Write content should be truncated")
		}
	})

	t.Run("unknown tool falls back to default template", func(t *testing.T) {
		block := ContentBlock{
			Type:  "tool_use",
			Name:  "SomeFutureTool",
			Input: map[string]interface{}{"key": "value"},
		}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "Tool: SomeFutureTool") {
			t.Errorf("fallback not applied: %q", got)
		}
	})

	t.Run("empty name becomes unknown", func(t *testing.T) {
		block := ContentBlock{Type: "tool_use", Name: "", Input: map[string]interface{}{}}
		got := conv.formatToolUse(block)
		if !strings.Contains(got, "unknown") {
			t.Errorf("empty name should become 'unknown': %q", got)
		}
	})
}

func TestFormatToolResult(t *testing.T) {
	conv := testConverter(t)

	t.Run("normal result wraps in template", func(t *testing.T) {
		block := ContentBlock{
			Type:    "tool_result",
			Content: json.RawMessage(`"file contents here"`),
		}
		got := conv.formatToolResult(block)
		if !strings.Contains(got, "Tool Result") {
			t.Errorf("missing template wrapper: %q", got)
		}
		if !strings.Contains(got, "file contents here") {
			t.Errorf("missing content: %q", got)
		}
	})

	t.Run("empty result returns empty string", func(t *testing.T) {
		block := ContentBlock{
			Type:    "tool_result",
			Content: json.RawMessage(`""`),
		}
		got := conv.formatToolResult(block)
		if got != "" {
			t.Errorf("empty result should return empty, got: %q", got)
		}
	})

	t.Run("whitespace-only result returns empty string", func(t *testing.T) {
		block := ContentBlock{
			Type:    "tool_result",
			Content: json.RawMessage(`"   \n  "`),
		}
		got := conv.formatToolResult(block)
		if got != "" {
			t.Errorf("whitespace-only result should return empty, got: %q", got)
		}
	})

	t.Run("long result is truncated at 2000 chars", func(t *testing.T) {
		long := strings.Repeat("x", 3000)
		block := ContentBlock{
			Type:    "tool_result",
			Content: mustJSON(t, long),
		}
		got := conv.formatToolResult(block)
		if !strings.Contains(got, "truncated") {
			t.Error("long result should indicate truncation")
		}
		if strings.Contains(got, long) {
			t.Error("full 3000-char content should not appear")
		}
	})
}

func TestFormatThinking(t *testing.T) {
	conv := testConverter(t)

	t.Run("short thinking is preserved", func(t *testing.T) {
		got := conv.formatThinking("Let me think about this carefully")
		if !strings.Contains(got, "Thinking") {
			t.Errorf("missing template wrapper: %q", got)
		}
		if !strings.Contains(got, "Let me think about this carefully") {
			t.Errorf("thinking content missing: %q", got)
		}
	})

	t.Run("long thinking is truncated at 500 chars", func(t *testing.T) {
		long := strings.Repeat("think ", 200) // 1200 chars
		got := conv.formatThinking(long)
		if strings.Contains(got, long) {
			t.Error("full thinking content should not appear")
		}
		if !strings.Contains(got, "…") {
			t.Error("truncated thinking should end with …")
		}
	})
}
