# Contributing to transcript-saver

Development guide for contributing to the transcript-saver plugin.

## Prerequisites

- Go 1.26.1+
- Claude Code with plugin support

## Plugin structure

```text
claude-transcript-plugin/
├── .claude-plugin/
│   ├── plugin.json              # Plugin manifest
│   └── marketplace.json         # Marketplace catalog
├── .github/workflows/
│   ├── ci.yml                   # Lint, vet, test on every push/PR
│   └── build.yml                # Cross-compile CI (linux/darwin/windows × amd64/arm64)
├── cmd/save-transcript/
│   └── main.go                  # Entry point
├── internal/transcript/
│   ├── templates/
│   │   ├── default.tmpl         # Structural blocks (header, user, assistant, etc.)
│   │   └── tools/               # One .tmpl file per tool (modular, overridable)
│   │       ├── bash.tmpl
│   │       ├── read.tmpl
│   │       └── …
│   ├── types.go                 # Data types
│   ├── parser.go                # JSONL parsing, classification, tool formatting
│   ├── config.go                # Config file loading and merging
│   └── convert.go               # Template loading, conversion loop
├── hooks/
│   └── hooks.json               # PreCompact Claude hook registration
├── scripts/
│   ├── run                      # Unix platform shim
│   └── run.cmd                  # Windows platform shim
├── skills/save-transcript/
│   └── SKILL.md                 # Manual trigger skill
├── bin/                         # Built binaries (gitignored)
├── go.mod
├── README.md                    # User-facing documentation
└── CONTRIBUTING.md              # This file
```

## Building

```bash
# Local build
go build -o bin/save-transcript ./cmd/save-transcript/

# Cross-compile for a specific platform
GOOS=linux GOARCH=amd64 go build -o bin/save-transcript-linux-amd64 ./cmd/save-transcript/
```

## Testing locally

Load the plugin from your local checkout without installing:

```bash
claude --plugin-dir /path/to/claude-transcript-plugin
```

Test the converter directly against any existing JSONL transcript:

```bash
echo '{"session_id": "test", "transcript_path": "/path/to/transcript.jsonl", "cwd": "/your/project"}' \
  | bin/save-transcript
```

Output will appear in `~/.claude/transcripts/` (or `$TRANSCRIPT_OUTPUT_DIR`).

## Key conventions

- Only `plugin.json` and `marketplace.json` go inside `.claude-plugin/` — everything else lives at the repo root.
- The hook command uses `${CLAUDE_PLUGIN_ROOT}` so paths resolve correctly when installed from the plugin cache.
- The shim scripts detect OS/arch at runtime to support WSL and cross-platform use.
- The converter always exits `0` on error (including panics) so it never blocks compaction.
- Go 1.26.1+ with no external dependencies (stdlib only).

## Adding a built-in template variant

To ship a new variant (e.g. `brief`) with the plugin:

1. Create `internal/transcript/templates/brief.tmpl` with the structural blocks.
2. Rebuild. The `//go:embed` directive picks it up automatically.
3. Users select it with `TRANSCRIPT_TEMPLATE=brief`.

The variant inherits all built-in tool templates. It can override specific tools by defining them in its own file.

## Adding a built-in tool template

Create a file in `internal/transcript/templates/tools/` named `<toolname>.tmpl` (lowercase) containing a single `{{define}}` block:

```go-template
{{define "tool_deploy" -}}
**Tool: Deploy** {{.Input.service}} → `{{.Input.target}}`
{{- end}}
```

Rebuild and it's picked up automatically. The template name must be `tool_` followed by the lowercased tool name.

All tool templates receive the same data:

| Field | Type | Description |
| ------- | ------ | ------------- |
| `.Name` | string | Tool name as-is (e.g. `"Bash"`, `"WebSearch"`) |
| `.Input` | map | All input fields from the tool call, keyed by their JSON field names |

## Truncation rules

Some tools produce very large input values. To keep transcripts readable, the converter truncates specific fields before passing them to templates. The rules are defined in `parser.go`:

```go
var truncationRules = map[string]map[string]int{
    "Write": {"content": 500},
    "Edit":  {"old_string": 200, "new_string": 200},
}
```

To add truncation for a new tool, add an entry to this map.

## CI / GitHub Actions

Two workflows run automatically:

**`ci.yml`** — on every push to `main` and every PR:

- `golangci-lint` — style and correctness checks
- `go vet` — compiler-level static analysis
- `go test -race` — tests with race detector, coverage artifact on PRs

**`build.yml`** — on PRs (compile check) + tagged versions + manual dispatch:

- Cross-compiles all 6 platform targets
- Creates a GitHub Release with binaries on version tags

## Publishing

This repo doubles as both a plugin and a marketplace (via `.claude-plugin/marketplace.json`).

1. Tag a release: `git tag v2.0.0 && git push --tags`
2. GitHub Actions builds binaries for all platforms and creates a release.
3. Users add the marketplace and install:

   ```bash
   /plugin marketplace add IDisposable/claude-transcript-plugin
   /plugin install transcript-saver@transcript-saver
   ```

4. To submit to the official Anthropic plugin directory, see [claude-plugins-official](https://github.com/anthropics/claude-plugins-official).

## Versioning

Bump the `version` field in `.claude-plugin/plugin.json` (and the matching version in `marketplace.json`) before each release. Claude Code uses this to detect updates — if the version doesn't change, users won't see the update.

## Recommended VSCode extensions

This project includes `.vscode/extensions.json` recommending:

- **Go Template Support** (`jinliming2.vscode-go-template`) — syntax highlighting for `.tmpl` files
- **Go** (`golang.go`) — official Go language support
