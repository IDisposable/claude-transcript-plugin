package main

import (
	"testing"
)

func TestResolve(t *testing.T) {
	t.Run("flag takes highest priority", func(t *testing.T) {
		t.Setenv("RESOLVE_TEST_FLAG", "from-env")
		got := resolve("from-flag", "RESOLVE_TEST_FLAG", "from-config", "fallback")
		if got != "from-flag" {
			t.Errorf("got %q, want %q", got, "from-flag")
		}
	})

	t.Run("env var used when flag is empty", func(t *testing.T) {
		t.Setenv("RESOLVE_TEST_ENV", "from-env")
		got := resolve("", "RESOLVE_TEST_ENV", "from-config", "fallback")
		if got != "from-env" {
			t.Errorf("got %q, want %q", got, "from-env")
		}
	})

	t.Run("config used when flag and env are empty", func(t *testing.T) {
		// Use a key unlikely to exist in the environment
		got := resolve("", "RESOLVE_TEST_UNSET_"+t.Name(), "from-config", "fallback")
		if got != "from-config" {
			t.Errorf("got %q, want %q", got, "from-config")
		}
	})

	t.Run("fallback used when all others are empty", func(t *testing.T) {
		got := resolve("", "RESOLVE_TEST_UNSET_"+t.Name(), "", "fallback")
		if got != "fallback" {
			t.Errorf("got %q, want %q", got, "fallback")
		}
	})

	t.Run("all empty returns empty", func(t *testing.T) {
		got := resolve("", "RESOLVE_TEST_UNSET_"+t.Name(), "", "")
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("env var set to empty is treated as absent", func(t *testing.T) {
		t.Setenv("RESOLVE_TEST_EMPTY", "")
		got := resolve("", "RESOLVE_TEST_EMPTY", "from-config", "fallback")
		if got != "from-config" {
			t.Errorf("got %q, want %q — empty env var should fall through", got, "from-config")
		}
	})
}
