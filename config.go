package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ConfigVersion is bumped when config.json changes in a way old
// versions of the app can't read.
const ConfigVersion = 1

// Config is the contents of config.json.
type Config struct {
	Version int `json:"version"`

	// LayerKey is held to turn on the layer rules. Tapping it on its own
	// still sends the key (so a CapsLock tap toggles caps lock).
	LayerKey string `json:"layerKey"`

	// TapTimeoutMs is how long the layer key may be held and still count
	// as a tap. 0 turns tapping off entirely.
	TapTimeoutMs int `json:"tapTimeoutMs"`

	// Layer rules apply while LayerKey is held.
	Layer []Rule `json:"layer"`

	// Remap rules always apply.
	Remap []Rule `json:"remap"`

	// DeadKeys lists the keys that don't work on this keyboard. The app
	// only uses it to refuse rules that start from a dead key and to
	// highlight them in settings; a future key test will fill it in.
	DeadKeys []string `json:"deadKeys"`
}

// Rule makes the From key send the To key.
type Rule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// DefaultConfig is used when there is no config.json.
func DefaultConfig() Config {
	return Config{
		Version:      ConfigVersion,
		LayerKey:     "CapsLock",
		TapTimeoutMs: 300,
		Layer: []Rule{
			{"Q", "W"},
			{"A", "S"},
			{"Z", "X"},
			{"I", "O"},
			{"1", "2"},
			{"8", "9"},
		},
		Remap:    []Rule{},
		DeadKeys: []string{"2", "W", "S", "X", "O", "9"},
	}
}

// ParseConfig reads config.json. Missing scalar fields keep their
// defaults; missing rule lists are empty.
func ParseConfig(data []byte) (Config, error) {
	def := DefaultConfig()
	cfg := Config{
		Version:      def.Version,
		LayerKey:     def.LayerKey,
		TapTimeoutMs: def.TapTimeoutMs,
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config.json is not valid: %w", err)
	}
	if cfg.Version > ConfigVersion {
		return Config{}, fmt.Errorf("config.json is version %d, but this build only understands up to %d", cfg.Version, ConfigVersion)
	}
	cfg.Version = ConfigVersion
	if cfg.Layer == nil {
		cfg.Layer = []Rule{}
	}
	if cfg.Remap == nil {
		cfg.Remap = []Rule{}
	}
	if cfg.DeadKeys == nil {
		cfg.DeadKeys = []string{}
	}
	if _, err := Compile(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Keymap is a checked Config turned into virtual-key codes, ready for
// the engine to use.
type Keymap struct {
	LayerVK    uint32
	TapTimeout uint32 // milliseconds, 0 = tapping off
	Layer      map[uint32]uint32
	Remap      map[uint32]uint32
}

// Compile checks a Config and converts it into a Keymap. The returned
// error lists every problem found, one per line.
func Compile(cfg Config) (*Keymap, error) {
	var errs []string
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	km := &Keymap{
		Layer: map[uint32]uint32{},
		Remap: map[uint32]uint32{},
	}

	layerVK, err := ParseKey(cfg.LayerKey)
	if err != nil {
		fail("layerKey: %v", err)
	}
	km.LayerVK = layerVK

	if cfg.TapTimeoutMs < 0 || cfg.TapTimeoutMs > 5000 {
		fail("tapTimeoutMs must be between 0 and 5000, got %d", cfg.TapTimeoutMs)
	} else {
		km.TapTimeout = uint32(cfg.TapTimeoutMs)
	}

	dead := map[uint32]bool{}
	for _, name := range cfg.DeadKeys {
		vk, err := ParseKey(name)
		if err != nil {
			fail("deadKeys: %v", err)
			continue
		}
		dead[vk] = true
	}
	if dead[layerVK] {
		fail("layerKey %s is marked dead, pick a key that works", KeyName(layerVK))
	}

	compile := func(section string, rules []Rule, into map[uint32]uint32) {
		for i, r := range rules {
			where := fmt.Sprintf("%s rule %d (%s -> %s)", section, i+1, r.From, r.To)
			from, err := ParseKey(r.From)
			if err != nil {
				fail("%s: from: %v", where, err)
				continue
			}
			to, err := ParseKey(r.To)
			if err != nil {
				fail("%s: to: %v", where, err)
				continue
			}
			switch {
			case from == layerVK:
				fail("%s: %s is the layer key and can't be remapped", where, KeyName(from))
			case from == to:
				fail("%s: a key can't be mapped to itself", where)
			case dead[from]:
				fail("%s: %s is marked dead, so it can't be the key you press", where, KeyName(from))
			default:
				if _, dup := into[from]; dup {
					fail("%s: %s already has a %s rule", where, KeyName(from), section)
					continue
				}
				into[from] = to
			}
		}
	}
	compile("layer", cfg.Layer, km.Layer)
	compile("remap", cfg.Remap, km.Remap)

	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return km, nil
}

// LoadConfig reads the config file at path. If it doesn't exist, the
// default config is written there and returned.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		cfg := DefaultConfig()
		if err := SaveConfig(path, cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	return ParseConfig(data)
}

// SaveConfig writes cfg to path, replacing the old file in one step so a
// crash never leaves a half-written config.
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
