# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [2.1.2] - 2026-03-13

### Changed

- Added `settings.json` for VSCode to apply the [`jinliming2.vscode-go-template`](https://marketplace.visualstudio.com/items?itemName=jinliming2.vscode-go-template) to our `**/*.tmpl` files for syntax rendering

## [2.1.1] - 2026-03-11

### Changed

- Truncated values use an actual ellipsis `…` character
- Fixed CHANGELOG.md (was lagging version number)
- Fixed version numbering and remove redundant uses

### Added

- Added `{{ucEllip}}` templateFunc to easily emit a Unicode ellipsis `…` instead of `...`
- Added description to SKILL.md frontmatter
- Suggest [plugin-dev@claude-plugins-official](https://github.com/anthropics/claude-plugins-official/tree/main/plugins/plugin-dev) for project development
- **Go unit tests**

## [2.0.1] - 2026-03-09

### Changed

- **Rewritten from Python to Go** — eliminates Python runtime dependency; single static binary per platform

### Added

- **Go `text/template` engine** with modular, overridable templates
  - Structural blocks in `internal/transcript/templates/default.tmpl`
  - Per-tool templates in `internal/transcript/templates/tools/*.tmpl` (bash, read, write, edit, grep, glob, agent, websearch, webfetch, skill, askuserquestion)
  - Three-layer loading: built-in tools → base template → external overrides
  - `tool_default` fallback — new tools render automatically without code changes
  - Tool suppression via empty template blocks
  - Custom template function `{{mdBr}}` for Markdown line breaks
- **External tool template directory** (`~/.claude/transcript-tools/`) for user and third-party overrides
- **Config file support** — `transcript-saver.json` at project (`.claude/`) and user (`~/.claude/`) levels, merged per-field
- **CLI flags** (`--template`, `--output-dir`, `--tools-dir`) with 4-way precedence: CLI > env var > config > default
- **Platform shim scripts** — `scripts/run` (Unix) and `scripts/run.cmd` (Windows) detect OS/arch at runtime, supporting WSL dual-boot
- **Cross-platform binaries** — linux/darwin/windows × amd64/arm64 (6 targets)
- **GitHub Actions CI** — `ci.yml` (golangci-lint, go vet, go test -race) and `build.yml` (cross-compile matrix, GitHub Releases on tags)
- **Data-driven truncation rules** for large tool inputs (Write content, Edit strings)
- `CONTRIBUTING.md` with developer documentation (build, test, CI, publishing, conventions)
- `.vscode/extensions.json` recommending Go and Go Template Support extensions

## [1.0.0] - 2026-03-05

### Added

- PreCompact hook that converts JSONL transcripts to Markdown before context compaction
- Human-readable output with timestamped User/Claude message sections
- Git branch change markers in transcript output
- Tool calls formatted with syntax-highlighted commands
- Tool results in collapsible `<details>` blocks
- Thinking blocks in collapsible sections
- System noise filtering
- Manual `/transcript-saver:save-transcript` skill for on-demand use
- Configurable output directory via `TRANSCRIPT_OUTPUT_DIR` environment variable
- Default output to `~/.claude/transcripts/`

[2.0.1]: https://github.com/IDisposable/claude-transcript-plugin/releases/tag/v2.0.1
[1.0.0]: https://github.com/IDisposable/claude-transcript-plugin/releases/tag/v1.0.0
