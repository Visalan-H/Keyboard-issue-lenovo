package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows"
)

// instanceMutex is held while DeadKeys runs, so a second copy can tell.
const instanceMutex = `Local\DeadKeys-single-instance`

func run(opts options) int {
	if opts.debug {
		if err := openDebugConsole(); err != nil {
			showError("Could not open the debug console: " + err.Error())
		}
	}

	release, err := claimSingleInstance()
	if err != nil {
		showError("DeadKeys is already running.")
		return 1
	}
	defer release()

	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		showError(fmt.Sprintf("Could not load %s:\n\n%v", opts.configPath, err))
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

	log.Printf("DeadKeys %s running with %s", version, opts.configPath)
	log.Printf("%d layer rules on %s, %d direct remaps", len(km.Layer), cfg.LayerKey, len(km.Remap))
	if opts.debug {
		log.Print("Press Ctrl+C or close this window to quit.")
	}

	// Ctrl+C and closing the console window both end up here.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Print("Quitting")
	return 0
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
