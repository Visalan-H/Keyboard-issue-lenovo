package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// instanceMutex is held while DeadKeys runs, so a second copy can tell.
const instanceMutex = `Local\DeadKeys-single-instance`

func run(opts options) int {
	// The tray window must stay on one OS thread: the one that created it.
	runtime.LockOSThread()

	if opts.debug {
		if err := openDebugConsole(); err != nil {
			showError("Could not open the debug console: " + err.Error())
		}
	}

	release, err := claimSingleInstance()
	if err != nil {
		// Running deadkeys.exe again opens the settings of the copy
		// that is already running.
		if !signalRunningCopy() {
			showError("DeadKeys is already running.")
		}
		return 0
	}
	defer release()

	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		showError(fmt.Sprintf("Could not load %s:\n\n%v\n\nFix the file or delete it to start from the defaults.", opts.configPath, err))
		return 1
	}
	km, err := Compile(cfg)
	if err != nil {
		showError(err.Error())
		return 1
	}

	engine := NewEngine(km)
	hook := newKeyboardHook(engine, opts.debug)
	if err := hook.Start(); err != nil {
		showError(err.Error())
		return 1
	}
	defer hook.Stop()

	settings := NewSettingsServer(opts.configPath, func(km *Keymap) {
		engine.SetKeymap(km)
		log.Printf("Settings saved: %d layer rules, %d direct remaps", len(km.Layer), len(km.Remap))
	})

	t, err := newTray(engine, debounce(time.Second, func() { go openSettings(settings) }))
	if err != nil {
		showError(err.Error())
		return 1
	}

	log.Printf("DeadKeys %s running with %s", version, opts.configPath)
	log.Printf("%d layer rules on %s, %d direct remaps", len(km.Layer), cfg.LayerKey, len(km.Remap))
	if opts.debug {
		log.Print("Press Ctrl+C or close this window to quit.")
	}

	// Ctrl+C and closing the debug console both end up here.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		t.Close()
	}()

	t.Run()
	log.Print("Quitting")
	return 0
}

// openSettings starts the settings server if needed and opens the page
// in the default browser. It runs on its own goroutine because starting
// a browser can take a while.
func openSettings(s *SettingsServer) {
	url, err := s.URL()
	if err != nil {
		showError("Could not start the settings page: " + err.Error())
		return
	}
	log.Printf("Opening settings: %s", url)
	// Going through explorer.exe opens the browser as the normal user
	// even when DeadKeys runs as admin.
	cmd := exec.Command("explorer.exe", url)
	if err := cmd.Start(); err != nil {
		showError("Could not open the browser: " + err.Error() + "\n\nOpen this address yourself:\n" + url)
		return
	}
	go cmd.Wait() // explorer.exe always exits with code 1; ignore it
}

// debounce wraps f so calls within d of the last run are ignored, e.g.
// the two clicks of a double-click on the tray icon.
func debounce(d time.Duration, f func()) func() {
	var mu sync.Mutex
	var last time.Time
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(last) < d {
			return
		}
		last = time.Now()
		f()
	}
}

// claimSingleInstance fails if another copy of DeadKeys is running. A
// copy running as admin makes CreateMutex fail with access denied for a
// normal copy, which also means "already running".
func claimSingleInstance() (release func(), err error) {
	name, _ := windows.UTF16PtrFromString(instanceMutex)
	h, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return nil, errors.New("already running")
	}
	if err != nil {
		return nil, err
	}
	return func() { windows.CloseHandle(h) }, nil
}

// openDebugConsole gives this GUI-subsystem program a console window
// and points stdout, stderr and the log at it.
func openDebugConsole() error {
	if r, _, err := procAllocConsole.Call(); r == 0 {
		return err
	}
	out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	os.Stdout = out
	os.Stderr = out
	log.SetOutput(out)
	log.SetFlags(log.Ltime)
	return nil
}
