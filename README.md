# transcript-saver

[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/IDisposable/claude-transcript-plugin/master/LICENSE) [![Build Status](https://github.com/IDisposable/claude-transcript-plugin/actions/workflows/build.yml/badge.svg)](https://github.com/IDisposable/claude-transcript-plugin/actions/workflows/build.yml)
[![CI](https://github.com/IDisposable/claude-transcript-plugin/actions/workflows/ci.yml/badge.svg)](https://github.com/IDisposable/claude-transcript-plugin/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/IDisposable/claude-transcript-plugin/graph/badge.svg)](https://codecov.io/gh/IDisposable/claude-transcript-plugin)

A Claude Code plugin that automatically saves readable Markdown transcripts of your sessions before context compaction.

## What It Does

When Claude Code compacts your conversation context (either automatically or via `/compact`), this plugin intercepts the `PreCompact` event and converts the full JSONL transcript into a clean, human-readable Markdown file.

**Output includes:**

- Timestamped User / Claude message sections
- Git branch change markers
- Tool calls with syntax-highlighted commands
- Tool results in collapsible `<details>` blocks
- Thinking blocks in collapsible sections
- System noise filtered out

### Why?

See <img width="1433" height="706" alt="image" src="https://github.com/user-attachments/assets/49b95b7d-4282-4bd0-9427-2e7c1bb6110b" />

### TL;DR

```bash
/plugin marketplace add IDisposable/claude-transcript-plugin
/plugin install transcript-saver@transcript-saver
```

```claude
/compact
```

Transcripts by default are in `~/.claude/transcripts/` or `C:\User\you\.claude\templates\`

## Installation

### From Marketplace

Add this repository as a plugin marketplace, then install:

```bash
/plugin marketplace add IDisposable/claude-transcript-plugin
/plugin install transcript-saver@transcript-saver
```

Or install directly from GitHub:

```bash
claude plugin install transcript-saver@transcript-saver --source github:IDisposable/claude-transcript-plugin
```

### Binaries

Download the pre-built binary for your platform from [Releases](https://github.com/IDisposable/claude-transcript-plugin/releases) and place it in the `bin/` directory:

```cmd
bin/save-transcript-<os>-<arch>[.exe]
```

Supported platforms: `linux`, `darwin`, `windows` × `amd64`, `arm64`.

Or build from source (requires Go 1.26.1+):

```bash
go build -o bin/save-transcript ./cmd/save-transcript/
```

### Local Development

```bash
claude --plugin-dir /path/to/claude-transcript-plugin
```

## Configuration

All settings follow the same precedence order:

`CLI flag > environment variable > config file > default`

There are two ways to configure the plugin: config files (recommended) and environment variables.

### Config files

Create a `transcript-saver.json` file to set defaults. The plugin checks two locations and merges them (project overrides user):

| Location | Scope |
| ---------- | ------- |
| `<project>/.claude/transcript-saver.json` | Per-project settings |
| `~/.claude/transcript-saver.json` | User-wide defaults |

```json
{
  "template": "brief",
  "output_dir": "~/project-transcripts",
  "tools_dir": "~/.claude/transcript-tools"
}
```

All fields are optional. Only the fields you specify are applied; the rest use defaults.

#### Example: per-project config

```bash
# In your project root
mkdir -p .claude
echo '{"template": "brief", "output_dir": "./transcripts"}' > .claude/transcript-saver.json
```

#### Example: user-wide defaults

```bash
echo '{"template": "default", "tools_dir": "~/.claude/transcript-tools"}' > ~/.claude/transcript-saver.json
```

Project settings override user settings on a per-field basis. If your user config sets `template` to `"default"` but a project config sets it to `"brief"`, that project uses `"brief"` while other projects use `"default"`.

### Environment variables

Settings can also be configured via environment variables, either in your shell or in Claude Code's settings:

| Variable | Purpose |
| ---------- | --------- |
| `TRANSCRIPT_TEMPLATE` | Template name or path to custom `.tmpl` file |
| `TRANSCRIPT_OUTPUT_DIR` | Output directory for transcript files |
| `TRANSCRIPT_TOOLS_DIR` | Directory of external tool template overrides |

```bash
export TRANSCRIPT_TEMPLATE=brief
export TRANSCRIPT_OUTPUT_DIR=~/my-transcripts
export TRANSCRIPT_TOOLS_DIR=~/my-tool-templates
```

You can also set these in Claude Code's own settings file for per-project or global use:

```json
// ~/.claude/settings.json (global) or <project>/.claude/settings.json
{
  "env": {
    "TRANSCRIPT_TEMPLATE": "brief"
  }
}
```

### CLI flags

All settings can be overridden on the command line (highest precedence):

```bash
echo '...' | save-transcript --template brief --output-dir /tmp/out --tools-dir ~/tools
```

Or via the hook command in `hooks/hooks.json`:

```json
"command": "${CLAUDE_PLUGIN_ROOT}/scripts/run --template brief"
```

### Settings reference

| Setting | CLI flag | Env var | Config key | Default |
| --------- | ---------- | --------- | ------------ | --------- |
| Template | `--template` | `TRANSCRIPT_TEMPLATE` | `template` | `"default"` |
| Output directory | `--output-dir` | `TRANSCRIPT_OUTPUT_DIR` | `output_dir` | `~/.claude/transcripts/` |
| Tools directory | `--tools-dir` | `TRANSCRIPT_TOOLS_DIR` | `tools_dir` | `~/.claude/transcript-tools/` (if exists) |

## Templates

Templates use [Go `text/template`](https://pkg.go.dev/text/template) syntax and control how the Markdown output is formatted. The built-in templates live in `internal/transcript/templates/` and are embedded into the binary at compile time.

### Template structure

A template file is a single `.tmpl` file containing multiple named blocks defined with `{{define "name"}}...{{end}}`. Each block renders one kind of element:

| Block | Data fields | Purpose |
| ------- | ------------- | --------- |
| `header` | `.ProjectName` `.SessionStart` `.Cwd` `.TranscriptPath` | Document header |
| `branch` | `.Branch` | Git branch change marker |
| `user` | `.Timestamp` `.Content` | User message |
| `tool_response` | `.Timestamp` `.Content` | Tool result returned to Claude |
| `assistant` | `.Timestamp` `.Content` | Claude's response |
| `tool_result` | `.Content` | Collapsible tool output detail |
| `thinking` | `.Preview` | Collapsible thinking block |
| `tool_*` | `.Name` `.Input` | Tool-specific formatting (see below) |
| `tool_default` | `.Name` `.Input` | Fallback for unknown tools |

### Template architecture

Templates are loaded in three layers, where later definitions override earlier ones:

1. **Built-in tool templates** (`internal/transcript/templates/tools/*.tmpl`) — one file per tool, shipped with the binary
2. **Base template** (`internal/transcript/templates/<name>.tmpl`) — structural blocks (header, user, assistant, etc.)
3. **External tool templates** (`~/.claude/transcript-tools/*.tmpl`) — user or third-party overrides

This means you can override any single tool's rendering without touching the rest of the template.

### Creating a custom base template

A custom base template only needs to define the structural blocks — tool rendering is inherited from the built-in tool templates unless you explicitly override them.

1. Copy the default template as a starting point:

   ```bash
   cp internal/transcript/templates/default.tmpl ~/my-template.tmpl
   ```

2. Edit the blocks you want to change. For example, to make user messages more prominent:

   ```markdown go-template
   {{define "user" -}}
   ---
   # USER {{.Timestamp}}

   {{.Content}}

   {{end}}
   ```

3. Use it:

   ```bash
   export TRANSCRIPT_TEMPLATE=~/my-template.tmpl
   ```

### Suppressing tools

To hide a tool's output entirely, define an empty template block for it. Any tool template that renders to empty/whitespace is silently skipped.

In a custom base template:

```go-template
{{define "tool_write" -}}{{- end}}
{{define "tool_edit" -}}{{- end}}
```

Or as an external override file (e.g. `~/.claude/transcript-tools/write.tmpl`):

```go-template
{{define "tool_write" -}}{{- end}}
```

This works for any tool — built-in or third-party. For example, a `brief.tmpl` variant that omits generated code, file writes, and thinking blocks might define:

```go-template
{{define "tool_write" -}}{{- end}}
{{define "tool_edit" -}}{{- end}}
{{define "thinking" -}}{{- end}}
```

All other tools inherit their default rendering.

### Template functions

There may be template rendering that is difficult or non-obvious to express in markdown syntax. You can use these functions to represent the specified outputs.

| Function | Output | Purpose |
| ---------- | -------- | --------- |
| `{{mdBr}}` | `  ` (two spaces) | Markdown line break — use instead of invisible trailing spaces |
| `{{ucEllip}}` | `…` (Unicode ellipsis) | [Unicode ellipsis](https://www.compart.com/en/unicode/U+2026) - use to easily indicate truncations with `…` instead of `...` |

### Adding a built-in template variant

To ship a new variant (e.g. `brief`) with the plugin:

1. Create `internal/transcript/templates/brief.tmpl` with the structural blocks.
2. Rebuild. The `//go:embed` directive picks it up automatically.
3. Users select it with `TRANSCRIPT_TEMPLATE=brief`.

The variant inherits all built-in tool templates. It can override specific tools by defining them in its own file.

## Adding Support for New Tools

When Claude Code adds a new tool, the converter automatically handles it via the `tool_default` template, which renders `**Tool: ToolName**`. To give a new tool richer formatting, you only need to add a template file — no Go code changes required.

### How tool template lookup works

1. The converter lowercases the tool name and looks for a template named `tool_<name>`.
2. If found, it renders that template. If not, it falls back to `tool_default`.
3. All tool templates receive the same data structure:

   | Field | Type | Description |
   | ------- | ------ | ------------- |
   | `.Name` | string | Tool name as-is (e.g. `"Bash"`, `"WebSearch"`) |
   | `.Input` | map | All input fields from the tool call, keyed by their JSON field names |

### Adding a tool template (built-in)

Create a file in `internal/transcript/templates/tools/` named `<toolname>.tmpl` (lowercase) containing a single `{{define}}` block:

```markdown go-template
{{define "tool_deploy" -}}
**Tool: Deploy** {{.Input.service}} → `{{.Input.target}}`
{{- end}}
```

Rebuild and it's picked up automatically.

### Adding a tool template (external / third-party)

Drop a `.tmpl` file into the tools directory. The default location is `~/.claude/transcript-tools/`, overridable with `--tools-dir` or `TRANSCRIPT_TOOLS_DIR`.

```bash
# Create the tools directory
mkdir -p ~/.claude/transcript-tools

# Add a template for a custom MCP tool
cat > ~/.claude/transcript-tools/deploy.tmpl << 'EOF'
{{define "tool_deploy" -}}
**Tool: Deploy** {{.Input.service}} → `{{.Input.target}}`
{{- end}}
EOF
```

External templates override built-in ones with the same name, so you can also use this to customize how existing tools (like `Bash` or `Read`) are rendered.

### Convention for tool authors

If you're building a Claude Code tool (MCP server, plugin, etc.) and want to provide a default transcript rendering, ship a `.tmpl` file alongside your tool with the following convention:

- **File name:** `<toolname>.tmpl` (lowercase, matching your tool's name)
- **Contents:** A single `{{define "tool_<toolname>"}}` block
- **Data available:** `{{.Name}}` (tool name) and `{{.Input.<field>}}` (all input fields from the tool call)
- **Install instructions:** Tell users to copy the file to `~/.claude/transcript-tools/`

### Accessing input fields

Input fields are accessed via `{{.Input.field_name}}` using the exact JSON key names from the tool call. Common patterns:

```go-template
{{.Input.file_path}}             Access a string field
{{with .Input.description}}      Conditional: only render if field exists
  — {{.}}
{{end}}
{{with .Input.path}}             Provide a default value
  {{.}}
{{else}}
  .
{{end}}
{{range .Input.items}}           Iterate over an array field
  {{index . "key"}}
{{end}}
```

### Truncation

Some tools produce very large input values. To keep transcripts readable, the converter truncates specific fields before passing them to templates. The rules are defined in `parser.go`:

```go
var truncationRules = map[string]map[string]int{
    "Write": {"content": 500},
    "Edit":  {"old_string": 200, "new_string": 200},
}
```

To add truncation for a new tool, add an entry to this map.

## How It Works

| Component | Purpose |
| --------- | ------------- |
| `hooks/hooks.json` | Registers a `PreCompact` hook |
| `scripts/run` | Unix shim — detects OS/arch, runs correct binary |
| `scripts/run.cmd` | Windows shim for native Windows |
| `cmd/save-transcript/` | Go entry point — reads hook stdin, writes Markdown |
| `internal/transcript/` | JSONL parsing, classification, template rendering |
| `internal/transcript/templates/` | Embedded `.tmpl` template files |
| `skills/save-transcript/` | Manual `/transcript-saver:save-transcript` skill |

The hook receives the session's `transcript_path` from Claude Code via stdin JSON, reads the JSONL file, and writes a formatted Markdown file. It exits cleanly on any error (including panics) so it never blocks compaction.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, plugin structure, CI details, and development conventions.

## Example Output

```markdown
# Session Transcript: my-project

**Started:** 2026-03-05T15:21:47.027Z
**Project:** `/home/user/my-project`

---

> *Branch: `main`*

## User _15:21:47_

Please review the authentication module.

---

## Claude _15:21:50_

I'll start by reading the auth files.

**Tool: Read** `src/auth/handler.ts`

---

### Tool Response _15:21:51_

<details><summary>Tool Result</summary>
...
</details>
```

## License

MIT
