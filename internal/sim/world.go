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

// Rock is an oval obstacle: R is the half-length along Angle and B the
// half-width (B == 0 means a round rock of radius R).
type Rock struct {
	Pos, Vel Vec
	R, B     float64
	Angle    float64
	Known    float64 // seconds left while it is known to all boats
	Buoy     bool    // permanently marked
	Log      bool    // drifting
	Lit      bool    // inside a light this frame
}

func (r *Rock) IsKnown() bool { return r.Buoy || r.Known > 0 }

// Circle is one of the overlapping circles that approximate a rock's shape.
type Circle struct {
	Pos Vec
	R   float64
}

// Parts approximates the oval with a row of circles along its long axis, which
// keeps lighting, collision and steering simple circle checks.
func (r *Rock) Parts() []Circle {
	if r.B <= 0 || r.B >= r.R {
		return []Circle{{r.Pos, r.R}}
	}
	axis := FromAngle(r.Angle)
	n := int(math.Ceil((r.R-r.B)/(r.B*0.5))) + 1
	parts := make([]Circle, n)
	for i := range parts {
		// Spread centres from one end to the other; circles near the ends
		// shrink a little so the outline stays oval rather than a capsule.
		t := -1 + 2*float64(i)/float64(n-1)
		parts[i] = Circle{r.Pos.Add(axis.Scale(t * (r.R - r.B))), r.B * math.Sqrt(1-0.3*t*t)}
	}
	return parts
}

func (r *Rock) hits(p Vec, radius float64) bool {
	for _, c := range r.Parts() {
		if p.Dist(c.Pos) < c.R+radius {
			return true
		}
	}
	return false
}

type Boat struct {
	Kind    BoatKind
	Pos     Vec
	Heading float64
	Slow    float64 // seconds of foghorn slowdown left
	Lit     bool
	Avoid   bool              // steering around a rock this frame
	Debug   bool              // spawned by hand; never pays coins or costs hearts
	Target  float64           // heading the captain is steering for
	memory  map[*Rock]float64 // rock -> time the captain noticed it
	turnVel float64           // current rate of turn, rad/s
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
	Spawn      int  // night, debug: spawns boat kind Spawn-1 at Aim, 0 = none
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
		r.Lit = false
		for _, c := range r.Parts() {
			if w.Lit(c.Pos, c.R) {
				r.Lit = true
				break
			}
		}
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
	if in.Spawn > 0 && in.Spawn <= len(Specs) {
		b := w.addBoat(BoatKind(in.Spawn-1), in.Aim)
		b.Debug = true
	}
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
			if b.Debug {
				w.addEffect(EffectDock, b.Pos, "docked")
				continue
			}
			s := b.Spec()
			w.Coins += s.Reward
			w.Docked++
			w.TotalSaved++
			w.addEffect(EffectDock, b.Pos, "+"+strconv.Itoa(s.Reward))
		case boatWrecked:
			w.addEffect(EffectWreck, b.Pos, "")
			if b.Debug {
				continue
			}
			w.Wrecked++
			w.Hearts -= b.Spec().RepCost
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
			if _, ok := b.memory[r]; !ok && r.IsKnown() && b.Pos.Dist(r.Pos) < s.Lookahead*1.5 {
				b.memory[r] = w.Time
			}
		}
	}

	desired := LighthousePos.Sub(b.Pos).Angle()
	if b.Kind == Rowboat && b.Pos.Dist(LighthousePos) > 120 {
		desired += math.Sin(w.Time*0.7+b.phase) * 1.1
	}
	if target, ok := w.beamPull(b.Pos); ok {
		desired = target.Sub(b.Pos).Angle()
	}

	// Captains look for the heading closest to the one they want whose path
	// is clear of every rock they know about. They need a moment to react to a
	// newly lit rock and the hull turns gradually, so a rock lit too late is
	// hit anyway.
	var known []Circle
	for r, seen := range b.memory {
		if w.Time-seen >= s.React {
			known = append(known, r.Parts()...)
		}
	}
	reach := math.Min(s.Lookahead, b.Pos.Dist(LighthousePos)-IslandRadius)
	var wind Vec
	if w.Storm {
		wind = w.Wind
	}
	clear := func(a, margin float64) bool {
		return b.courseClear(a, reach, s.Radius+margin, wind, known)
	}
	heading := desired
	b.Avoid = false
	if len(known) > 0 && !clear(desired, 16) {
		b.Avoid = true
		// Pick the clear heading that strays least from the goal while
		// staying close to where the bow already points, so the captain
		// commits to one way round instead of flipping sides mid-turn. A wide
		// berth is preferred over a narrow one.
		best := math.Inf(1)
		for k := -12; k <= 12; k++ {
			a := desired + float64(k)*0.25
			cost := math.Abs(float64(k)*0.25) + 1.5*math.Abs(angleDiff(b.Heading, a))
			if !clear(a, 16) {
				if !clear(a, 4) {
					continue
				}
				cost += 1
			}
			if cost < best {
				best, heading = cost, a
			}
		}
	}
	b.Target = heading
	turn := clamp(angleDiff(b.Heading, heading)*3, -s.Turn, s.Turn)
	// The rudder needs about half a second to reach full lock.
	maxStep := s.Turn * 2 * dt
	b.turnVel += clamp(turn-b.turnVel, -maxStep, maxStep)
	b.Heading += b.turnVel * dt

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
		if r.hits(b.Pos, s.Radius*0.8) {
			return boatWrecked
		}
	}
	if b.Pos.Dist(LighthousePos) < IslandRadius+s.Radius {
		return boatDocked
	}
	return boatSailing
}

// courseClear plays the boat's own steering forward for a couple of seconds
// as it turns toward heading a, then follows a straight line, and reports
// whether that course stays clear of the rocks. Simulating the turn matters:
// a slow rudder sweeps the bow through water a straight line would miss.
func (b *Boat) courseClear(a, reach, radius float64, wind Vec, rocks []Circle) bool {
	s := b.Spec()
	const step = 0.15
	horizon := math.Min(2.4, math.Max(1.2, reach/s.Speed))
	p, head, tv := b.Pos, b.Heading, b.turnVel
	start := make([]float64, len(rocks))
	for i, c := range rocks {
		start[i] = p.Dist(c.Pos)
	}
	for t := 0.0; t < horizon; t += step {
		turn := clamp(angleDiff(head, a)*3, -s.Turn, s.Turn)
		tv += clamp(turn-tv, -s.Turn*2*step, s.Turn*2*step)
		head += tv * step
		p = p.Add(FromAngle(head).Scale(s.Speed * step)).Add(wind.Scale(step))
		for i, c := range rocks {
			// Inside the margin already: only moving closer counts.
			if d := p.Dist(c.Pos); d < c.R+radius && (start[i] >= c.R+radius || d < start[i]) {
				return false
			}
		}
	}
	course := FromAngle(head).Scale(s.Speed).Add(wind).Angle()
	return pathClear(p, course, reach-s.Speed*horizon, radius, rocks)
}

// pathClear reports whether a hull of the given radius (including any safety
// margin) can sail reach pixels from p along angle a without touching any of
// the circles.
func pathClear(p Vec, a, reach, radius float64, rocks []Circle) bool {
	dir := FromAngle(a)
	for _, c := range rocks {
		d := c.Pos.Sub(p)
		clear := c.R + radius
		if d.Len() < clear {
			// Already inside the safety margin: any heading away is fine.
			if d.Dot(dir) > 0 {
				return false
			}
			continue
		}
		along := clamp(d.Dot(dir), 0, math.Max(reach, 0))
		if p.Add(dir.Scale(along)).Dist(c.Pos) < clear {
			return false
		}
	}
	return true
}

// spawnBoat brings a boat in from a random edge, through open water so it has
// room to react to the first rock on its way.
func (w *World) spawnBoat(k BoatKind) {
	var p Vec
	for try := 0; try < 30; try++ {
		p = w.edgePoint()
		if w.openWater(p, 200) {
			break
		}
	}
	w.addBoat(k, p)
}

func (w *World) openWater(p Vec, dist float64) bool {
	for _, r := range w.Rocks {
		if r.Log {
			continue
		}
		for _, c := range r.Parts() {
			if p.Dist(c.Pos) < c.R+dist {
				return false
			}
		}
	}
	return true
}

func (w *World) edgePoint() Vec {
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
	return p
}

func (w *World) addBoat(k BoatKind, p Vec) *Boat {
	b := &Boat{
		Kind:    k,
		Pos:     p,
		Heading: LighthousePos.Sub(p).Angle(),
		memory:  map[*Rock]float64{},
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
		if r := w.placeRock(); r != nil {
			w.Rocks = append(w.Rocks, r)
		}
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
		w.Rocks = append(w.Rocks, &Rock{Pos: p, Vel: Vec{v, 0}, R: 30, B: 9, Log: true})
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

// placeRock makes an oval rock lying across the route to the lighthouse, away
// from the island, the screen edges and other rocks. Every rock keeps a gap
// wide enough for boats to pass. It returns nil if no spot is free.
func (w *World) placeRock() *Rock {
	const gap = 55
	for try := 0; try < 300; try++ {
		p := Vec{90 + w.rng.Float64()*(Width-180), 90 + w.rng.Float64()*(Height-180)}
		r := &Rock{
			Pos:   p,
			R:     45 + w.rng.Float64()*30,
			B:     18 + w.rng.Float64()*8,
			Angle: LighthousePos.Sub(p).Angle() + math.Pi/2 + (w.rng.Float64()-0.5)*0.4,
		}
		if w.rockFits(r, gap) {
			return r
		}
	}
	return nil
}

func (w *World) rockFits(r *Rock, gap float64) bool {
	for _, c := range r.Parts() {
		if c.Pos.Dist(LighthousePos) < c.R+IslandRadius+80 ||
			c.Pos.X-c.R < 80 || c.Pos.X+c.R > Width-80 || c.Pos.Y-c.R < 70 || c.Pos.Y+c.R > Height-70 {
			return false
		}
		for _, o := range w.Rocks {
			for _, oc := range o.Parts() {
				if c.Pos.Dist(oc.Pos) < c.R+oc.R+gap {
					return false
				}
			}
		}
	}
	return true
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
