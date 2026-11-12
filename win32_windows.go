package main

// Raw Win32 functions, constants and structs. Only what DeadKeys uses.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	// The hook functions only exist as ...W (Unicode) variants in
	// user32; plain "SetWindowsHookEx" is not exported.
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procMapVirtualKeyW      = user32.NewProc("MapVirtualKeyW")
	procSetTimer            = user32.NewProc("SetTimer")
	procKillTimer           = user32.NewProc("KillTimer")

	procAllocConsole = kernel32.NewProc("AllocConsole")
)

const (
	whKeyboardLL = 13
	hcAction     = 0

	wmQuit       = 0x0012
	wmTimer      = 0x0113
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	llkhfInjected = 0x10

	inputKeyboard        = 1
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002

	mapvkVkToVscEx = 4
)

// kbdllhookstruct is KBDLLHOOKSTRUCT, what lParam points to in a
// low-level keyboard hook.
type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// keybdinput is KEYBDINPUT.
type keybdinput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// input is INPUT with type INPUT_KEYBOARD. In C the second field is a
// union whose largest member (MOUSEINPUT) is 32 bytes, so the struct is
// padded out to 40 bytes on 64-bit Windows. SendInput rejects calls
// where the size is wrong.
type input struct {
	Type uint32
	_    uint32
	Ki   keybdinput
	_    [8]byte
}

// Fail to compile if input is not exactly 40 bytes (64-bit builds only).
var (
	_ [unsafe.Sizeof(input{}) - 40]struct{}
	_ [40 - unsafe.Sizeof(input{})]struct{}
)

// msg is MSG.
type msg struct {
	HWnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       struct{ X, Y int32 }
	LPrivate uint32
}

// getMessage wraps GetMessageW. It returns false on WM_QUIT or error.
func getMessage(m *msg) bool {
	r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(m)), 0, 0, 0)
	return int32(r) > 0
}

func dispatchMessage(m *msg) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(m)))
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(m)))
}

func messageBox(text string, flags uint32) {
	t, _ := windows.UTF16PtrFromString(text)
	c, _ := windows.UTF16PtrFromString("DeadKeys")
	windows.MessageBox(0, t, c, flags)
}

func showError(msg string) { messageBox(msg, windows.MB_OK|windows.MB_ICONERROR) }

func showInfo(msg string) { messageBox(msg, windows.MB_OK|windows.MB_ICONINFORMATION) }
