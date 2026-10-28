package sim

import (
	"math"
	"testing"
)

const dt = 1.0 / 60

// emptyNight returns a world on night n with no rocks, fog or queued boats.
func emptyNight(n int) *World {
	w := New(1, 1)
	w.startNight(n)
	w.Rocks, w.Fogs, w.spawnQueue = nil, nil, nil
	w.Storm = false
	return w
}

// aimAway points the beam straight down, away from the test boats
// (the boats in these tests sail in from the top).
var aimAway = Input{Aim: Vec{LighthousePos.X, Height + 100}}

func run(w *World, in Input, seconds float64) {
	for t := 0.0; t < seconds && w.Phase == PhaseNight; t += dt {
		w.Update(dt, in)
	}
}

func TestBeamLightsPointInsideCone(t *testing.T) {
	w := emptyNight(1)
	w.BeamAngle = -math.Pi / 2 // straight up
	if !w.Lit(Vec{LighthousePos.X, LighthousePos.Y - 200}, 5) {
		t.Fatal("point straight ahead should be lit")
	}
	if w.Lit(Vec{LighthousePos.X + 300, LighthousePos.Y - 20}, 5) {
		t.Fatal("point far to the side should not be lit")
	}
	if w.Lit(Vec{LighthousePos.X - 450, LighthousePos.Y}, 5) {
		t.Fatal("point beyond range should not be lit")
	}
	w.BeamAngle = math.Pi // straight left
	w.Focus = true
	if !w.Lit(Vec{LighthousePos.X - 450, LighthousePos.Y}, 5) {
		t.Fatal("focused beam should reach further")
	}
}

func TestFogShortensBeam(t *testing.T) {
	w := emptyNight(1)
	w.BeamAngle = -math.Pi / 2
	p := Vec{LighthousePos.X, LighthousePos.Y - 250}
	w.Fogs = []Fog{{Pos: p, R: 50}}
	if w.Lit(p, 1) {
		t.Fatal("fog should hide a point at 250px")
	}
}

func TestBoatDocksAndPays(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing} // keep the night going
	w.spawnTimer = 1e9
	w.addBoat(Fishing, Vec{LighthousePos.X, 20})
	run(w, aimAway, 30)
	if w.Docked != 1 || w.Coins != Specs[Fishing].Reward {
		t.Fatalf("docked=%d coins=%d", w.Docked, w.Coins)
	}
}

func TestBoatWrecksOnUnlitRock(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	w.Rocks = []*Rock{{Pos: Vec{LighthousePos.X, 170}, R: 30}}
	w.addBoat(Fishing, Vec{LighthousePos.X, -20})
	w.LampOn = false
	run(w, aimAway, 30)
	if w.Wrecked != 1 || w.Hearts != HeartsPerNite-1 {
		t.Fatalf("wrecked=%d hearts=%d", w.Wrecked, w.Hearts)
	}
}

// Boats in these tests sail in from the left edge toward a rock 220 px
// before the lighthouse.
var sideRock = Vec{LighthousePos.X - 280, LighthousePos.Y}

func TestBoatAvoidsLitRock(t *testing.T) {
	for _, k := range []BoatKind{Fishing, Merchant, Speedboat, Ferry} {
		w := emptyNight(1)
		w.spawnQueue = []BoatKind{Fishing}
		w.spawnTimer = 1e9
		w.Rocks = []*Rock{{Pos: sideRock, R: 30}}
		w.BeamAngle = math.Pi
		w.addBoat(k, Vec{-20, LighthousePos.Y})
		run(w, Input{Aim: sideRock}, 40)
		if w.Wrecked != 0 || w.Docked != 1 {
			t.Errorf("%s: wrecked=%d docked=%d", Specs[k].Name, w.Wrecked, w.Docked)
		}
	}
}

func TestRockLitTooLateIsHit(t *testing.T) {
	for _, k := range []BoatKind{Fishing, Merchant, Speedboat, Ferry} {
		w := emptyNight(1)
		w.spawnQueue = []BoatKind{Fishing}
		w.spawnTimer = 1e9
		rock := &Rock{Pos: sideRock, R: 30}
		w.Rocks = []*Rock{rock}
		b := w.addBoat(k, Vec{-20, LighthousePos.Y})
		w.LampOn = false
		// Sail in the dark until the bow is a few hull lengths from the rock...
		for b.Pos.Dist(rock.Pos) > rock.R+b.Spec().Radius+25 {
			w.Update(dt, aimAway)
		}
		// ...then light it.
		w.LampOn = true
		w.BeamAngle = math.Pi
		run(w, Input{Aim: rock.Pos}, 10)
		if w.Wrecked != 1 {
			t.Errorf("%s: lit too late but wrecked=%d", Specs[k].Name, w.Wrecked)
		}
	}
}

func TestBoatTurnsTowardBeam(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	// A boat coming from the top-left corner, beam pointing straight up.
	b := w.addBoat(Fishing, Vec{LighthousePos.X - 200, 40})
	start := b.Pos.X
	run(w, Input{Aim: Vec{LighthousePos.X, 0}}, 3)
	pulled := b.Pos.X - start
	w = emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	b = w.addBoat(Fishing, Vec{LighthousePos.X - 200, 40})
	w.LampOn = false
	run(w, Input{Aim: Vec{LighthousePos.X, 0}}, 3)
	free := b.Pos.X - start
	if pulled <= free+5 {
		t.Fatalf("beam should pull the boat sideways: pulled=%.1f free=%.1f", pulled, free)
	}
}

func TestBoatsSpawnOnEverySide(t *testing.T) {
	w := emptyNight(1)
	var top, bottom, left, right bool
	for i := 0; i < 200; i++ {
		w.spawnBoat(Fishing)
	}
	for _, b := range w.Boats {
		top = top || b.Pos.Y < 0
		bottom = bottom || b.Pos.Y > Height
		left = left || b.Pos.X < 0
		right = right || b.Pos.X > Width
	}
	if !top || !bottom || !left || !right {
		t.Fatalf("top=%v bottom=%v left=%v right=%v", top, bottom, left, right)
	}
}

func TestThreeWrecksEndsRun(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	for i := 0; i < 3; i++ {
		x := 200 + float64(i)*250
		w.Rocks = append(w.Rocks, &Rock{Pos: Vec{x, 100}, R: 30})
		w.addBoat(Fishing, Vec{x, 99})
	}
	w.Update(dt, aimAway)
	if w.Phase != PhaseLost {
		t.Fatalf("phase=%v hearts=%d", w.Phase, w.Hearts)
	}
}

func TestOilDrainsOnlyWhenLimited(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	run(w, aimAway, 5)
	if w.Oil != w.OilCap() {
		t.Fatalf("night 1 should not drain oil, got %v", w.Oil)
	}
	w = emptyNight(2)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	run(w, aimAway, 5)
	normal := w.OilCap() - w.Oil
	w = emptyNight(2)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	run(w, Input{Aim: aimAway.Aim, Focus: true}, 5)
	focused := w.OilCap() - w.Oil
	if normal <= 0 || focused <= normal {
		t.Fatalf("normal=%v focused=%v", normal, focused)
	}
	w.Oil = 0
	if w.BeamOn() {
		t.Fatal("empty lamp should be dark")
	}
}

func TestNightEndsIntoDayAndShop(t *testing.T) {
	w := emptyNight(1)
	w.Update(dt, aimAway)
	if w.Phase != PhaseDay {
		t.Fatalf("phase=%v", w.Phase)
	}
	w.Coins = 25
	w.Update(dt, Input{Buy: int(FasterGears) + 1})
	if w.Owned[FasterGears] != 1 || w.Coins != 0 {
		t.Fatalf("owned=%v coins=%d", w.Owned, w.Coins)
	}
	w.Update(dt, Input{Buy: int(PolishedLens) + 1})
	if w.Owned[PolishedLens] != 0 {
		t.Fatal("should not afford the lens")
	}
	w.Update(dt, Input{Start: true})
	if w.Phase != PhaseNight || w.Night != 2 || w.Hearts != HeartsPerNite {
		t.Fatalf("phase=%v night=%d hearts=%d", w.Phase, w.Night, w.Hearts)
	}
}

func TestBuoyMarksRock(t *testing.T) {
	w := New(3, 1)
	w.Owned[HarborBuoy] = 1
	w.startNight(2)
	n := 0
	for _, r := range w.Rocks {
		if r.Buoy {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("buoys=%d", n)
	}
}

func TestWinAfterLastNight(t *testing.T) {
	w := emptyNight(len(Nights))
	w.Update(dt, aimAway)
	if w.Phase != PhaseWon {
		t.Fatalf("phase=%v", w.Phase)
	}
}

// TestNightsArePlayable plays every night with a simple bot that sweeps the
// beam toward the boat closest to the harbor, checking the run never stalls.
func TestNightsArePlayable(t *testing.T) {
	for n := 1; n <= len(Nights); n++ {
		w := New(int64(n), 1)
		w.Owned[BiggerTank] = 1
		w.Owned[StormShutters] = 1
		w.startNight(n)
		for i := 0; i < 60*60*6 && w.Phase == PhaseNight; i++ {
			aim := botAim(w)
			w.Update(dt, Input{Aim: aim, Focus: aim.Dist(LighthousePos) > 300})
		}
		if w.Phase == PhaseNight {
			t.Errorf("night %d did not finish (remaining %d)", n, w.Remaining())
		}
		// The early, tutorial-like nights must be survivable by a basic bot.
		if n <= 5 && w.Phase == PhaseLost {
			t.Errorf("night %d lost by the bot: %s", n, w.LoseReason)
		}
		t.Logf("night %d: phase=%v docked=%d wrecked=%d time=%.0fs", n, w.Phase, w.Docked, w.Wrecked, w.Time)
	}
}

// botAim lights the unknown rock that the most boats are about to reach.
func botAim(w *World) Vec {
	aim, best := LighthousePos.Sub(Vec{0, 200}), math.Inf(1)
	for _, b := range w.Boats {
		h := FromAngle(LighthousePos.Sub(b.Pos).Angle())
		for _, r := range w.Rocks {
			if _, ok := b.memory[r]; ok {
				continue
			}
			d := r.Pos.Sub(b.Pos)
			along := d.Dot(h)
			if along < 0 || math.Abs(h.Cross(d)) > r.R+b.Spec().Radius+30 {
				continue
			}
			if t := along / b.Spec().Speed; t < best {
				best, aim = t, r.Pos
			}
		}
	}
	return aim
}

func TestDebugSpawnAtCursor(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	rock := &Rock{Pos: Vec{200, 200}, R: 60, B: 20}
	w.Rocks = []*Rock{rock}
	w.Update(dt, Input{Aim: Vec{200, 200}, Spawn: int(Merchant) + 1})
	w.Update(dt, aimAway)
	if w.Wrecked != 0 || w.Hearts != HeartsPerNite {
		t.Fatalf("debug wreck should not count: wrecked=%d hearts=%d", w.Wrecked, w.Hearts)
	}
	w.Update(dt, Input{Aim: Vec{100, 500}, Spawn: int(Speedboat) + 1})
	if len(w.Boats) != 1 || w.Boats[0].Kind != Speedboat || !w.Boats[0].Debug {
		t.Fatalf("boats=%+v", w.Boats)
	}
}

func TestOvalRocksLieAcrossTheRoute(t *testing.T) {
	w := New(7, 1)
	w.startNight(5)
	for _, r := range w.Rocks {
		if r.Log {
			continue
		}
		if r.R <= r.B {
			t.Fatalf("rock is not oval: R=%v B=%v", r.R, r.B)
		}
		// The long axis should be roughly perpendicular to the lighthouse.
		radial := LighthousePos.Sub(r.Pos).Angle()
		if d := math.Abs(math.Cos(r.Angle - radial)); d > 0.3 {
			t.Errorf("rock at %v is not across the route (cos=%.2f)", r.Pos, d)
		}
	}
}
