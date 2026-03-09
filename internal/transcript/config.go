package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds plugin configuration loaded from transcript-saver.json files.
type Config struct {
	Template  string `json:"template,omitempty"`
	OutputDir string `json:"output_dir,omitempty"`
	ToolsDir  string `json:"tools_dir,omitempty"`
}

// LoadConfig loads configuration by merging project-level and user-level config files.
// Project settings override user settings on a per-field basis.
// Search path:
//  1. <cwd>/.claude/transcript-saver.json  (project)
//  2. ~/.claude/transcript-saver.json       (user)
func LoadConfig(cwd string) Config {
	var cfg Config

	// User-level config (lowest priority, loaded first)
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".claude", "transcript-saver.json")
		if c, err := readConfigFile(userPath); err == nil {
			cfg = c
		}
	}

	// Project-level config (overrides user on a per-field basis)
	if cwd != "" {
		projPath := filepath.Join(cwd, ".claude", "transcript-saver.json")
		if c, err := readConfigFile(projPath); err == nil {
			cfg.merge(c)
		}
	}

	return cfg
}

// merge applies non-empty fields from other onto c.
func (c *Config) merge(other Config) {
	if other.Template != "" {
		c.Template = other.Template
	}
	if other.OutputDir != "" {
		c.OutputDir = other.OutputDir
	}
	if other.ToolsDir != "" {
		c.ToolsDir = other.ToolsDir
	}
}

// readConfigFile reads and parses a single config file. Returns an error if
// the file doesn't exist or can't be parsed.
func readConfigFile(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}
