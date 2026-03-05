# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

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

[1.0.0]: https://github.com/IDisposable/claude-transcript-plugin/releases/tag/v1.0.0
