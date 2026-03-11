package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// jsonlEntry constructs a single JSONL line for test transcripts.
func jsonlEntry(t *testing.T, typ, ts, branch string, meta bool, content interface{}) string {
	t.Helper()

	type msg struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	}
	type entry struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp,omitempty"`
		Message   json.RawMessage `json:"message"`
		IsMeta    bool            `json:"isMeta"`
		GitBranch string          `json:"gitBranch,omitempty"`
	}

	msgBytes, err := json.Marshal(msg{Role: typ, Content: content})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	entryBytes, err := json.Marshal(entry{
		Type:      typ,
		Timestamp: ts,
		Message:   json.RawMessage(msgBytes),
		IsMeta:    meta,
		GitBranch: branch,
	})
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	return string(entryBytes)
}

// writeTranscript writes JSONL lines to a temp file and returns its path.
func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("writing test transcript: %v", err)
	}
	return path
}

// convertToString runs Convert and returns the Markdown output.
func convertToString(t *testing.T, conv *Converter, jsonlPath, cwd string) string {
	t.Helper()
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")
	if err := conv.Convert(jsonlPath, outPath, cwd); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	return string(data)
}

func TestTemplateFuncs(t *testing.T) {
	// execFunc renders a minimal template using templateFuncs and returns the output.
	execFunc := func(t *testing.T, tmplText string) string {
		t.Helper()
		tmpl, err := template.New("test").Funcs(templateFuncs).Parse(tmplText)
		if err != nil {
			t.Fatalf("parse template: %v", err)
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, nil); err != nil {
			t.Fatalf("execute template: %v", err)
		}
		return buf.String()
	}

	t.Run("mdBr returns two trailing spaces for Markdown line break", func(t *testing.T) {
		got := execFunc(t, `line one{{mdBr}}`)
		if got != "line one  " {
			t.Errorf("got %q, want %q", got, "line one  ")
		}
	})

	t.Run("ucEllip returns Unicode ellipsis character", func(t *testing.T) {
		got := execFunc(t, `truncated{{ucEllip}}`)
		if got != "truncated…" {
			t.Errorf("got %q, want %q", got, "truncated…")
		}
	})

	t.Run("mdBr is used in header template output", func(t *testing.T) {
		conv := testConverter(t)
		got := conv.renderToString("header", HeaderData{
			ProjectName:    "test-proj",
			SessionStart:   "2026-03-10T14:30:00Z",
			Cwd:            "/test",
			TranscriptPath: "/test/transcript.jsonl",
		})
		// Each header field line should end with two trailing spaces (mdBr)
		for _, line := range strings.Split(got, "\n") {
			if strings.HasPrefix(line, "**Project:") ||
				strings.HasPrefix(line, "**Started:") ||
				strings.HasPrefix(line, "**Transcript source:") {
				if !strings.HasSuffix(line, "  ") {
					t.Errorf("header line missing mdBr trailing spaces: %q", line)
				}
			}
		}
	})

	t.Run("ucEllip is available in external template overrides", func(t *testing.T) {
		dir := t.TempDir()
		tmpl := `{{define "tool_read"}}**Read** {{.Input.file_path}}{{ucEllip}}{{end}}`
		if err := os.WriteFile(filepath.Join(dir, "read.tmpl"), []byte(tmpl), 0644); err != nil {
			t.Fatal(err)
		}

		c, err := NewConverter("default", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Read",
			Input: map[string]interface{}{"file_path": "/foo.go"},
		}
		got := c.formatToolUse(block)
		if !strings.HasSuffix(strings.TrimSpace(got), "…") {
			t.Errorf("ucEllip not rendered in external template, got: %q", got)
		}
	})
}

func TestNewConverter(t *testing.T) {
	t.Run("default template loads successfully", func(t *testing.T) {
		c, err := NewConverter("default", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("converter is nil")
		}
	})

	t.Run("invalid template name returns error", func(t *testing.T) {
		_, err := NewConverter("nonexistent-template-name", "")
		if err == nil {
			t.Error("expected error for invalid template name")
		}
	})

	t.Run("external tools directory overrides built-in templates", func(t *testing.T) {
		dir := t.TempDir()
		override := `{{define "tool_read"}}**Custom Read** {{.Input.file_path}}{{end}}`
		if err := os.WriteFile(filepath.Join(dir, "read.tmpl"), []byte(override), 0644); err != nil {
			t.Fatal(err)
		}

		c, err := NewConverter("default", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		block := ContentBlock{
			Type:  "tool_use",
			Name:  "Read",
			Input: map[string]interface{}{"file_path": "/test.go"},
		}
		got := c.formatToolUse(block)
		if !strings.Contains(got, "Custom Read") {
			t.Errorf("external override not applied, got: %q", got)
		}
	})

	t.Run("empty tools directory is ignored", func(t *testing.T) {
		dir := t.TempDir() // exists but empty
		c, err := NewConverter("default", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("converter is nil")
		}
	})

	t.Run("nonexistent tools directory is ignored", func(t *testing.T) {
		c, err := NewConverter("default", "/nonexistent/tools/dir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("converter is nil")
		}
	})

	t.Run("external template can add new tool without Go changes", func(t *testing.T) {
		dir := t.TempDir()
		newTool := `{{define "tool_customtool"}}**Custom Tool** key={{.Input.key}}{{end}}`
		if err := os.WriteFile(filepath.Join(dir, "customtool.tmpl"), []byte(newTool), 0644); err != nil {
			t.Fatal(err)
		}

		c, err := NewConverter("default", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		block := ContentBlock{
			Type:  "tool_use",
			Name:  "CustomTool",
			Input: map[string]interface{}{"key": "value"},
		}
		got := c.formatToolUse(block)
		if !strings.Contains(got, "Custom Tool") || !strings.Contains(got, "key=value") {
			t.Errorf("new tool template not loaded, got: %q", got)
		}
	})
}

func TestConvert(t *testing.T) {
	conv := testConverter(t)

	t.Run("basic conversation renders header, user, and assistant", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "main", false, "What is Go?"),
			jsonlEntry(t, "assistant", "2026-03-10T14:30:05Z", "", false, "Go is a programming language."),
		)
		out := convertToString(t, conv, jsonl, "/projects/myapp")

		if !strings.Contains(out, "# Session Transcript: myapp") {
			t.Error("missing project name in header")
		}
		if !strings.Contains(out, "/projects/myapp") {
			t.Error("missing project path in header")
		}
		if !strings.Contains(out, "## User") {
			t.Error("missing user section")
		}
		if !strings.Contains(out, "What is Go?") {
			t.Error("missing user content")
		}
		if !strings.Contains(out, "## Claude") {
			t.Error("missing assistant section")
		}
		if !strings.Contains(out, "Go is a programming language.") {
			t.Error("missing assistant content")
		}
	})

	t.Run("timestamps appear in message sections", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "hello"),
			jsonlEntry(t, "assistant", "2026-03-10T14:30:05Z", "", false, "hi"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "14:30:00") {
			t.Error("user timestamp missing")
		}
		if !strings.Contains(out, "14:30:05") {
			t.Error("assistant timestamp missing")
		}
	})

	t.Run("session start time appears in header", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T09:15:00Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "Started:") || !strings.Contains(out, "2026-03-10T09:15:00Z") {
			t.Error("session start time should appear in header")
		}
	})

	t.Run("empty transcript produces header only", func(t *testing.T) {
		jsonl := writeTranscript(t)
		out := convertToString(t, conv, jsonl, "/projects/empty")
		if !strings.Contains(out, "# Session Transcript:") {
			t.Error("missing header")
		}
		if strings.Contains(out, "## User") || strings.Contains(out, "## Claude") {
			t.Error("empty transcript should contain no messages")
		}
	})

	t.Run("no-timestamp transcript omits Started line", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "", "", false, "no timestamp"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if strings.Contains(out, "Started:") {
			t.Error("header should not show Started when no timestamp available")
		}
		if !strings.Contains(out, "no timestamp") {
			t.Error("message content should still render")
		}
	})

	t.Run("meta entries are filtered out", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", true, "meta content"),
			jsonlEntry(t, "user", "2026-03-10T14:30:01Z", "", false, "real content"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if strings.Contains(out, "meta content") {
			t.Error("meta entries should be filtered")
		}
		if !strings.Contains(out, "real content") {
			t.Error("non-meta entries should be included")
		}
	})

	t.Run("branch changes produce markers", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "main", false, "on main"),
			jsonlEntry(t, "assistant", "2026-03-10T14:30:01Z", "main", false, "ok"),
			jsonlEntry(t, "user", "2026-03-10T14:31:00Z", "feature-x", false, "switched"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "Branch: `main`") {
			t.Error("missing main branch marker")
		}
		if !strings.Contains(out, "Branch: `feature-x`") {
			t.Error("missing feature-x branch marker")
		}
	})

	t.Run("same branch is not repeated", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "main", false, "first"),
			jsonlEntry(t, "user", "2026-03-10T14:30:01Z", "main", false, "second"),
			jsonlEntry(t, "user", "2026-03-10T14:30:02Z", "main", false, "third"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		count := strings.Count(out, "Branch: `main`")
		if count != 1 {
			t.Errorf("branch marker should appear once, appeared %d times", count)
		}
	})

	t.Run("malformed JSONL lines are skipped gracefully", func(t *testing.T) {
		jsonl := writeTranscript(t,
			"this is not json",
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "valid message"),
			"{also bad json",
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "valid message") {
			t.Error("valid entries should render despite surrounding malformed lines")
		}
	})

	t.Run("blank lines in JSONL are skipped", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "hello"),
			"",
			"",
			jsonlEntry(t, "assistant", "2026-03-10T14:30:01Z", "", false, "hi"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "hello") || !strings.Contains(out, "hi") {
			t.Error("messages around blank lines should render")
		}
	})

	t.Run("system noise in user messages is filtered", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "<local-command-caveat>system noise"),
			jsonlEntry(t, "user", "2026-03-10T14:30:01Z", "", false, "real question"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if strings.Contains(out, "local-command-caveat") {
			t.Error("system noise should not appear in output")
		}
		if !strings.Contains(out, "real question") {
			t.Error("real user messages should appear")
		}
	})

	t.Run("assistant tool use is rendered", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": "Let me check that."},
			{"type": "tool_use", "name": "Read", "input": map[string]interface{}{"file_path": "/src/main.go"}},
		}
		jsonl := writeTranscript(t,
			jsonlEntry(t, "assistant", "2026-03-10T14:30:00Z", "", false, blocks),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "Let me check that.") {
			t.Error("text content missing")
		}
		if !strings.Contains(out, "Tool: Read") || !strings.Contains(out, "/src/main.go") {
			t.Error("tool use not rendered")
		}
	})

	t.Run("tool result in user message is rendered", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "tool_result", "tool_use_id": "t1", "content": "file contents here"},
		}
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, blocks),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "Tool Result") || !strings.Contains(out, "file contents here") {
			t.Error("tool result not rendered")
		}
	})

	t.Run("unknown entry types are ignored", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "system", "2026-03-10T14:30:00Z", "", false, "system prompt"),
			jsonlEntry(t, "user", "2026-03-10T14:30:01Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if strings.Contains(out, "system prompt") {
			t.Error("unknown entry types should not appear in output")
		}
		if !strings.Contains(out, "hello") {
			t.Error("known entry types should still render")
		}
	})

	t.Run("empty assistant messages are not rendered", func(t *testing.T) {
		blocks := []map[string]interface{}{
			{"type": "text", "text": ""},
			{"type": "text", "text": "   "},
		}
		jsonl := writeTranscript(t,
			jsonlEntry(t, "assistant", "2026-03-10T14:30:00Z", "", false, blocks),
			jsonlEntry(t, "user", "2026-03-10T14:30:01Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		// Count "## Claude" appearances — the empty one should be skipped
		count := strings.Count(out, "## Claude")
		if count != 0 {
			t.Errorf("empty assistant message should not render, got %d Claude sections", count)
		}
	})

	t.Run("transcript source path appears in header", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "/test")
		if !strings.Contains(out, "Transcript source:") {
			t.Error("transcript source should appear in header")
		}
		if !strings.Contains(out, "transcript.jsonl") {
			t.Error("transcript filename should appear in header")
		}
	})

	t.Run("missing transcript file returns error", func(t *testing.T) {
		dir := t.TempDir()
		outPath := filepath.Join(dir, "output.md")
		err := conv.Convert("/nonexistent/transcript.jsonl", outPath, "/test")
		if err == nil {
			t.Error("expected error for missing transcript file")
		}
	})

	t.Run("cwd determines project name from last path component", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "/home/user/projects/my-awesome-app")
		if !strings.Contains(out, "# Session Transcript: my-awesome-app") {
			t.Error("project name should be derived from last path component of cwd")
		}
	})

	t.Run("empty cwd uses unknown as project name", func(t *testing.T) {
		jsonl := writeTranscript(t,
			jsonlEntry(t, "user", "2026-03-10T14:30:00Z", "", false, "hello"),
		)
		out := convertToString(t, conv, jsonl, "")
		if !strings.Contains(out, "# Session Transcript: unknown") {
			t.Error("empty cwd should produce 'unknown' project name")
		}
	})
}
