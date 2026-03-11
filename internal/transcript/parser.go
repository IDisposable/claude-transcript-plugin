package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Prefixes that indicate system/command noise, not real conversation.
var noisePrefixes = []string{
	"<local-command-caveat>",
	"<local-command-stdout>",
	"<command-name>/exit",
	"<command-name>/plugin",
	"<command-name>/help",
	"<command-name>/clear",
}

// isSystemNoise returns true for system/command noise that isn't real conversation.
func isSystemNoise(text string) bool {
	stripped := strings.TrimSpace(text)
	for _, prefix := range noisePrefixes {
		if strings.HasPrefix(stripped, prefix) {
			return true
		}
	}
	return false
}

// classifyUserMessage classifies a user message as human input, tool result, or noise.
// Returns (kind, formattedText) where kind is "human", "tool_result", "mixed", "system", or "noise".
func (c *Converter) classifyUserMessage(raw json.RawMessage) (string, string) {
	// Try as plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if isSystemNoise(s) {
			return "noise", ""
		}
		if strings.HasPrefix(strings.TrimSpace(s), "<system-reminder>") {
			return "system", ""
		}
		return "human", s
	}

	// Try as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "noise", ""
	}

	hasToolResult := false
	hasText := false
	for _, b := range blocks {
		switch b.Type {
		case "tool_result":
			hasToolResult = true
		case "text":
			hasText = true
		}
	}

	if hasToolResult && !hasText {
		var parts []string
		for _, b := range blocks {
			if b.Type == "tool_result" {
				if formatted := c.formatToolResult(b); formatted != "" {
					parts = append(parts, formatted)
				}
			}
		}
		return "tool_result", strings.Join(parts, "\n\n")
	}

	if hasText && !hasToolResult {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" {
				texts = append(texts, b.Text)
			}
		}
		combined := strings.Join(texts, "\n")
		if isSystemNoise(combined) {
			return "noise", ""
		}
		return "human", combined
	}

	// Mixed content
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if !isSystemNoise(b.Text) {
				parts = append(parts, b.Text)
			}
		case "tool_result":
			if formatted := c.formatToolResult(b); formatted != "" {
				parts = append(parts, formatted)
			}
		}
	}
	return "mixed", strings.Join(parts, "\n\n")
}

// formatAssistantContent formats assistant message content to readable markdown.
func (c *Converter) formatAssistantContent(raw json.RawMessage) string {
	// Try as plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return string(raw)
	}

	var parts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		case "tool_use":
			if rendered := c.formatToolUse(block); strings.TrimSpace(rendered) != "" {
				parts = append(parts, rendered)
			}
		case "thinking":
			if block.Thinking != "" {
				parts = append(parts, c.formatThinking(block.Thinking))
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

// truncationRules defines which input fields to truncate per tool, and to what length.
var truncationRules = map[string]map[string]int{
	"Write": {"content": 500},
	"Edit":  {"old_string": 200, "new_string": 200},
}

// formatToolUse renders a tool_use block using the appropriate sub-template.
// Template lookup: tries "tool_<lowercase name>", falls back to "tool_default".
// Adding a new tool only requires a new {{define}} block in the template.
func (c *Converter) formatToolUse(block ContentBlock) string {
	name := block.Name
	if name == "" {
		name = "unknown"
	}

	data := ToolData{
		Name:  name,
		Input: prepareInput(block.Input, name),
	}

	tmplName := "tool_" + strings.ToLower(name)
	if c.tmpl.Lookup(tmplName) != nil {
		return c.renderToString(tmplName, data)
	}
	return c.renderToString("tool_default", data)
}

// prepareInput clones the input map and applies truncation rules.
func prepareInput(inp map[string]interface{}, toolName string) map[string]interface{} {
	clone := make(map[string]interface{}, len(inp))
	for k, v := range inp {
		clone[k] = v
	}

	if rules, ok := truncationRules[toolName]; ok {
		for field, maxLen := range rules {
			if s, ok := clone[field].(string); ok {
				clone[field] = truncate(s, maxLen)
			}
		}
	}

	return clone
}

// formatToolResult formats a tool_result block compactly.
func (c *Converter) formatToolResult(block ContentBlock) string {
	resultText := extractToolResultText(block.Content)
	if strings.TrimSpace(resultText) == "" {
		return ""
	}
	if runes := []rune(resultText); len(runes) > 2000 {
		resultText = string(runes[:2000]) + "\n… (truncated)"
	}
	return c.renderToString("tool_result", ToolResultData{Content: resultText})
}

// formatThinking formats a thinking block with truncation.
func (c *Converter) formatThinking(thinking string) string {
	preview := truncate(thinking, 500)
	return c.renderToString("thinking", ThinkingData{Preview: preview})
}

// formatTimestamp formats an ISO timestamp to " _HH:MM:SS_" or returns "".
func formatTimestamp(ts string) string {
	if ts == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, ts); err == nil {
			return fmt.Sprintf(" _%s_", t.Format("15:04:05"))
		}
	}
	return ""
}

// extractToolResultText extracts text content from a tool_result's content field.
func extractToolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return string(raw)
	}

	var texts []string
	for _, b := range blocks {
		if b.Type == "text" {
			texts = append(texts, b.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// truncate shortens a string to maxLen runes, appending "…" if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}
