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

// aimAway points the beam straight down, away from the sea.
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
	if w.Lit(Vec{LighthousePos.X, LighthousePos.Y - 500}, 5) {
		t.Fatal("point beyond range should not be lit")
	}
	w.Focus = true
	if !w.Lit(Vec{LighthousePos.X, LighthousePos.Y - 500}, 5) {
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
	w.addBoat(Fishing, Vec{HarborPos.X, 100})
	run(w, aimAway, 30)
	if w.Docked != 1 || w.Coins != Specs[Fishing].Reward {
		t.Fatalf("docked=%d coins=%d", w.Docked, w.Coins)
	}
}

func TestBoatWrecksOnUnlitRock(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	w.Rocks = []*Rock{{Pos: Vec{HarborPos.X, 250}, R: 18}}
	w.addBoat(Fishing, Vec{HarborPos.X, 60})
	w.LampOn = false
	run(w, aimAway, 30)
	if w.Wrecked != 1 || w.Hearts != HeartsPerNite-1 {
		t.Fatalf("wrecked=%d hearts=%d", w.Wrecked, w.Hearts)
	}
}

func TestBoatAvoidsLitRock(t *testing.T) {
	for _, k := range []BoatKind{Fishing, Merchant, Speedboat, Ferry} {
		w := emptyNight(1)
		w.spawnQueue = []BoatKind{Fishing}
		w.spawnTimer = 1e9
		w.Rocks = []*Rock{{Pos: Vec{HarborPos.X, 250}, R: 18}}
		w.addBoat(k, Vec{HarborPos.X, 40})
		run(w, Input{Aim: Vec{HarborPos.X, 250}, Focus: true}, 40)
		if w.Wrecked != 0 || w.Docked != 1 {
			t.Errorf("%s: wrecked=%d docked=%d", Specs[k].Name, w.Wrecked, w.Docked)
		}
	}
}

func TestThreeWrecksEndsRun(t *testing.T) {
	w := emptyNight(1)
	w.spawnQueue = []BoatKind{Fishing}
	w.spawnTimer = 1e9
	for i := 0; i < 3; i++ {
		x := 200 + float64(i)*250
		w.Rocks = append(w.Rocks, &Rock{Pos: Vec{x, 200}, R: 18})
		w.addBoat(Fishing, Vec{x, 199})
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
	aim, best := HarborPos.Sub(Vec{0, 200}), math.Inf(1)
	for _, b := range w.Boats {
		h := FromAngle(HarborPos.Sub(b.Pos).Angle())
		for _, r := range w.Rocks {
			if b.memory[r] {
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
