package main

// The low-level keyboard hook. It stays thin: it turns Windows events
// into Events, asks the Engine what to do, and does it. All decisions
// live in layer.go.

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// rehookInterval is how often the hook is reinstalled. Windows silently
// removes a low-level hook whose callback takes longer than
// LowLevelHooksTimeout (at most 1s on Windows 10 1709+), for example if
// the PC stalls under heavy load, and never tells the app. Reinstalling
// now and then brings it back if that happens.
const rehookInterval = 5 * time.Minute

// deadkeysMarker tags the events we send, for the debug log.
const deadkeysMarker = 0x444B // "DK"

// hookEvent is one handled event, sent to the debug logger.
type hookEvent struct {
	ev  Event
	act Action
}

type keyboardHook struct {
	engine   *Engine
	log      chan hookEvent // nil unless --debug
	handle   uintptr
	threadID uint32
	done     chan struct{}
}

// The hook callback is a plain function, so it finds its state here.
// Only one hook is ever running.
var (
	currentHook  *keyboardHook
	hookCallback = windows.NewCallback(hookProc)
)

func newKeyboardHook(engine *Engine, debug bool) *keyboardHook {
	h := &keyboardHook{engine: engine, done: make(chan struct{})}
	if debug {
		h.log = make(chan hookEvent, 256)
		go h.printLog()
	}
	return h
}

// Start runs the hook on its own OS thread and returns once it is
// installed.
func (h *keyboardHook) Start() error {
	ready := make(chan error, 1)
	go h.loop(ready)
	return <-ready
}

// Stop removes the hook, releases any keys it is holding down, and
// waits for its thread to finish.
func (h *keyboardHook) Stop() {
	procPostThreadMessageW.Call(uintptr(h.threadID), wmQuit, 0, 0)
	<-h.done
}

func (h *keyboardHook) loop(ready chan<- error) {
	defer close(h.done)

	// A low-level hook is called on the thread that installed it, and
	// only while that thread is waiting in GetMessage. So this goroutine
	// must stay on one OS thread and keep pumping messages.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	h.threadID = windows.GetCurrentThreadId()
	currentHook = h
	if err := h.install(); err != nil {
		ready <- err
		return
	}
	timer, _, _ := procSetTimer.Call(0, 0, uintptr(rehookInterval/time.Millisecond), 0)
	ready <- nil

	var m msg
	for getMessage(&m) {
		if m.Message == wmTimer && m.HWnd == 0 && m.WParam == timer {
			h.reinstall()
			continue
		}
		dispatchMessage(&m)
	}

	procKillTimer.Call(0, timer)
	procUnhookWindowsHookEx.Call(h.handle)
	sendKeys(h.engine.ReleaseAll())
}

func (h *keyboardHook) install() error {
	r, _, err := procSetWindowsHookExW.Call(whKeyboardLL, hookCallback, 0, 0)
	if r == 0 {
		return fmt.Errorf("could not install the keyboard hook: %w", err)
	}
	h.handle = r
	return nil
}

// reinstall puts a fresh hook in place before removing the old one, so
// no key is missed in between.
func (h *keyboardHook) reinstall() {
	old := h.handle
	if err := h.install(); err != nil {
		return // keep the old one
	}
	procUnhookWindowsHookEx.Call(old)
}

// hookProc runs for every key event in the whole system, so it must be
// fast: no I/O, no logging, no locks that could block.
func hookProc(nCode, wParam, lParam uintptr) uintptr {
	h := currentHook
	if int32(nCode) == hcAction && h != nil {
		// Reading lParam this way keeps go vet happy; it is a pointer
		// Windows gave us, valid for the duration of this call.
		kb := *(**kbdllhookstruct)(unsafe.Pointer(&lParam))

		var down, isKey bool
		switch wParam {
		case wmKeyDown, wmSysKeyDown:
			down, isKey = true, true
		case wmKeyUp, wmSysKeyUp:
			isKey = true
		}

		if isKey {
			ev := Event{
				VK:       kb.VkCode,
				Down:     down,
				Injected: kb.Flags&llkhfInjected != 0,
				Time:     kb.Time,
			}
			act := h.engine.Handle(ev)
			if act.N > 0 {
				sendKeys(act.Strokes())
			}
			if h.log != nil && kb.DwExtraInfo != deadkeysMarker {
				select {
				case h.log <- hookEvent{ev, act}:
				default: // logger is behind; drop rather than wait
				}
			}
			if act.Swallow {
				return 1
			}
		}
	}
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

// sendKeys sends key presses and releases with SendInput.
func sendKeys(strokes []Stroke) error {
	var buf [8]input
	for len(strokes) > 0 {
		n := min(len(strokes), len(buf))
		for i, s := range strokes[:n] {
			buf[i] = makeInput(s)
		}
		sent, _, err := procSendInput.Call(uintptr(n), uintptr(unsafe.Pointer(&buf[0])), unsafe.Sizeof(buf[0]))
		if int(sent) != n {
			return fmt.Errorf("SendInput sent %d of %d events: %w", sent, n, err)
		}
		strokes = strokes[n:]
	}
	return nil
}

func makeInput(s Stroke) input {
	// Fill in the scan code too: some apps (games, remote desktop)
	// read that instead of the virtual-key code.
	scan, _, _ := procMapVirtualKeyW.Call(uintptr(s.VK), mapvkVkToVscEx)
	var flags uint32
	if scan&0xFF00 == 0xE000 || isExtendedKey(s.VK) {
		flags |= keyeventfExtendedKey
	}
	if !s.Down {
		flags |= keyeventfKeyUp
	}
	return input{
		Type: inputKeyboard,
		Ki: keybdinput{
			WVk:         uint16(s.VK),
			WScan:       uint16(scan & 0xFF),
			DwFlags:     flags,
			DwExtraInfo: deadkeysMarker,
		},
	}
}

// isExtendedKey lists keys that need KEYEVENTF_EXTENDEDKEY, in case
// MapVirtualKey doesn't say so.
func isExtendedKey(vk uint32) bool {
	switch vk {
	case vkInsert, vkDelete, vkHome, vkEnd, vkPrior, vkNext,
		vkLeft, vkUp, vkRight, vkDown,
		vkRControl, vkRMenu, vkLWin, vkRWin, vkApps,
		vkDivide, vkNumLock, vkSnapshot:
		return true
	}
	return false
}

func (h *keyboardHook) printLog() {
	for e := range h.log {
		dir := "up  "
		if e.ev.Down {
			dir = "down"
		}
		line := fmt.Sprintf("%-12s %s", KeyName(e.ev.VK), dir)
		switch {
		case e.ev.Injected:
			line += "  (sent by software, passed)"
		case e.act.N > 0:
			line += "  -> " + fmt.Sprint(StrokeNames(e.act.Strokes()))
		case e.act.Swallow:
			line += "  (swallowed)"
		}
		fmt.Println(line)
	}
}
