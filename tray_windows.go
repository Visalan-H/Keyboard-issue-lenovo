package main

// The tray icon and its menu. A hidden window receives the icon's mouse
// messages; it lives on the main thread, separate from the hook thread,
// so a slow menu can never delay key events.

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procShellNotifyIconW            = shell32.NewProc("Shell_NotifyIconW")
	procRegisterClassExW            = user32.NewProc("RegisterClassExW")
	procCreateWindowExW             = user32.NewProc("CreateWindowExW")
	procDefWindowProcW              = user32.NewProc("DefWindowProcW")
	procDestroyWindow               = user32.NewProc("DestroyWindow")
	procPostQuitMessage             = user32.NewProc("PostQuitMessage")
	procPostMessageW                = user32.NewProc("PostMessageW")
	procFindWindowW                 = user32.NewProc("FindWindowW")
	procRegisterWindowMessageW      = user32.NewProc("RegisterWindowMessageW")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
	procCreatePopupMenu             = user32.NewProc("CreatePopupMenu")
	procAppendMenuW                 = user32.NewProc("AppendMenuW")
	procSetMenuDefaultItem          = user32.NewProc("SetMenuDefaultItem")
	procTrackPopupMenu              = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                 = user32.NewProc("DestroyMenu")
	procSetForegroundWindow         = user32.NewProc("SetForegroundWindow")
	procGetCursorPos                = user32.NewProc("GetCursorPos")
	procGetSystemMetrics            = user32.NewProc("GetSystemMetrics")
	procCreateIconFromResourceEx    = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon                 = user32.NewProc("DestroyIcon")
	procGetModuleHandleW            = kernel32.NewProc("GetModuleHandleW")
)

const (
	trayClassName = "DeadKeysTrayWindow"

	wmNull         = 0x0000
	wmDestroy      = 0x0002
	wmClose        = 0x0010
	wmApp          = 0x8000
	wmTrayCallback = wmApp + 1
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmContextMenu  = 0x007B

	nimAdd     = 0
	nimModify  = 1
	nimDelete  = 2
	nifMessage = 0x1
	nifIcon    = 0x2
	nifTip     = 0x4

	mfString    = 0x0000
	mfChecked   = 0x0008
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100

	msgfltAllow  = 1
	smCxSmIcon   = 49
	lrDefaultClr = 0

	menuSettings = 1
	menuPause    = 2
	menuQuit     = 3
)

// openSettingsMessage is posted to the running copy by a second copy of
// deadkeys.exe, asking it to open the settings page.
const openSettingsMessage = "DeadKeys.OpenSettings"

type wndclassex struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     uintptr
}

// Fail to compile if the structs don't match their 64-bit C sizes.
var (
	_ [unsafe.Sizeof(wndclassex{}) - 80]struct{}
	_ [80 - unsafe.Sizeof(wndclassex{})]struct{}
	_ [unsafe.Sizeof(notifyIconData{}) - 976]struct{}
	_ [976 - unsafe.Sizeof(notifyIconData{})]struct{}
)

type point struct{ X, Y int32 }

type tray struct {
	hwnd           uintptr
	icons          [2]uintptr // running, paused
	engine         *Engine
	openSettings   func()
	msgOpen        uint32
	msgTaskbarMade uint32
}

// The window procedure is a plain function, so it finds its state here.
var (
	currentTray *tray
	trayWndProc = windows.NewCallback(trayProc)
)

func registerMessage(name string) uint32 {
	p, _ := windows.UTF16PtrFromString(name)
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(p)))
	return uint32(r)
}

// signalRunningCopy asks an already running DeadKeys to open its
// settings. It reports whether one was found.
func signalRunningCopy() bool {
	class, _ := windows.UTF16PtrFromString(trayClassName)
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(class)), 0)
	if hwnd == 0 {
		return false
	}
	procPostMessageW.Call(hwnd, uintptr(registerMessage(openSettingsMessage)), 0, 0)
	return true
}

// newTray creates the hidden window and the tray icon. It must be called
// on the thread that will run the message loop.
func newTray(engine *Engine, openSettings func()) (*tray, error) {
	t := &tray{
		engine:         engine,
		openSettings:   openSettings,
		msgOpen:        registerMessage(openSettingsMessage),
		msgTaskbarMade: registerMessage("TaskbarCreated"),
	}
	currentTray = t

	hinst, _, _ := procGetModuleHandleW.Call(0)
	class, _ := windows.UTF16PtrFromString(trayClassName)
	wc := wndclassex{
		LpfnWndProc:   trayWndProc,
		HInstance:     hinst,
		LpszClassName: class,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return nil, fmt.Errorf("RegisterClassEx: %w", err)
	}
	title, _ := windows.UTF16PtrFromString("DeadKeys")
	hwnd, _, err := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0, 0, 0, hinst, 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("CreateWindowEx: %w", err)
	}
	t.hwnd = hwnd

	// When DeadKeys runs as admin, Windows blocks messages from normal
	// programs unless we allow them: one from a second deadkeys.exe, and
	// one from Explorer when the taskbar restarts.
	procChangeWindowMessageFilterEx.Call(hwnd, uintptr(t.msgOpen), msgfltAllow, 0)
	procChangeWindowMessageFilterEx.Call(hwnd, uintptr(t.msgTaskbarMade), msgfltAllow, 0)

	size, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	if size == 0 {
		size = 16
	}
	for i, paused := range []bool{false, true} {
		img := iconImage(int(size), paused)
		t.icons[i], _, _ = procCreateIconFromResourceEx.Call(
			uintptr(unsafe.Pointer(&img[0])), uintptr(len(img)), 1, 0x00030000, size, size, lrDefaultClr)
	}

	if !t.notify(nimAdd) {
		return nil, fmt.Errorf("could not add the tray icon")
	}
	return t, nil
}

// notify adds, updates or removes the tray icon.
func (t *tray) notify(action uintptr) bool {
	nid := notifyIconData{
		HWnd:             t.hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTrayCallback,
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	tip := "DeadKeys"
	nid.HIcon = t.icons[0]
	if t.engine.Paused() {
		tip = "DeadKeys (paused)"
		nid.HIcon = t.icons[1]
	}
	copy(nid.SzTip[:len(nid.SzTip)-1], windows.StringToUTF16(tip))
	r, _, _ := procShellNotifyIconW.Call(action, uintptr(unsafe.Pointer(&nid)))
	return r != 0
}

// Run pumps window messages until Quit.
func (t *tray) Run() {
	var m msg
	for getMessage(&m) {
		dispatchMessage(&m)
	}
}

// Close asks the tray to quit; safe to call from any goroutine.
func (t *tray) Close() {
	procPostMessageW.Call(t.hwnd, wmClose, 0, 0)
}

func (t *tray) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	defer procDestroyMenu.Call(menu)
	add := func(flags uintptr, id uintptr, text string) {
		p, _ := windows.UTF16PtrFromString(text)
		procAppendMenuW.Call(menu, flags, id, uintptr(unsafe.Pointer(p)))
	}
	add(mfString, menuSettings, "Settings...")
	pauseFlags := uintptr(mfString)
	if t.engine.Paused() {
		pauseFlags |= mfChecked
	}
	add(pauseFlags, menuPause, "Pause")
	add(mfSeparator, 0, "")
	add(mfString, menuQuit, "Quit DeadKeys")
	procSetMenuDefaultItem.Call(menu, menuSettings, 0)

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Without this the menu doesn't close when you click elsewhere.
	procSetForegroundWindow.Call(t.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmNoNotify,
		uintptr(pt.X), uintptr(pt.Y), 0, t.hwnd, 0)
	procPostMessageW.Call(t.hwnd, wmNull, 0, 0)

	switch cmd {
	case menuSettings:
		t.openSettings()
	case menuPause:
		t.engine.SetPaused(!t.engine.Paused())
		t.notify(nimModify)
	case menuQuit:
		procDestroyWindow.Call(t.hwnd)
	}
}

func trayProc(hwnd, message, wParam, lParam uintptr) uintptr {
	t := currentTray
	if t != nil {
		switch uint32(message) {
		case wmTrayCallback:
			switch uint32(lParam) & 0xFFFF {
			case wmLButtonUp:
				t.openSettings()
			case wmRButtonUp, wmContextMenu:
				t.showMenu()
			}
			return 0
		case t.msgOpen:
			t.openSettings()
			return 0
		case t.msgTaskbarMade:
			// Explorer restarted and forgot every tray icon.
			t.notify(nimAdd)
			return 0
		case wmDestroy:
			t.notify(nimDelete)
			for _, ic := range t.icons {
				procDestroyIcon.Call(ic)
			}
			procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}
