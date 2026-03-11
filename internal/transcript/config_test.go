package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigMerge(t *testing.T) {
	t.Run("all fields override", func(t *testing.T) {
		base := Config{Template: "default", OutputDir: "/base/out", ToolsDir: "/base/tools"}
		override := Config{Template: "custom", OutputDir: "/new/out", ToolsDir: "/new/tools"}
		base.merge(override)
		if base.Template != "custom" {
			t.Errorf("Template = %q, want %q", base.Template, "custom")
		}
		if base.OutputDir != "/new/out" {
			t.Errorf("OutputDir = %q, want %q", base.OutputDir, "/new/out")
		}
		if base.ToolsDir != "/new/tools" {
			t.Errorf("ToolsDir = %q, want %q", base.ToolsDir, "/new/tools")
		}
	})

	t.Run("empty fields do not clobber existing values", func(t *testing.T) {
		base := Config{Template: "default", OutputDir: "/base/out", ToolsDir: "/base/tools"}
		override := Config{Template: "custom"} // OutputDir and ToolsDir intentionally empty
		base.merge(override)
		if base.Template != "custom" {
			t.Errorf("Template = %q, want %q", base.Template, "custom")
		}
		if base.OutputDir != "/base/out" {
			t.Errorf("OutputDir = %q, want %q (should not be clobbered)", base.OutputDir, "/base/out")
		}
		if base.ToolsDir != "/base/tools" {
			t.Errorf("ToolsDir = %q, want %q (should not be clobbered)", base.ToolsDir, "/base/tools")
		}
	})

	t.Run("merge into empty config", func(t *testing.T) {
		base := Config{}
		override := Config{Template: "fancy", OutputDir: "/out"}
		base.merge(override)
		if base.Template != "fancy" || base.OutputDir != "/out" {
			t.Errorf("merge into empty config failed: %+v", base)
		}
	})

	t.Run("merge empty config is no-op", func(t *testing.T) {
		base := Config{Template: "keep", OutputDir: "/keep", ToolsDir: "/keep"}
		base.merge(Config{})
		if base.Template != "keep" || base.OutputDir != "/keep" || base.ToolsDir != "/keep" {
			t.Errorf("merge of empty config should be no-op: %+v", base)
		}
	})

	t.Run("partial overlap merges correctly", func(t *testing.T) {
		base := Config{Template: "base-tmpl", OutputDir: "/base/out"}
		override := Config{OutputDir: "/override/out", ToolsDir: "/override/tools"}
		base.merge(override)
		if base.Template != "base-tmpl" {
			t.Errorf("Template = %q, should be preserved", base.Template)
		}
		if base.OutputDir != "/override/out" {
			t.Errorf("OutputDir = %q, should be overridden", base.OutputDir)
		}
		if base.ToolsDir != "/override/tools" {
			t.Errorf("ToolsDir = %q, should be set from override", base.ToolsDir)
		}
	})
}

func TestReadConfigFile(t *testing.T) {
	t.Run("valid config with all fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		content := `{"template":"custom","output_dir":"/out","tools_dir":"/tools"}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := readConfigFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Template != "custom" || cfg.OutputDir != "/out" || cfg.ToolsDir != "/tools" {
			t.Errorf("config not parsed correctly: %+v", cfg)
		}
	})

	t.Run("partial config leaves other fields empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := os.WriteFile(path, []byte(`{"tools_dir":"/tools"}`), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := readConfigFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ToolsDir != "/tools" {
			t.Errorf("ToolsDir = %q, want %q", cfg.ToolsDir, "/tools")
		}
		if cfg.Template != "" || cfg.OutputDir != "" {
			t.Errorf("unset fields should be empty: %+v", cfg)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := readConfigFile("/nonexistent/path/config.json")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := os.WriteFile(path, []byte(`{not valid json}`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := readConfigFile(path)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("empty JSON object returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := readConfigFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Template != "" || cfg.OutputDir != "" || cfg.ToolsDir != "" {
			t.Errorf("empty JSON should yield empty config: %+v", cfg)
		}
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		content := `{"template":"ok","unknown_field":"ignored","another":123}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := readConfigFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Template != "ok" {
			t.Errorf("Template = %q, want %q", cfg.Template, "ok")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("project config is loaded from cwd", func(t *testing.T) {
		dir := t.TempDir()
		cfgDir := filepath.Join(dir, ".claude")
		if err := os.MkdirAll(cfgDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{"template":"project-tmpl","output_dir":"/project/out"}`
		if err := os.WriteFile(filepath.Join(cfgDir, "transcript-saver.json"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg := LoadConfig(dir)
		if cfg.Template != "project-tmpl" {
			t.Errorf("Template = %q, want %q", cfg.Template, "project-tmpl")
		}
		if cfg.OutputDir != "/project/out" {
			t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "/project/out")
		}
	})

	t.Run("no config files returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		cfg := LoadConfig(dir)
		if cfg.Template != "" || cfg.OutputDir != "" || cfg.ToolsDir != "" {
			t.Errorf("expected empty config with no files, got: %+v", cfg)
		}
	})

	t.Run("empty cwd does not panic", func(t *testing.T) {
		// Should gracefully handle empty cwd (skips project config)
		cfg := LoadConfig("")
		_ = cfg // Just verify no panic
	})
}
