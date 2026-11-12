package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	opts, err := parseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.debug || filepath.Base(opts.configPath) != "config.json" {
		t.Errorf("defaults: %+v", opts)
	}

	opts, err = parseFlags([]string{"--debug", "--config", `C:\keys\mine.json`})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.debug || opts.configPath != `C:\keys\mine.json` {
		t.Errorf("got %+v", opts)
	}

	if _, err := parseFlags([]string{"--nope"}); err == nil || !strings.Contains(err.Error(), "-debug") {
		t.Errorf("unknown flag should fail and show usage, got %v", err)
	}
	if _, err := parseFlags([]string{"extra"}); err == nil {
		t.Error("stray argument should fail")
	}
}
