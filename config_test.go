package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseKey(t *testing.T) {
	cases := map[string]uint32{
		"W":         'W',
		"w":         'W',
		"2":         '2',
		"capslock":  vkCapital,
		"Caps":      vkCapital,
		"Num5":      vkNumpad0 + 5,
		"F12":       vkF1 + 11,
		"0x57":      'W',
		"87":        'W',
		".":         0xBE,
		"period":    0xBE,
		" Insert ":  vkInsert,
		"AltGr":     vkRMenu,
		"PrtSc":     vkSnapshot,
		"backslash": 0xDC,
	}
	for in, want := range cases {
		got, err := ParseKey(in)
		if err != nil || got != want {
			t.Errorf("ParseKey(%q) = %#x, %v; want %#x", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "Hyper", "0x100", "0x0", "00", "999"} {
		if vk, err := ParseKey(bad); err == nil {
			t.Errorf("ParseKey(%q) = %#x, want error", bad, vk)
		}
	}
}

func TestKeyNamesRoundTrip(t *testing.T) {
	for _, k := range AllKeys() {
		vk, err := ParseKey(k.Name)
		if err != nil || vk != k.VK {
			t.Errorf("%s: ParseKey = %#x, %v; want %#x", k.Name, vk, err, k.VK)
		}
		if KeyName(k.VK) != k.Name {
			t.Errorf("KeyName(%#x) = %q, want %q", k.VK, KeyName(k.VK), k.Name)
		}
	}
	if KeyName(0xE2) != "0xE2" {
		t.Errorf("unnamed key: got %q", KeyName(0xE2))
	}
}

func TestDefaultConfigCompiles(t *testing.T) {
	km, err := Compile(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint32]uint32{'Q': 'W', 'A': 'S', 'Z': 'X', 'I': 'O', '1': '2', '8': '9'}
	if !reflect.DeepEqual(km.Layer, want) {
		t.Errorf("layer = %v, want %v", km.Layer, want)
	}
	if km.LayerVK != vkCapital || km.TapTimeout != 300 {
		t.Errorf("layer key %#x, tap %d", km.LayerVK, km.TapTimeout)
	}
}

func TestRepoConfigMatchesDefault(t *testing.T) {
	data, err := os.ReadFile("config.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Errorf("config.json differs from DefaultConfig():\n%+v\n%+v", cfg, DefaultConfig())
	}
}

func TestParseConfigDefaultsMissingFields(t *testing.T) {
	cfg, err := ParseConfig([]byte(`{"layer": [{"from": "q", "to": "0x57"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LayerKey != "CapsLock" || cfg.TapTimeoutMs != 300 || cfg.Version != ConfigVersion {
		t.Errorf("defaults not applied: %+v", cfg)
	}
	if cfg.Remap == nil || cfg.DeadKeys == nil {
		t.Error("missing lists should be empty, not null")
	}
}

func TestParseConfigErrors(t *testing.T) {
	cases := map[string]string{
		`{"layer": [{"from": "Q", "to": "Hyper"}]}`:                       "unknown key",
		`{"layer": [{"from": "Q", "to": "W"}, {"from": "q", "to": "X"}]}`: "already has a layer rule",
		`{"layer": [{"from": "CapsLock", "to": "W"}]}`:                    "is the layer key",
		`{"remap": [{"from": "W", "to": "W"}]}`:                           "mapped to itself",
		`{"deadKeys": ["W"], "remap": [{"from": "W", "to": "Q"}]}`:        "marked dead",
		`{"deadKeys": ["CapsLock"]}`:                                      "layerKey CapsLock is marked dead",
		`{"tapTimeoutMs": -1}`:                                            "tapTimeoutMs",
		`{"version": 99}`:                                                 "version 99",
		`{"layers": []}`:                                                  "unknown field",
		`{`:                                                               "not valid",
	}
	for in, want := range cases {
		_, err := ParseConfig([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseConfig(%s) error = %v, want it to mention %q", in, err, want)
		}
	}
}

func TestCompileReportsAllErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Layer = []Rule{{"Q", "Nope"}, {"Nah", "W"}}
	_, err := Compile(cfg)
	if err == nil || strings.Count(err.Error(), "\n") != 1 {
		t.Errorf("want two errors on two lines, got %v", err)
	}
}

func TestLoadConfigWritesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Error("expected default config")
	}
	again, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again, cfg) {
		t.Error("saved config did not load back the same")
	}
}

func TestSaveConfigReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := DefaultConfig()
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Remap = []Rule{{"Insert", "S"}}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Remap, cfg.Remap) {
		t.Errorf("remap = %v", got.Remap)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}
