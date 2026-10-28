package config

import "testing"

func TestLoadReadsBrowserWebSocketEndpoint(t *testing.T) {
	t.Setenv("BROWSER_WS_ENDPOINT", "ws://browser:3000/")

	cfg := Load()
	if cfg.BrowserEndpoint != "ws://browser:3000/" {
		t.Errorf("BrowserEndpoint = %q, want %q", cfg.BrowserEndpoint, "ws://browser:3000/")
	}
}
