# Claude Code Transcript Plugin

## Project Overview

A Claude Code plugin that saves readable Markdown transcripts of sessions before context compaction. It hooks into the `PreCompact` event, reads the session's JSONL transcript, and writes a formatted Markdown file.

## Architecture

- `.claude-plugin/plugin.json` — Plugin manifest (name, version, metadata)
- `.claude-plugin/marketplace.json` — Marketplace catalog for distribution
- `hooks/hooks.json` — Registers a `PreCompact` hook that runs the converter script
- `scripts/save-transcript.py` — Python script that converts JSONL → Markdown (receives hook input via stdin JSON)
- `skills/save-transcript/SKILL.md` — Manual `/transcript-saver:save-transcript` skill for on-demand use

## Key Conventions

- Only `plugin.json` and `marketplace.json` go inside `.claude-plugin/` — all other files live at the repo root
- The hook command uses `${CLAUDE_PLUGIN_ROOT}` so paths resolve correctly when installed from the plugin cache
- The script always exits 0 on error so it never blocks compaction
- Python 3.8+ with no external dependencies (stdlib only: json, os, sys, datetime, pathlib)
- Output goes to `~/.claude/transcripts/` by default, overridable via `TRANSCRIPT_OUTPUT_DIR` env var

## Development

- Test locally: `claude --plugin-dir /path/to/claude-transcript-plugin`
- Test script directly: pipe JSON with `session_id`, `transcript_path`, and `cwd` to `scripts/save-transcript.py`
- Bump `version` in both `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` before each release

## Git & PR Workflow

- Never add `Co-Authored-By` trailers to commit messages.
- Never add "Generated with" or similar attribution lines to PR descriptions.
- When switching to main, always `git fetch origin main` and `git pull --ff-only` first. Check for uncommitted changes before switching so nothing is lost.

## Repository

- Owner: Marc Brooks @IDisposable (github.com/IDisposable/claude-transcript-plugin)
- License: MIT
