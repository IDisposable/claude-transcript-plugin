package transcript

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl templates/tools/*.tmpl
var defaultTemplates embed.FS

// Converter holds a parsed template and renders JSONL transcripts to Markdown.
type Converter struct {
	tmpl *template.Template
}

// NewConverter creates a Converter with the given template and tools directory.
// templateName can be a built-in name (e.g. "default") or a path to a custom .tmpl file.
// toolsDir is an optional directory of additional tool template overrides (may be empty).
func NewConverter(templateName, toolsDir string) (*Converter, error) {
	tmpl, err := loadTemplate(templateName, toolsDir)
	if err != nil {
		return nil, err
	}
	return &Converter{tmpl: tmpl}, nil
}

// templateFuncs provides functions available to all templates.
var templateFuncs = template.FuncMap{
	// mdBr returns two trailing spaces for a Markdown line break.
	// Use {{mdBr}} at the end of a line instead of invisible trailing spaces.
	"mdBr": func() string { return "  " },
}

// loadTemplate assembles templates in three layers (later definitions override earlier):
//  1. Built-in tool templates (templates/tools/*.tmpl) — shipped defaults
//  2. Base template (templates/<name>.tmpl or custom file) — structural blocks
//  3. External tool templates (<toolsDir>/*.tmpl) — user/third-party overrides
func loadTemplate(name, toolsDir string) (*template.Template, error) {
	tmpl := template.New("").Funcs(templateFuncs)

	// Layer 1: built-in tool templates
	var err error
	tmpl, err = tmpl.ParseFS(defaultTemplates, "templates/tools/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing built-in tool templates: %w", err)
	}

	// Layer 2: base template (structural blocks, may override tool definitions)
	if filepath.IsAbs(name) || strings.HasSuffix(name, ".tmpl") {
		tmpl, err = tmpl.ParseFiles(name)
	} else {
		tmpl, err = tmpl.ParseFS(defaultTemplates, "templates/"+name+".tmpl")
	}
	if err != nil {
		return nil, fmt.Errorf("parsing base template %q: %w", name, err)
	}

	// Layer 3: external tool templates (user/third-party overrides)
	if toolsDir != "" {
		if matches, _ := filepath.Glob(filepath.Join(toolsDir, "*.tmpl")); len(matches) > 0 {
			tmpl, err = tmpl.ParseFiles(matches...)
			if err != nil {
				return nil, fmt.Errorf("parsing external tool templates from %q: %w", toolsDir, err)
			}
		}
	}

	return tmpl, nil
}

// render executes a named template into the given writer.
func (c *Converter) render(w io.Writer, name string, data interface{}) {
	if err := c.tmpl.ExecuteTemplate(w, name, data); err != nil {
		fmt.Fprintf(os.Stderr, "transcript-saver: template %q error: %v\n", name, err)
	}
}

// renderToString executes a named template and returns the result as a string.
func (c *Converter) renderToString(name string, data interface{}) string {
	var buf bytes.Buffer
	c.render(&buf, name, data)
	return buf.String()
}

// Convert reads a JSONL transcript and writes a formatted Markdown file.
func (c *Converter) Convert(jsonlPath, outputPath, cwd string) error {
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		return fmt.Errorf("reading transcript: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	projectName := "unknown"
	if cwd != "" {
		projectName = filepath.Base(cwd)
	}

	// Find session start time from the first entry with a timestamp
	var sessionStart string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Timestamp != "" {
			sessionStart = entry.Timestamp
			break
		}
	}

	// Render header
	c.render(out, "header", HeaderData{
		ProjectName:    projectName,
		SessionStart:   sessionStart,
		Cwd:            cwd,
		TranscriptPath: jsonlPath,
	})

	var currentBranch string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Track branch changes
		if entry.GitBranch != "" && entry.GitBranch != currentBranch {
			currentBranch = entry.GitBranch
			c.render(out, "branch", BranchData{Branch: currentBranch})
		}

		switch entry.Type {
		case "user":
			if entry.IsMeta {
				continue
			}
			c.renderUserMessage(out, entry)
		case "assistant":
			c.renderAssistantMessage(out, entry)
		}
	}

	return nil
}

// renderUserMessage classifies and renders a user message.
func (c *Converter) renderUserMessage(w io.Writer, entry Entry) {
	var msg Message
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return
	}

	kind, text := c.classifyUserMessage(msg.Content)
	if kind == "noise" || kind == "system" || text == "" {
		return
	}

	ts := formatTimestamp(entry.Timestamp)

	switch kind {
	case "human", "mixed":
		c.render(w, "user", MessageData{Timestamp: ts, Content: text})
	case "tool_result":
		c.render(w, "tool_response", MessageData{Timestamp: ts, Content: text})
	}
}

// renderAssistantMessage formats and renders an assistant message.
func (c *Converter) renderAssistantMessage(w io.Writer, entry Entry) {
	var msg Message
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return
	}

	text := c.formatAssistantContent(msg.Content)
	if strings.TrimSpace(text) == "" {
		return
	}

	ts := formatTimestamp(entry.Timestamp)
	c.render(w, "assistant", MessageData{Timestamp: ts, Content: text})
}
