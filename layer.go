package main

import "sync/atomic"

// Event is one key press or release seen by the keyboard hook.
type Event struct {
	VK       uint32
	Down     bool   // false = key released
	Injected bool   // sent by software (SendInput), not the keyboard
	Time     uint32 // milliseconds, from the event; may wrap around
}

// Stroke is one key event the engine wants sent.
type Stroke struct {
	VK   uint32
	Down bool
}

// Action tells the hook what to do with an Event.
type Action struct {
	// Swallow drops the original event so no app sees it.
	Swallow bool
	// Send lists keys to send, in order; only the first N are used.
	Send [2]Stroke
	N    int
}

func (a *Action) send(vk uint32, down bool) {
	a.Send[a.N] = Stroke{vk, down}
	a.N++
}

// Strokes returns the keys to send.
func (a *Action) Strokes() []Stroke { return a.Send[:a.N] }

// Engine is the layer state machine. Handle must only be called from
// one goroutine (the hook thread). SetKeymap and SetPaused are safe to
// call from anywhere.
type Engine struct {
	keymap atomic.Pointer[Keymap]
	paused atomic.Bool

	// State below is only touched by Handle and ReleaseAll.

	// layerHeld is the layer key currently held down, or 0. It is kept
	// separately from the keymap so a config change while the key is
	// held still releases it properly.
	layerHeld   uint32
	layerDownAt uint32
	layerUsed   bool // another key was pressed while the layer key was held

	// active maps each physical key we turned into another key to the
	// key we sent for it, so its release sends the matching release
	// even if the layer key was let go first.
	active map[uint32]uint32

	// passed holds keys whose press we let through untouched. Their
	// auto-repeats and release must pass too, even if the layer key goes
	// down in between; otherwise the app would see a press with no release.
	passed map[uint32]bool
}

// NewEngine returns an engine using km.
func NewEngine(km *Keymap) *Engine {
	e := &Engine{active: map[uint32]uint32{}, passed: map[uint32]bool{}}
	e.keymap.Store(km)
	return e
}

// SetKeymap swaps in a new keymap. Keys already held keep sending what
// they were sending until they are released.
func (e *Engine) SetKeymap(km *Keymap) { e.keymap.Store(km) }

// SetPaused turns remapping off or back on.
func (e *Engine) SetPaused(p bool) { e.paused.Store(p) }

// Paused reports whether remapping is off.
func (e *Engine) Paused() bool { return e.paused.Load() }

// Handle decides what to do with one key event.
func (e *Engine) Handle(ev Event) Action {
	var act Action

	// Our own SendInput events (and any other software's) pass through.
	// Without this, the keys we send would be remapped again.
	if ev.Injected {
		return act
	}

	km := e.keymap.Load()

	if !ev.Down {
		return e.handleUp(ev, km)
	}

	// Auto-repeat of a key we already turned into another key.
	if target, ok := e.active[ev.VK]; ok {
		act.Swallow = true
		act.send(target, true)
		return act
	}

	if e.layerHeld != 0 && ev.VK == e.layerHeld {
		// Auto-repeat of the layer key: keep swallowing, don't restart
		// the tap timer.
		act.Swallow = true
		return act
	}

	if e.passed[ev.VK] {
		// Auto-repeat of a key we let through (often a held Shift).
		// It doesn't count as using the layer.
		return act
	}

	if e.paused.Load() {
		e.passed[ev.VK] = true
		return act
	}

	if ev.VK == km.LayerVK {
		e.layerHeld = ev.VK
		e.layerDownAt = ev.Time
		e.layerUsed = false
		act.Swallow = true
		return act
	}

	if e.layerHeld != 0 {
		e.layerUsed = true
		if target, ok := km.Layer[ev.VK]; ok {
			return e.press(ev.VK, target)
		}
	}

	if target, ok := km.Remap[ev.VK]; ok {
		return e.press(ev.VK, target)
	}

	e.passed[ev.VK] = true
	return act
}

func (e *Engine) handleUp(ev Event, km *Keymap) Action {
	var act Action

	if target, ok := e.active[ev.VK]; ok {
		delete(e.active, ev.VK)
		act.Swallow = true
		act.send(target, false)
		return act
	}

	if e.layerHeld != 0 && ev.VK == e.layerHeld {
		heldFor := ev.Time - e.layerDownAt // unsigned, so wrap-around is fine
		tap := !e.layerUsed && km.TapTimeout > 0 && heldFor <= km.TapTimeout
		e.layerHeld = 0
		act.Swallow = true
		if tap {
			act.send(ev.VK, true)
			act.send(ev.VK, false)
		}
		return act
	}

	delete(e.passed, ev.VK)
	return act
}

func (e *Engine) press(from, to uint32) Action {
	var act Action
	e.active[from] = to
	act.Swallow = true
	act.send(to, true)
	return act
}

// ReleaseAll forgets all held keys and returns the releases to send,
// so nothing is left stuck down when the app quits.
func (e *Engine) ReleaseAll() []Stroke {
	var out []Stroke
	for from, to := range e.active {
		out = append(out, Stroke{to, false})
		delete(e.active, from)
	}
	clear(e.passed)
	e.layerHeld = 0
	return out
}
