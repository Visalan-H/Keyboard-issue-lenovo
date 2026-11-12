package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Windows virtual-key codes for the keys we care about by name.
// The low-level keyboard hook reports left/right specific codes for
// Shift, Ctrl and Alt, so the generic VK_SHIFT/VK_CONTROL/VK_MENU are
// deliberately not listed.
const (
	vkBack     = 0x08
	vkTab      = 0x09
	vkReturn   = 0x0D
	vkPause    = 0x13
	vkCapital  = 0x14
	vkEscape   = 0x1B
	vkSpace    = 0x20
	vkPrior    = 0x21
	vkNext     = 0x22
	vkEnd      = 0x23
	vkHome     = 0x24
	vkLeft     = 0x25
	vkUp       = 0x26
	vkRight    = 0x27
	vkDown     = 0x28
	vkSnapshot = 0x2C
	vkInsert   = 0x2D
	vkDelete   = 0x2E
	vkLWin     = 0x5B
	vkRWin     = 0x5C
	vkApps     = 0x5D
	vkNumpad0  = 0x60
	vkMultiply = 0x6A
	vkAdd      = 0x6B
	vkSubtract = 0x6D
	vkDecimal  = 0x6E
	vkDivide   = 0x6F
	vkF1       = 0x70
	vkNumLock  = 0x90
	vkScroll   = 0x91
	vkLShift   = 0xA0
	vkRShift   = 0xA1
	vkLControl = 0xA2
	vkRControl = 0xA3
	vkLMenu    = 0xA4
	vkRMenu    = 0xA5
)

// keyNames maps the canonical name of each key to its virtual-key code.
var keyNames = map[string]uint32{
	"Backspace":   vkBack,
	"Tab":         vkTab,
	"Enter":       vkReturn,
	"Pause":       vkPause,
	"CapsLock":    vkCapital,
	"Esc":         vkEscape,
	"Space":       vkSpace,
	"PageUp":      vkPrior,
	"PageDown":    vkNext,
	"End":         vkEnd,
	"Home":        vkHome,
	"Left":        vkLeft,
	"Up":          vkUp,
	"Right":       vkRight,
	"Down":        vkDown,
	"PrintScreen": vkSnapshot,
	"Insert":      vkInsert,
	"Delete":      vkDelete,
	"LWin":        vkLWin,
	"RWin":        vkRWin,
	"Menu":        vkApps,
	"NumMultiply": vkMultiply,
	"NumAdd":      vkAdd,
	"NumSubtract": vkSubtract,
	"NumDecimal":  vkDecimal,
	"NumDivide":   vkDivide,
	"NumLock":     vkNumLock,
	"ScrollLock":  vkScroll,
	"LShift":      vkLShift,
	"RShift":      vkRShift,
	"LCtrl":       vkLControl,
	"RCtrl":       vkRControl,
	"LAlt":        vkLMenu,
	"RAlt":        vkRMenu,
	";":           0xBA,
	"=":           0xBB,
	",":           0xBC,
	"-":           0xBD,
	".":           0xBE,
	"/":           0xBF,
	"`":           0xC0,
	"[":           0xDB,
	"\\":          0xDC,
	"]":           0xDD,
	"'":           0xDE,
}

// keyAliases are extra spellings accepted in config.json.
var keyAliases = map[string]string{
	"escape":      "Esc",
	"return":      "Enter",
	"capslock":    "CapsLock",
	"caps":        "CapsLock",
	"ins":         "Insert",
	"del":         "Delete",
	"pgup":        "PageUp",
	"pgdn":        "PageDown",
	"prtsc":       "PrintScreen",
	"apps":        "Menu",
	"semicolon":   ";",
	"equals":      "=",
	"comma":       ",",
	"minus":       "-",
	"period":      ".",
	"slash":       "/",
	"backquote":   "`",
	"grave":       "`",
	"lbracket":    "[",
	"rbracket":    "]",
	"backslash":   "\\",
	"quote":       "'",
	"lcontrol":    "LCtrl",
	"rcontrol":    "RCtrl",
	"altgr":       "RAlt",
	"spacebar":    "Space",
	"numenter":    "Enter",
	"numdot":      "NumDecimal",
	"numplus":     "NumAdd",
	"numminus":    "NumSubtract",
	"numstar":     "NumMultiply",
	"numslash":    "NumDivide",
	"printscreen": "PrintScreen",
}

var (
	vkToName   = map[uint32]string{}
	lowerNames = map[string]string{}
)

func init() {
	for c := 'A'; c <= 'Z'; c++ {
		keyNames[string(c)] = uint32(c)
	}
	for c := '0'; c <= '9'; c++ {
		keyNames[string(c)] = uint32(c)
		keyNames["Num"+string(c)] = vkNumpad0 + uint32(c-'0')
	}
	for i := 1; i <= 24; i++ {
		keyNames["F"+strconv.Itoa(i)] = vkF1 + uint32(i-1)
	}
	for name, vk := range keyNames {
		vkToName[vk] = name
		lowerNames[strings.ToLower(name)] = name
	}
}

// ParseKey turns a key name ("W", "capslock", "Num5"), a hex code
// ("0x57") or a decimal code ("87") into a virtual-key code.
func ParseKey(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty key name")
	}
	if name, ok := lowerNames[strings.ToLower(s)]; ok {
		return keyNames[name], nil
	}
	if name, ok := keyAliases[strings.ToLower(s)]; ok {
		return keyNames[name], nil
	}
	if len(s) > 1 && s[0] >= '0' && s[0] <= '9' {
		n, err := strconv.ParseUint(s, 0, 32)
		if err == nil && n > 0 && n < 0xFF {
			return uint32(n), nil
		}
		return 0, fmt.Errorf("bad key code %q (must be 1-254, e.g. 0x57)", s)
	}
	return 0, fmt.Errorf("unknown key %q", s)
}

// KeyName returns the canonical name for a virtual-key code, or a hex
// code like "0xE2" for keys without a name.
func KeyName(vk uint32) string {
	if name, ok := vkToName[vk]; ok {
		return name
	}
	return fmt.Sprintf("0x%02X", vk)
}

// KeyInfo is one entry in the list of named keys sent to the settings page.
type KeyInfo struct {
	Name string `json:"name"`
	VK   uint32 `json:"vk"`
}

// AllKeys lists every named key, sorted by virtual-key code.
func AllKeys() []KeyInfo {
	keys := make([]KeyInfo, 0, len(keyNames))
	for name, vk := range keyNames {
		keys = append(keys, KeyInfo{name, vk})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].VK < keys[j].VK })
	return keys
}
