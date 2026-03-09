package transcript

import "encoding/json"

// HookInput represents the JSON received from Claude Code hooks via stdin.
type HookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
}

// Entry represents a single line in the JSONL transcript.
type Entry struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
	IsMeta    bool            `json:"isMeta"`
	GitBranch string          `json:"gitBranch"`
}

// Message represents the message field within an Entry.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentBlock represents a block within a message's content array.
type ContentBlock struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Thinking  string                 `json:"thinking,omitempty"`
	Content   json.RawMessage        `json:"content,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
}

// --- Template data types ---

// HeaderData holds data for the "header" template.
type HeaderData struct {
	ProjectName    string
	SessionStart   string
	Cwd            string
	TranscriptPath string
}

// BranchData holds data for the "branch" template.
type BranchData struct {
	Branch string
}

// MessageData holds data for message templates (user, assistant, tool_response).
type MessageData struct {
	Timestamp string
	Content   string
}

// ToolData holds data for any tool_use template.
// Templates access fields via {{.Name}} and {{.Input.fieldname}}.
// To add support for a new tool, just add a {{define "tool_<lowercase>"}} block
// to the template — no Go code changes needed.
type ToolData struct {
	Name  string                 // Tool name (e.g. "Bash", "Read")
	Input map[string]interface{} // Tool input fields, passed directly to template
}

// ToolResultData holds data for the "tool_result" template.
type ToolResultData struct {
	Content string
}

// ThinkingData holds data for the "thinking" template.
type ThinkingData struct {
	Preview string
}
