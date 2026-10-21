// Package sim contains the Tide Keeper game rules. It has no rendering or
// input dependencies so it can be tested headlessly.
package sim

import (
	"math"
	"math/rand"
	"strconv"
)

const (
	Width  = 960
	Height = 640

	KnownTime     = 5.0   // seconds a lit rock stays known to every boat
	IslandRadius  = 40.0  // boats dock when they reach the island
	BeamPull      = 110.0 // boats this close to the beam turn toward it
	SecondLampR   = 75.0
	HornRadius    = 170.0
	HornCooldown  = 20.0
	HornSlowTime  = 5.0
	BaseOil       = 100.0
	MaxCondition  = 100.0
	RepairCost    = 10
	RepairAmount  = 35.0
	HeartsPerNite = 3
)

var (
	// The lighthouse stands on an island in the middle of the bay and is
	// also the harbor every boat sails to.
	LighthousePos = Vec{Width / 2, Height / 2}
)

type Phase int

const (
	PhaseNight Phase = iota
	PhaseDay
	PhaseWon
	PhaseLost
)

type Rock struct {
	Pos, Vel Vec
	R        float64
	Known    float64 // seconds left while it is known to all boats
	Buoy     bool    // permanently marked
	Log      bool    // drifting
	Lit      bool    // inside a light this frame
}

func (r *Rock) IsKnown() bool { return r.Buoy || r.Known > 0 }

type Boat struct {
	Kind    BoatKind
	Pos     Vec
	Heading float64
	Slow    float64 // seconds of foghorn slowdown left
	Lit     bool
	Avoid   bool // steering around a rock this frame
	memory  map[*Rock]bool
	phase   float64
}

func (b *Boat) Spec() BoatSpec { return Specs[b.Kind] }

type Fog struct {
	Pos Vec
	R   float64
	VX  float64
}

type EffectKind int

const (
	EffectWreck EffectKind = iota
	EffectDock
	EffectLightning
	EffectHorn
)

type Effect struct {
	Kind EffectKind
	Pos  Vec
	T    float64 // seconds since spawn
	Text string
}

const EffectDuration = 1.6

// Input is everything the player can do in one frame.
type Input struct {
	Aim        Vec  // cursor / touch position
	Focus      bool // held: narrow, long beam
	ToggleLamp bool // night: turn the lamp on/off
	Horn       bool // night: sound the foghorn at Aim
	Click      bool // day: place second lamp at Aim
	Buy        int  // day: 1..8 buys an upgrade, 0 = none
	Repair     bool // day
	Start      bool // day: light the lamp / end screens: restart
}

type World struct {
	Phase      Phase
	Night      int // 1-based
	Coins      int
	Hearts     int
	Oil        float64
	Condition  float64
	Owned      [numUpgrades]int
	BeamAngle  float64
	LampOn     bool
	Focus      bool
	Rocks      []*Rock
	Boats      []*Boat
	Fogs       []Fog
	Effects    []Effect
	Storm      bool
	Wind       Vec
	HornTimer  float64
	LampPos    Vec
	LampPlaced bool
	Docked     int // tonight
	Wrecked    int // tonight
	TotalSaved int
	Note       string
	LoseReason string
	Message    string // short feedback line in the day phase
	Time       float64
	Flash      float64 // lightning flash intensity 0..1

	spawnQueue []BoatKind
	spawnTimer float64
	spawnEvery float64
	lightning  float64
	rng        *rand.Rand
}

// New creates a run that starts on the given night (1 for a normal run).
func New(seed int64, night int) *World {
	w := &World{
		rng:       rand.New(rand.NewSource(seed)),
		Condition: MaxCondition,
		BeamAngle: -math.Pi / 2,
		LampOn:    true,
	}
	w.startNight(max(1, min(night, len(Nights))))
	return w
}

func (w *World) Config() NightConfig { return Nights[w.Night-1] }

func (w *World) OilCap() float64 {
	if w.Owned[BiggerTank] > 0 {
		return BaseOil * 1.5
	}
	return BaseOil
}

// BeamOn reports whether the main lamp is emitting light.
func (w *World) BeamOn() bool { return w.LampOn && (w.Oil > 0 || !w.Config().OilLimit) }

func (w *World) BeamRange() float64 {
	r := 360.0
	if w.Owned[PolishedLens] > 0 {
		r *= 1.25
	}
	if w.Focus {
		r *= 1.6
	}
	return r * (0.5 + 0.5*w.Condition/MaxCondition)
}

func (w *World) BeamHalfAngle() float64 {
	a := 0.30
	if w.Owned[WideReflector] > 0 {
		a *= 1.45
	}
	if w.Focus {
		a *= 0.45
	}
	return a
}

func (w *World) rotateSpeed() float64 {
	if w.Owned[FasterGears] > 0 {
		return 2.4 * 1.3
	}
	return 2.4
}

func (w *World) oilDrain() float64 {
	d := 0.75
	if w.Focus {
		d = 1.9
	}
	if w.Owned[WideReflector] > 0 {
		d *= 1.15
	}
	return d
}

// fogFactor is the multiplier applied to beam reach for a point.
func (w *World) fogFactor(p Vec) float64 {
	f := 1.0
	for _, fg := range w.Fogs {
		if p.Dist(fg.Pos) < fg.R {
			f = math.Min(f, 0.55)
		}
	}
	return f
}

// Lit reports whether a circle of radius r at p is touched by any light.
func (w *World) Lit(p Vec, r float64) bool {
	if w.LampPlaced && w.Owned[SecondLamp] > 0 && p.Dist(w.LampPos) < SecondLampR+r {
		return true
	}
	if !w.BeamOn() {
		return false
	}
	d := p.Sub(LighthousePos)
	dist := d.Len()
	if dist-r > w.BeamRange()*w.fogFactor(p) {
		return false
	}
	if dist <= r {
		return true
	}
	slack := math.Asin(math.Min(1, r/dist))
	return math.Abs(angleDiff(w.BeamAngle, d.Angle())) <= w.BeamHalfAngle()+slack
}

// beamPull reports where a boat at p steers when it sees the main beam: to a
// point on the beam's centre line a little closer to the lighthouse, so
// boats are drawn into the light and then follow it home.
func (w *World) beamPull(p Vec) (Vec, bool) {
	if !w.BeamOn() {
		return Vec{}, false
	}
	axis := FromAngle(w.BeamAngle)
	along := p.Sub(LighthousePos).Dot(axis)
	reach := w.BeamRange() * w.fogFactor(p)
	if along <= IslandRadius || along > reach+BeamPull {
		return Vec{}, false
	}
	// Distance from the edge of the cone, measured perpendicular to the axis.
	lat := math.Abs(axis.Cross(p.Sub(LighthousePos)))
	if lat-along*math.Tan(w.BeamHalfAngle()) > BeamPull {
		return Vec{}, false
	}
	return LighthousePos.Add(axis.Scale(math.Min(along, reach) - 60)), true
}

// Update advances the simulation by dt seconds.
func (w *World) Update(dt float64, in Input) {
	w.Time += dt
	w.Flash = math.Max(0, w.Flash-dt*2.5)
	w.updateEffects(dt)
	switch w.Phase {
	case PhaseNight:
		w.updateNight(dt, in)
	case PhaseDay:
		w.updateDay(in)
	case PhaseWon, PhaseLost:
		if in.Start {
			*w = *New(w.rng.Int63(), 1)
		}
	}
}

func (w *World) updateEffects(dt float64) {
	kept := w.Effects[:0]
	for _, e := range w.Effects {
		e.T += dt
		if e.T < EffectDuration {
			kept = append(kept, e)
		}
	}
	w.Effects = kept
}

func (w *World) addEffect(k EffectKind, p Vec, text string) {
	w.Effects = append(w.Effects, Effect{Kind: k, Pos: p, Text: text})
}

func (w *World) updateNight(dt float64, in Input) {
	cfg := w.Config()

	// Beam aim and lamp.
	target := in.Aim.Sub(LighthousePos).Angle()
	diff := angleDiff(w.BeamAngle, target)
	step := w.rotateSpeed() * dt
	w.BeamAngle += clamp(diff, -step, step)
	w.Focus = in.Focus
	if in.ToggleLamp {
		w.LampOn = !w.LampOn
	}
	if cfg.OilLimit && w.LampOn && w.Oil > 0 {
		w.Oil = math.Max(0, w.Oil-w.oilDrain()*dt)
	}

	// Foghorn.
	w.HornTimer = math.Max(0, w.HornTimer-dt)
	if in.Horn && w.Owned[Foghorn] > 0 && w.HornTimer == 0 {
		w.HornTimer = HornCooldown
		w.addEffect(EffectHorn, in.Aim, "")
		for _, b := range w.Boats {
			if b.Pos.Dist(in.Aim) < HornRadius {
				b.Slow = HornSlowTime
			}
		}
	}

	// Hazards.
	for i := range w.Fogs {
		f := &w.Fogs[i]
		f.Pos.X += f.VX * dt
		if f.Pos.X > Width+f.R {
			f.Pos.X = -f.R
		} else if f.Pos.X < -f.R {
			f.Pos.X = Width + f.R
		}
	}
	for _, r := range w.Rocks {
		if r.Log {
			r.Pos = r.Pos.Add(r.Vel.Scale(dt))
			if r.Pos.X > Width+20 {
				r.Pos.X = -20
			} else if r.Pos.X < -20 {
				r.Pos.X = Width + 20
			}
		}
		r.Known = math.Max(0, r.Known-dt)
		r.Lit = w.Lit(r.Pos, r.R)
		if r.Lit {
			r.Known = KnownTime
		}
	}
	if w.Storm {
		w.updateStorm(dt)
		if w.Phase != PhaseNight {
			return
		}
	}

	// Spawning.
	if len(w.spawnQueue) > 0 {
		w.spawnTimer -= dt
		if w.spawnTimer <= 0 {
			w.spawnTimer = w.spawnEvery
			w.spawnBoat(w.spawnQueue[0])
			w.spawnQueue = w.spawnQueue[1:]
		}
	}

	// Boats.
	kept := w.Boats[:0]
	for _, b := range w.Boats {
		switch w.updateBoat(b, dt) {
		case boatSailing:
			kept = append(kept, b)
		case boatDocked:
			s := b.Spec()
			w.Coins += s.Reward
			w.Docked++
			w.TotalSaved++
			w.addEffect(EffectDock, b.Pos, "+"+strconv.Itoa(s.Reward))
		case boatWrecked:
			w.Wrecked++
			w.Hearts -= b.Spec().RepCost
			w.addEffect(EffectWreck, b.Pos, "")
		}
	}
	w.Boats = kept

	if w.Hearts <= 0 {
		w.Hearts = 0
		w.Phase = PhaseLost
		w.LoseReason = "Too many wrecks tonight. The town has lost faith in its keeper."
		return
	}
	if len(w.spawnQueue) == 0 && len(w.Boats) == 0 {
		w.endNight()
	}
}

func (w *World) updateStorm(dt float64) {
	// Wind slowly veers.
	a := w.Time*0.15 + math.Sin(w.Time*0.4)*0.8
	w.Wind = FromAngle(a).Scale(16)
	w.lightning -= dt
	if w.lightning <= 0 {
		w.lightning = 14 + w.rng.Float64()*8
		dmg := 10.0
		if w.Owned[StormShutters] > 0 {
			dmg /= 2
		}
		w.Condition = math.Max(0, w.Condition-dmg)
		w.Flash = 1
		w.addEffect(EffectLightning, LighthousePos, "-"+strconv.Itoa(int(dmg))+"%")
		if w.Condition <= 0 {
			w.Phase = PhaseLost
			w.LoseReason = "Lightning broke the lamp. The bay goes dark."
		}
	}
}

type boatResult int

const (
	boatSailing boatResult = iota
	boatDocked
	boatWrecked
)

func (w *World) updateBoat(b *Boat, dt float64) boatResult {
	s := b.Spec()
	b.Lit = w.Lit(b.Pos, s.Radius)

	// Captains remember rocks that become known near them. Lost rowboats only
	// notice rocks while they themselves are in the light.
	if b.Kind != Rowboat || b.Lit {
		for _, r := range w.Rocks {
			if r.IsKnown() && b.Pos.Dist(r.Pos) < s.Lookahead*2.5 {
				b.memory[r] = true
			}
		}
	}

	h := FromAngle(b.Heading)
	desired := LighthousePos.Sub(b.Pos).Angle()
	if b.Kind == Rowboat && b.Pos.Dist(LighthousePos) > 120 {
		desired += math.Sin(w.Time*0.7+b.phase) * 1.1
	}
	if target, ok := w.beamPull(b.Pos); ok {
		desired = target.Sub(b.Pos).Angle()
	}

	// Feeler: find the nearest remembered rock in our path and turn away.
	avoid, closest := 0.0, math.Inf(1)
	for r := range b.memory {
		d := r.Pos.Sub(b.Pos)
		along := d.Dot(h)
		if along < -r.R || along > s.Lookahead+r.R {
			continue
		}
		lat := h.Cross(d)
		if math.Abs(lat) < r.R+s.Radius+10 && along < closest {
			closest = along
			if lat > 0 {
				avoid = -1
			} else {
				avoid = 1
			}
		}
	}
	b.Avoid = avoid != 0
	var turn float64
	if b.Avoid {
		turn = avoid * s.Turn
	} else {
		turn = clamp(angleDiff(b.Heading, desired)*3, -s.Turn, s.Turn)
	}
	b.Heading += turn * dt

	speed := s.Speed
	if b.Slow > 0 {
		b.Slow -= dt
		speed *= 0.4
	}
	b.Pos = b.Pos.Add(FromAngle(b.Heading).Scale(speed * dt))
	if w.Storm {
		b.Pos = b.Pos.Add(w.Wind.Scale(dt))
	}
	b.Pos.X = clamp(b.Pos.X, -30, Width+30)
	b.Pos.Y = clamp(b.Pos.Y, -30, Height+30)

	for _, r := range w.Rocks {
		if b.Pos.Dist(r.Pos) < r.R+s.Radius*0.8 {
			return boatWrecked
		}
	}
	if b.Pos.Dist(LighthousePos) < IslandRadius+s.Radius {
		return boatDocked
	}
	return boatSailing
}

func (w *World) spawnBoat(k BoatKind) {
	var p Vec
	switch w.rng.Intn(4) {
	case 0: // left edge
		p = Vec{-20, 40 + w.rng.Float64()*(Height-80)}
	case 1: // right edge
		p = Vec{Width + 20, 40 + w.rng.Float64()*(Height-80)}
	case 2: // top edge
		p = Vec{40 + w.rng.Float64()*(Width-80), -20}
	default: // bottom edge
		p = Vec{40 + w.rng.Float64()*(Width-80), Height + 20}
	}
	w.addBoat(k, p)
}

func (w *World) addBoat(k BoatKind, p Vec) *Boat {
	b := &Boat{
		Kind:    k,
		Pos:     p,
		Heading: LighthousePos.Sub(p).Angle(),
		memory:  map[*Rock]bool{},
		phase:   w.rng.Float64() * 2 * math.Pi,
	}
	w.Boats = append(w.Boats, b)
	return b
}

func (w *World) startNight(n int) {
	w.Night = n
	w.Phase = PhaseNight
	cfg := w.Config()
	w.Hearts = HeartsPerNite
	w.Oil = w.OilCap()
	w.LampOn = true
	w.Docked, w.Wrecked = 0, 0
	w.Boats = nil
	w.Effects = nil
	w.Storm = cfg.Storm
	w.Wind = Vec{}
	w.lightning = 10
	w.HornTimer = 0

	w.Rocks = nil
	for i := 0; i < cfg.Rocks; i++ {
		w.Rocks = append(w.Rocks, &Rock{Pos: w.freeSpot(125), R: 30 + w.rng.Float64()*20})
	}
	for i := 0; i < cfg.Logs; i++ {
		// Logs drift in a band above or below the island.
		y := 60 + w.rng.Float64()*160
		if w.rng.Intn(2) == 0 {
			y = Height - y
		}
		p := Vec{w.rng.Float64() * Width, y}
		v := 10 + w.rng.Float64()*10
		if w.rng.Intn(2) == 0 {
			v = -v
		}
		w.Rocks = append(w.Rocks, &Rock{Pos: p, Vel: Vec{v, 0}, R: 18, Log: true})
	}
	w.placeBuoys()

	w.Fogs = nil
	for i := 0; i < cfg.Fogs; i++ {
		vx := 6 + w.rng.Float64()*8
		if w.rng.Intn(2) == 0 {
			vx = -vx
		}
		w.Fogs = append(w.Fogs, Fog{
			Pos: Vec{w.rng.Float64() * Width, 40 + w.rng.Float64()*(Height-80)},
			R:   80 + w.rng.Float64()*60,
			VX:  vx,
		})
	}

	w.spawnQueue = w.spawnQueue[:0]
	for _, k := range cfg.New {
		w.spawnQueue = append(w.spawnQueue, k, k)
	}
	for len(w.spawnQueue) < cfg.Boats {
		w.spawnQueue = append(w.spawnQueue, cfg.Kinds[w.rng.Intn(len(cfg.Kinds))])
	}
	w.rng.Shuffle(len(w.spawnQueue), func(i, j int) {
		w.spawnQueue[i], w.spawnQueue[j] = w.spawnQueue[j], w.spawnQueue[i]
	})
	w.spawnEvery = math.Max(3, 8-float64(n)*0.5)
	w.spawnTimer = 2
}

// freeSpot picks a rock position away from the harbor, the screen edges and
// other rocks.
func (w *World) freeSpot(minSep float64) Vec {
	var p Vec
	for try := 0; try < 200; try++ {
		p = Vec{90 + w.rng.Float64()*(Width-180), 90 + w.rng.Float64()*(Height-180)}
		if p.Dist(LighthousePos) < 145 {
			continue
		}
		ok := true
		for _, r := range w.Rocks {
			if p.Dist(r.Pos) < minSep {
				ok = false
				break
			}
		}
		if ok {
			return p
		}
	}
	return p
}

// placeBuoys marks the static rocks closest to the harbor.
func (w *World) placeBuoys() {
	for n := 0; n < w.Owned[HarborBuoy]; n++ {
		var best *Rock
		for _, r := range w.Rocks {
			if r.Log || r.Buoy {
				continue
			}
			if best == nil || r.Pos.Dist(LighthousePos) < best.Pos.Dist(LighthousePos) {
				best = r
			}
		}
		if best != nil {
			best.Buoy = true
		}
	}
}

func (w *World) endNight() {
	if w.Night >= len(Nights) {
		w.Phase = PhaseWon
		return
	}
	w.Phase = PhaseDay
	w.Message = ""
	w.Boats = nil
	mood := thanks[w.rng.Intn(len(thanks))]
	if w.Wrecked > 0 {
		mood = complaints[w.rng.Intn(len(complaints))]
	}
	w.Note = mood + "\n" + story[w.Night-1]
}

func (w *World) updateDay(in Input) {
	if in.Buy >= 1 && in.Buy <= int(numUpgrades) {
		w.Message = w.Buy(UpgradeID(in.Buy - 1))
	}
	if in.Repair {
		w.Message = w.Repair()
	}
	if in.Click && w.Owned[SecondLamp] > 0 && in.Aim.Dist(LighthousePos) > IslandRadius+10 {
		w.LampPos = in.Aim
		w.LampPlaced = true
		w.Message = "Second lamp placed."
	}
	if in.Start {
		w.startNight(w.Night + 1)
	}
}

// Buy purchases an upgrade and returns a feedback message.
func (w *World) Buy(id UpgradeID) string {
	u := Upgrades[id]
	switch {
	case w.Owned[id] >= u.Max:
		return u.Name + " already owned."
	case w.Coins < u.Cost:
		return "Not enough coins for " + u.Name + "."
	}
	w.Coins -= u.Cost
	w.Owned[id]++
	if id == BiggerTank {
		w.Oil = w.OilCap()
	}
	return "Bought " + u.Name + "."
}

// Repair fixes storm damage.
func (w *World) Repair() string {
	switch {
	case w.Condition >= MaxCondition:
		return "The lighthouse is in perfect shape."
	case w.Coins < RepairCost:
		return "Not enough coins to repair."
	}
	w.Coins -= RepairCost
	w.Condition = math.Min(MaxCondition, w.Condition+RepairAmount)
	return "Repaired the lighthouse."
}

// Remaining is the number of boats still to arrive or sailing tonight.
func (w *World) Remaining() int { return len(w.spawnQueue) + len(w.Boats) }
