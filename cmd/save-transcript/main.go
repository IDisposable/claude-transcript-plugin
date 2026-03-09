package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IDisposable/claude-transcript-plugin/internal/transcript"
)

var (
	flagTemplate  = flag.String("template", "", "template name or path to .tmpl file (default: \"default\")")
	flagOutputDir = flag.String("output-dir", "", "output directory for transcripts (default: ~/.claude/transcripts/)")
	flagToolsDir  = flag.String("tools-dir", "", "directory of additional tool template overrides (default: ~/.claude/transcript-tools/)")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "transcript-saver: panic — %v\n", r)
			os.Exit(0) // Exit cleanly even on panic — never block compaction
		}
	}()

	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "transcript-saver: error — %v\n", err)
	}
	os.Exit(0) // Always exit cleanly — never block compaction
}

// resolve returns the first non-empty value from: CLI flag, env var, config file, fallback.
func resolve(flagVal, envKey, configVal, fallback string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if configVal != "" {
		return configVal
	}
	return fallback
}

func run() error {
	var input transcript.HookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return fmt.Errorf("reading hook input: %w", err)
	}

	if input.TranscriptPath == "" {
		return fmt.Errorf("no transcript path provided")
	}
	if _, err := os.Stat(input.TranscriptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "transcript-saver: no transcript found at %s\n", input.TranscriptPath)
		return nil
	}

	// Load config files: project (.claude/transcript-saver.json) > user (~/.claude/transcript-saver.json)
	cfg := transcript.LoadConfig(input.Cwd)

	// Select template: CLI flag > env var > config > "default"
	templateName := resolve(*flagTemplate, "TRANSCRIPT_TEMPLATE", cfg.Template, "default")

	// Tools directory: CLI flag > env var > config > ~/.claude/transcript-tools/ (if exists)
	toolsDir := resolve(*flagToolsDir, "TRANSCRIPT_TOOLS_DIR", cfg.ToolsDir, "")
	if toolsDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			candidate := filepath.Join(home, ".claude", "transcript-tools")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				toolsDir = candidate
			}
		}
	}

	converter, err := transcript.NewConverter(templateName, toolsDir)
	if err != nil {
		return fmt.Errorf("loading template %q: %w", templateName, err)
	}

	// Output directory: CLI flag > env var > config > ~/.claude/transcripts/
	outputDir := resolve(*flagOutputDir, "TRANSCRIPT_OUTPUT_DIR", cfg.OutputDir, "")
	if outputDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		outputDir = filepath.Join(home, ".claude", "transcripts")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate output filename
	timestamp := time.Now().Format("2006-01-02_150405")
	shortID := "none"
	if input.SessionID != "" && input.SessionID != "unknown" {
		shortID = input.SessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
	}
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s.md", timestamp, shortID))

	if err := converter.Convert(input.TranscriptPath, outputFile, input.Cwd); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "transcript-saver: saved %s\n", outputFile)
	return nil
}
