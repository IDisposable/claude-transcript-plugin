# Claude Code Transcript Plugin

## Project Overview

A Claude Code plugin that saves readable Markdown transcripts of sessions before context compaction. It hooks into the `PreCompact` event, reads the session's JSONL transcript, and writes a formatted Markdown file.

## Architecture

- `.claude-plugin/plugin.json` — Plugin manifest (name, version, metadata)
- `.claude-plugin/marketplace.json` — Marketplace catalog for distribution
- `hooks/hooks.json` — Registers a `PreCompact` hook that runs the converter via platform shim
- `scripts/run` — Unix shim that detects OS/arch and executes the correct binary
- `scripts/run.cmd` — Windows shim for native Windows environments
- `cmd/save-transcript/main.go` — Go entry point (reads hook stdin, writes Markdown)
- `internal/transcript/` — Core conversion logic (JSONL parsing, message classification, template rendering)
- `internal/transcript/templates/*.tmpl` — Embedded base templates (structural blocks: header, user, assistant, etc.)
- `internal/transcript/templates/tools/*.tmpl` — Embedded tool templates (one file per tool, modular/overridable)
- `bin/` — Platform-specific compiled binaries (gitignored, built by CI)
- `skills/save-transcript/SKILL.md` — Manual `/transcript-saver:save-transcript` skill for on-demand use

## Key Conventions

- Only `plugin.json` and `marketplace.json` go inside `.claude-plugin/` — all other files live at the repo root
- The hook command uses `${CLAUDE_PLUGIN_ROOT}` so paths resolve correctly when installed from the plugin cache
- The shim scripts detect OS/arch at runtime to support WSL and cross-platform use
- The converter always exits 0 on error so it never blocks compaction
- Go 1.26.1+ with no external dependencies (stdlib only)
- Output goes to `~/.claude/transcripts/` by default, overridable via config/env/flag
- All settings follow: CLI flag > env var > project config > user config > defaults
- Config files: `<project>/.claude/transcript-saver.json` and `~/.claude/transcript-saver.json`

## Templates

- Templates use Go `text/template` syntax with `{{define "name"}}...{{end}}` blocks
- Three-layer loading: built-in tools → base template → external tools directory (later overrides earlier)
- Built-in templates are embedded via `//go:embed` from `internal/transcript/templates/`
- Tool templates live in `templates/tools/` — one `.tmpl` file per tool for modularity
- External tool overrides go in `~/.claude/transcript-tools/` (or `TRANSCRIPT_TOOLS_DIR` / `--tools-dir`)
- Adding a new tool = drop a `<toolname>.tmpl` file in the tools directory, no Go code changes
- Select base template via `TRANSCRIPT_TEMPLATE` env var / `--template` flag
- Template files use the `.tmpl` extension (VSCode: `jinliming2.vscode-go-template` extension recommended)

## Development

- Build locally: `go build -o bin/save-transcript ./cmd/save-transcript/`
- Cross-compile: `GOOS=linux GOARCH=amd64 go build -o bin/save-transcript-linux-amd64 ./cmd/save-transcript/`
- Test locally: `claude --plugin-dir /path/to/claude-transcript-plugin`
- Test converter directly: pipe JSON with `session_id`, `transcript_path`, and `cwd` to `bin/save-transcript`
- Bump `version` in both `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` before each release
- Go 1.26.1+ required

## Git & PR Workflow

- Never add `Co-Authored-By` trailers to commit messages.
- Never add "Generated with" or similar attribution lines to PR descriptions.
- When switching to main, always `git fetch origin main` and `git pull --ff-only` first. Check for uncommitted changes before switching so nothing is lost.

## Repository

- Owner: Marc Brooks @IDisposable (github.com/IDisposable/claude-transcript-plugin)
- License: MIT
