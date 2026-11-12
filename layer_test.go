package main

import (
	"reflect"
	"testing"
)

// harness feeds events to an engine and checks what comes out.
type harness struct {
	t   *testing.T
	e   *Engine
	now uint32
}

func newHarness(t *testing.T, cfg Config) *harness {
	t.Helper()
	km, err := Compile(cfg)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return &harness{t: t, e: NewEngine(km), now: 1000}
}

func (h *harness) wait(ms uint32) { h.now += ms }

// key sends one event and checks the result. want lists the strokes the
// engine should send, written as "+W" (press) or "-W" (release).
func (h *harness) key(name string, down bool, swallow bool, want ...string) {
	h.t.Helper()
	vk, err := ParseKey(name)
	if err != nil {
		h.t.Fatal(err)
	}
	act := h.e.Handle(Event{VK: vk, Down: down, Time: h.now})
	h.check(name, down, act, swallow, want)
}

func (h *harness) check(name string, down bool, act Action, swallow bool, want []string) {
	h.t.Helper()
	dir := "up"
	if down {
		dir = "down"
	}
	if act.Swallow != swallow {
		h.t.Errorf("%s %s: swallow = %v, want %v", name, dir, act.Swallow, swallow)
	}
	got := StrokeNames(act.Strokes())
	if want == nil {
		want = []string{}
	}
	if !reflect.DeepEqual(got, want) {
		h.t.Errorf("%s %s: sent %v, want %v", name, dir, got, want)
	}
}

const (
	pass    = false
	swallow = true
)

func TestUnmappedKeysPassThrough(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("E", true, pass)
	h.key("E", false, pass)
	h.key("Q", true, pass) // Q is only mapped inside the layer
	h.key("Q", false, pass)
}

func TestLayerMapsKeys(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	h.key("Q", false, swallow, "-W")
	h.key("A", true, swallow, "+S")
	h.key("A", false, swallow, "-S")
	h.key("1", true, swallow, "+2")
	h.key("1", false, swallow, "-2")
	h.key("8", true, swallow, "+9")
	h.key("8", false, swallow, "-9")
	h.key("E", true, pass) // unmapped keys still work inside the layer
	h.key("E", false, pass)
	h.key("CapsLock", false, swallow)
	h.key("Q", true, pass) // layer is off again
	h.key("Q", false, pass)
}

func TestShiftPassesThroughSoItApplies(t *testing.T) {
	// Shift + CapsLock + 1 must send only 2; the real Shift turns it into @.
	h := newHarness(t, DefaultConfig())
	h.key("LShift", true, pass)
	h.key("CapsLock", true, swallow)
	h.key("1", true, swallow, "+2")
	h.key("1", false, swallow, "-2")
	h.key("CapsLock", false, swallow)
	h.key("LShift", false, pass)
}

func TestShiftAutoRepeatDoesNotBlockTap(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("LShift", true, pass)
	h.key("CapsLock", true, swallow)
	h.key("LShift", true, pass) // auto-repeat of the held Shift
	h.wait(50)
	h.key("CapsLock", false, swallow, "+CapsLock", "-CapsLock")
	h.key("LShift", false, pass)
}

func TestAutoRepeat(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("CapsLock", true, swallow) // CapsLock auto-repeat
	h.key("Q", true, swallow, "+W")
	h.key("Q", true, swallow, "+W")
	h.key("Q", true, swallow, "+W")
	h.key("Q", false, swallow, "-W")
	h.key("CapsLock", false, swallow)
}

func TestReleasingLayerKeyFirstDoesNotStickKeys(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	h.key("CapsLock", false, swallow)
	h.key("Q", true, swallow, "+W") // still repeating as W
	h.key("Q", false, swallow, "-W")
	h.key("Q", true, pass)
	h.key("Q", false, pass)
}

func TestKeyHeldBeforeLayerIsLeftAlone(t *testing.T) {
	// Q is pressed (and passed through) before CapsLock goes down. Its
	// repeats and release must pass too, or the app would see Q stuck.
	h := newHarness(t, DefaultConfig())
	h.key("Q", true, pass)
	h.key("CapsLock", true, swallow)
	h.key("Q", true, pass)
	h.key("Q", false, pass)
	h.key("Q", true, swallow, "+W") // a fresh press is mapped
	h.key("Q", false, swallow, "-W")
	h.key("CapsLock", false, swallow)
}

func TestCapsLockTap(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.wait(120)
	h.key("CapsLock", false, swallow, "+CapsLock", "-CapsLock")
}

func TestCapsLockHoldTooLongIsNotATap(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.wait(301)
	h.key("CapsLock", false, swallow)
}

func TestCapsLockRepeatDoesNotRestartTapTimer(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.wait(250)
	h.key("CapsLock", true, swallow)
	h.wait(250)
	h.key("CapsLock", false, swallow)
}

func TestCapsLockWithKeyIsNotATap(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("E", true, pass)
	h.key("E", false, pass)
	h.key("CapsLock", false, swallow)
}

func TestTapTimeoutZeroTurnsTapOff(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TapTimeoutMs = 0
	h := newHarness(t, cfg)
	h.key("CapsLock", true, swallow)
	h.key("CapsLock", false, swallow)
}

func TestTapTimerSurvivesClockWrap(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.now = 0xFFFFFFFF - 50
	h.key("CapsLock", true, swallow)
	h.wait(100) // wraps past zero
	h.key("CapsLock", false, swallow, "+CapsLock", "-CapsLock")
}

func TestDirectRemap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Remap = []Rule{{"Insert", "S"}, {"Q", "O"}}
	h := newHarness(t, cfg)
	h.key("Insert", true, swallow, "+S")
	h.key("Insert", true, swallow, "+S")
	h.key("Insert", false, swallow, "-S")

	// The layer wins over a direct remap while the layer key is held.
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	h.key("Q", false, swallow, "-W")
	// Direct remaps still work inside the layer.
	h.key("Insert", true, swallow, "+S")
	h.key("Insert", false, swallow, "-S")
	h.key("CapsLock", false, swallow)

	h.key("Q", true, swallow, "+O")
	h.key("Q", false, swallow, "-O")
}

func TestInjectedEventsPassThrough(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	for _, down := range []bool{true, false} {
		act := h.e.Handle(Event{VK: 'Q', Down: down, Injected: true, Time: h.now})
		h.check("injected Q", down, act, pass, nil)
		act = h.e.Handle(Event{VK: vkCapital, Down: down, Injected: true, Time: h.now})
		h.check("injected CapsLock", down, act, pass, nil)
	}
	// Injected keys do not count as using the layer, so this is still a tap.
	h.key("CapsLock", false, swallow, "+CapsLock", "-CapsLock")
}

func TestPause(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	h.e.SetPaused(true)
	h.key("Q", false, swallow, "-W") // held keys still release cleanly
	h.key("CapsLock", false, swallow)

	h.key("CapsLock", true, pass)
	h.key("Q", true, pass)
	h.key("Q", false, pass)
	h.key("CapsLock", false, pass)

	h.e.SetPaused(false)
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	h.key("Q", false, swallow, "-W")
	h.key("CapsLock", false, swallow)
}

func TestKeymapSwapWhileHeld(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")

	cfg := DefaultConfig()
	cfg.LayerKey = "RAlt"
	cfg.Layer = []Rule{{"Q", "X"}}
	km, err := Compile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h.e.SetKeymap(km)

	h.key("Q", false, swallow, "-W") // releases what it pressed
	h.key("CapsLock", false, swallow)
	h.key("CapsLock", true, pass) // no longer the layer key
	h.key("CapsLock", false, pass)
	h.key("RAlt", true, swallow)
	h.key("Q", true, swallow, "+X")
	h.key("Q", false, swallow, "-X")
	h.key("RAlt", false, swallow)
}

func TestReleaseAll(t *testing.T) {
	h := newHarness(t, DefaultConfig())
	h.key("CapsLock", true, swallow)
	h.key("Q", true, swallow, "+W")
	got := StrokeNames(h.e.ReleaseAll())
	if !reflect.DeepEqual(got, []string{"-W"}) {
		t.Errorf("ReleaseAll = %v, want [-W]", got)
	}
	if len(h.e.ReleaseAll()) != 0 {
		t.Error("second ReleaseAll should send nothing")
	}
	h.key("Q", false, pass)
}
