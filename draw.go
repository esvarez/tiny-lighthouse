package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/esvarez/tiny-lighthouse/internal/sim"
)

var (
	colSeaNight = color.RGBA{10, 22, 48, 255}
	colSeaDay   = color.RGBA{120, 170, 205, 255}
	colSand     = color.RGBA{222, 205, 160, 255}
	colRockDark = color.RGBA{16, 20, 30, 255}
	colRockLit  = color.RGBA{90, 86, 80, 255}
	colBeam     = color.RGBA{255, 205, 95, 255}
	colLantern  = color.RGBA{255, 200, 90, 255}
	colText     = color.RGBA{235, 230, 215, 255}
	colDim      = color.RGBA{160, 160, 170, 255}
	colWarn     = color.RGBA{255, 130, 90, 255}
	colGood     = color.RGBA{140, 220, 150, 255}
	colPanel    = color.RGBA{8, 14, 30, 220}
)

var (
	whiteImage = ebiten.NewImage(3, 3)
	whitePixel = whiteImage.SubImage(whiteImage.Bounds().Inset(1)).(*ebiten.Image)
	fontSource *text.GoTextFaceSource
	faceSmall  *text.GoTextFace
	faceMid    *text.GoTextFace
	faceBig    *text.GoTextFace
)

func init() {
	whiteImage.Fill(color.White)
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	fontSource = src
	faceSmall = &text.GoTextFace{Source: src, Size: 14}
	faceMid = &text.GoTextFace{Source: src, Size: 18}
	faceBig = &text.GoTextFace{Source: src, Size: 34}
}

func (g *Game) Draw(screen *ebiten.Image) {
	w := g.world
	night := w.Phase != sim.PhaseDay
	if night {
		screen.Fill(colSeaNight)
	} else {
		screen.Fill(colSeaDay)
	}
	drawWaves(screen, w.Time, night)

	if night {
		drawBeam(screen, w)
	} else if w.LampPlaced && w.Owned[sim.SecondLamp] > 0 {
		vector.StrokeCircle(screen, f32(w.LampPos.X), f32(w.LampPos.Y), sim.SecondLampR, 2, color.RGBA{255, 230, 150, 255}, true)
	}
	for _, r := range w.Rocks {
		drawRock(screen, r, night)
	}
	for _, b := range w.Boats {
		drawBoat(screen, b, w.Time)
	}
	drawIsland(screen, w, night)
	for _, f := range w.Fogs {
		drawFog(screen, f, w.Time)
	}
	drawEffects(screen, w)
	if w.Storm && w.Phase == sim.PhaseNight {
		drawRain(screen, w)
	}
	if w.Flash > 0 {
		vector.DrawFilledRect(screen, 0, 0, sim.Width, sim.Height, rgba(color.RGBA{220, 230, 255, 255}, 0.6*w.Flash), false)
	}

	switch w.Phase {
	case sim.PhaseNight:
		drawHUD(screen, w)
	case sim.PhaseDay:
		drawDay(screen, w)
	case sim.PhaseWon:
		drawEnd(screen, "Dawn after the tenth night", fmt.Sprintf("You kept the light burning. %d boats came home safely.", w.TotalSaved), colGood)
	case sim.PhaseLost:
		drawEnd(screen, "The run is over", w.LoseReason, colWarn)
	}
}

func f32(v float64) float32 { return float32(v) }

// rgba returns c at opacity a, premultiplied as image/color expects.
func rgba(c color.RGBA, a float64) color.RGBA {
	a = clamp01(a)
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(a * 255)}
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// fillPoly fills a polygon with a single (premultiplied) color.
func fillPoly(dst *ebiten.Image, pts []sim.Vec, c color.RGBA, blend ebiten.Blend) {
	var p vector.Path
	for i, q := range pts {
		if i == 0 {
			p.MoveTo(f32(q.X), f32(q.Y))
		} else {
			p.LineTo(f32(q.X), f32(q.Y))
		}
	}
	p.Close()
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR = float32(c.R) / 255
		vs[i].ColorG = float32(c.G) / 255
		vs[i].ColorB = float32(c.B) / 255
		vs[i].ColorA = float32(c.A) / 255
	}
	dst.DrawTriangles(vs, is, whitePixel, &ebiten.DrawTrianglesOptions{
		AntiAlias:      true,
		Blend:          blend,
		ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha,
	})
}

func drawWaves(dst *ebiten.Image, t float64, night bool) {
	c := color.RGBA{30, 50, 90, 255}
	if !night {
		c = color.RGBA{150, 195, 225, 255}
	}
	for y := 30.0; y < sim.Height; y += 46 {
		for x := 0.0; x < sim.Width; x += 70 {
			ox := math.Mod(x+y*0.7+t*8, sim.Width+70) - 35
			vector.StrokeLine(dst, f32(ox), f32(y), f32(ox+14), f32(y), 1.5, c, true)
		}
	}
}

func cone(origin sim.Vec, angle, half, reach float64) []sim.Vec {
	pts := []sim.Vec{origin}
	const steps = 24
	for i := 0; i <= steps; i++ {
		a := angle - half + 2*half*float64(i)/steps
		pts = append(pts, origin.Add(sim.FromAngle(a).Scale(reach)))
	}
	return pts
}

func drawBeam(dst *ebiten.Image, w *sim.World) {
	if w.LampPlaced && w.Owned[sim.SecondLamp] > 0 {
		for i := 3; i >= 1; i-- {
			r := sim.SecondLampR * float64(i) / 3
			vector.DrawFilledCircle(dst, f32(w.LampPos.X), f32(w.LampPos.Y), f32(r), rgba(colBeam, 0.08), true)
		}
	}
	if !w.BeamOn() {
		return
	}
	// Layered cones give a cheap glow that fades with distance.
	reach := w.BeamRange()
	for i := 4; i >= 1; i-- {
		k := float64(i) / 4
		fillPoly(dst, cone(sim.LighthousePos, w.BeamAngle, w.BeamHalfAngle()*(0.7+0.3*k), reach*k), rgba(colBeam, 0.15), ebiten.BlendSourceOver)
	}
}

func drawRock(dst *ebiten.Image, r *sim.Rock, night bool) {
	c := colRockDark
	switch {
	case !night:
		c = color.RGBA{70, 70, 75, 255}
	case r.Lit:
		c = colRockLit
	case r.IsKnown():
		c = color.RGBA{50, 52, 60, 255}
	}
	x, y := f32(r.Pos.X), f32(r.Pos.Y)
	if r.Log {
		vector.StrokeLine(dst, x-f32(r.R)*1.8, y, x+f32(r.R)*1.8, y, f32(r.R)*1.1, c, true)
	} else {
		// A lumpy rock: three overlapping circles.
		vector.DrawFilledCircle(dst, x, y, f32(r.R), c, true)
		vector.DrawFilledCircle(dst, x+f32(r.R)*0.5, y-f32(r.R)*0.35, f32(r.R)*0.7, c, true)
		vector.DrawFilledCircle(dst, x-f32(r.R)*0.55, y+f32(r.R)*0.2, f32(r.R)*0.6, c, true)
	}
	if night && r.IsKnown() {
		// The "known" ring fades as the warning expires.
		a := 0.9
		if !r.Buoy {
			a = 0.25 + 0.65*r.Known/sim.KnownTime
		}
		vector.StrokeCircle(dst, x, y, f32(r.R)+7, 1.5, rgba(colWarn, a), true)
	}
	if r.Buoy {
		vector.DrawFilledCircle(dst, x+f32(r.R)+8, y-f32(r.R), 4, color.RGBA{230, 60, 50, 255}, true)
	}
}

func drawBoat(dst *ebiten.Image, b *sim.Boat, t float64) {
	s := b.Spec()
	f := sim.FromAngle(b.Heading)
	side := sim.Vec{X: -f.Y, Y: f.X}
	length, beam := s.Radius*1.9, s.Radius*0.9
	p := b.Pos
	hull := []sim.Vec{
		p.Add(f.Scale(length)),
		p.Add(f.Scale(length * 0.2)).Add(side.Scale(beam)),
		p.Add(f.Scale(-length * 0.8)).Add(side.Scale(beam * 0.8)),
		p.Add(f.Scale(-length * 0.8)).Sub(side.Scale(beam * 0.8)),
		p.Add(f.Scale(length * 0.2)).Sub(side.Scale(beam)),
	}
	hullCol := map[sim.BoatKind]color.RGBA{
		sim.Fishing:   {150, 110, 80, 255},
		sim.Merchant:  {120, 120, 140, 255},
		sim.Speedboat: {210, 210, 220, 255},
		sim.Ferry:     {200, 80, 70, 255},
		sim.Rowboat:   {170, 140, 90, 255},
	}[b.Kind]
	if !b.Lit {
		hullCol = color.RGBA{hullCol.R / 3, hullCol.G / 3, hullCol.B / 3, 255}
	}
	fillPoly(dst, hull, hullCol, ebiten.BlendSourceOver)

	// Every boat carries a tiny flickering lantern so the player can spot it.
	flicker := 0.75 + 0.25*math.Sin(t*9+p.X)
	vector.DrawFilledCircle(dst, f32(p.X), f32(p.Y), 9, rgba(colLantern, 0.18*flicker), true)
	vector.DrawFilledCircle(dst, f32(p.X), f32(p.Y), 2.5, colLantern, true)
	if b.Slow > 0 {
		vector.StrokeCircle(dst, f32(p.X), f32(p.Y), f32(s.Radius)+6, 1, rgba(colText, 0.5), true)
	}
}

func drawIsland(dst *ebiten.Image, w *sim.World, night bool) {
	sand := colSand
	if night {
		sand = color.RGBA{70, 64, 50, 255}
	}
	ip := sim.LighthousePos
	vector.DrawFilledCircle(dst, f32(ip.X-20), f32(ip.Y+12), sim.IslandRadius*0.7, sand, true)
	vector.DrawFilledCircle(dst, f32(ip.X+18), f32(ip.Y+14), sim.IslandRadius*0.65, sand, true)
	vector.DrawFilledCircle(dst, f32(ip.X), f32(ip.Y), sim.IslandRadius*0.85, sand, true)

	// Boats dock anywhere along this ring around the island.
	vector.StrokeCircle(dst, f32(ip.X), f32(ip.Y), sim.IslandRadius+4, 2, rgba(colGood, 0.5), true)

	// Lighthouse tower.
	lp := sim.LighthousePos
	vector.DrawFilledCircle(dst, f32(lp.X), f32(lp.Y), 14, color.RGBA{230, 230, 225, 255}, true)
	vector.StrokeCircle(dst, f32(lp.X), f32(lp.Y), 14, 3, color.RGBA{190, 50, 50, 255}, true)
	lamp := color.RGBA{90, 80, 60, 255}
	if night && w.BeamOn() {
		lamp = colBeam
	}
	vector.DrawFilledCircle(dst, f32(lp.X), f32(lp.Y), 6, lamp, true)
}

func drawFog(dst *ebiten.Image, f sim.Fog, t float64) {
	for i := 0; i < 5; i++ {
		a := float64(i)*1.3 + t*0.1
		off := sim.FromAngle(a).Scale(f.R * 0.35)
		vector.DrawFilledCircle(dst, f32(f.Pos.X+off.X), f32(f.Pos.Y+off.Y), f32(f.R*0.7), rgba(color.RGBA{150, 160, 175, 255}, 0.09), true)
	}
}

func drawRain(dst *ebiten.Image, w *sim.World) {
	c := rgba(color.RGBA{150, 170, 210, 255}, 0.3)
	for i := 0; i < 120; i++ {
		x := math.Mod(float64(i*97)+w.Time*w.Wind.X*8, sim.Width)
		y := math.Mod(float64(i*53)+w.Time*420, sim.Height)
		if x < 0 {
			x += sim.Width
		}
		vector.StrokeLine(dst, f32(x), f32(y), f32(x+w.Wind.X*0.6), f32(y+12), 1, c, false)
	}
}

func drawEffects(dst *ebiten.Image, w *sim.World) {
	for _, e := range w.Effects {
		k := e.T / sim.EffectDuration
		x, y := f32(e.Pos.X), f32(e.Pos.Y)
		switch e.Kind {
		case sim.EffectWreck:
			vector.StrokeCircle(dst, x, y, f32(6+k*30), 2, rgba(colText, 1-k), true)
			vector.StrokeCircle(dst, x, y, f32(3+k*18), 1.5, rgba(colText, 1-k), true)
			for i := 0; i < 6; i++ {
				d := sim.FromAngle(float64(i) * 1.05).Scale(8 + k*16)
				vector.DrawFilledRect(dst, x+f32(d.X), y+f32(d.Y), 4, 2, rgba(color.RGBA{120, 90, 60, 255}, 1-k*0.7), false)
			}
		case sim.EffectDock:
			drawText(dst, e.Text, faceMid, e.Pos.X-10, e.Pos.Y-20-k*30, rgba(colLantern, 1-k))
		case sim.EffectLightning:
			drawText(dst, "Lightning! "+e.Text, faceMid, e.Pos.X+24, e.Pos.Y-40-k*20, rgba(colWarn, 1-k))
		case sim.EffectHorn:
			vector.StrokeCircle(dst, x, y, f32(sim.HornRadius*k), 3, rgba(colText, 0.6*(1-k)), true)
		}
	}
}

func drawText(dst *ebiten.Image, s string, face *text.GoTextFace, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = face.Size * 1.35
	text.Draw(dst, s, face, op)
}

func drawCentered(dst *ebiten.Image, s string, face *text.GoTextFace, y float64, c color.Color) {
	wd, _ := text.Measure(s, face, face.Size*1.35)
	drawText(dst, s, face, (sim.Width-wd)/2, y, c)
}

func bar(dst *ebiten.Image, x, y, wd, v float64, c color.RGBA) {
	vector.DrawFilledRect(dst, f32(x), f32(y), f32(wd), 8, color.RGBA{40, 45, 60, 255}, false)
	vector.DrawFilledRect(dst, f32(x), f32(y), f32(wd*clamp01(v)), 8, c, false)
}

func drawHUD(dst *ebiten.Image, w *sim.World) {
	vector.DrawFilledRect(dst, 8, 8, 220, 116, colPanel, false)
	drawText(dst, fmt.Sprintf("Night %d / %d", w.Night, len(sim.Nights)), faceMid, 16, 12, colText)
	hearts := strings.Repeat("♥ ", w.Hearts) + strings.Repeat("· ", max(0, sim.HeartsPerNite-w.Hearts))
	drawText(dst, hearts, faceMid, 150, 12, colWarn)
	drawText(dst, fmt.Sprintf("Coins %d   Boats left %d", w.Coins, w.Remaining()), faceSmall, 16, 38, colText)

	oilLabel := "Oil"
	if !w.Config().OilLimit {
		oilLabel = "Oil (unlimited tonight)"
	}
	drawText(dst, oilLabel, faceSmall, 16, 58, colDim)
	bar(dst, 16, 78, 200, w.Oil/w.OilCap(), colBeam)
	drawText(dst, "Lighthouse", faceSmall, 16, 88, colDim)
	bar(dst, 16, 108, 200, w.Condition/sim.MaxCondition, colGood)

	var status []string
	if !w.LampOn {
		status = append(status, "LAMP OFF (Space)")
	} else if !w.BeamOn() {
		status = append(status, "OUT OF OIL")
	}
	if w.Owned[sim.Foghorn] > 0 {
		if w.HornTimer > 0 {
			status = append(status, fmt.Sprintf("Foghorn %.0fs", w.HornTimer))
		} else {
			status = append(status, "Foghorn ready (right click)")
		}
	}
	if len(status) > 0 {
		drawText(dst, strings.Join(status, "   "), faceSmall, 16, 130, colWarn)
	}

	if w.Time < 7 {
		a := clamp01(7 - w.Time)
		drawText(dst, w.Config().Intro, faceMid, 244, 12, rgba(colText, a))
		drawText(dst, "Move the mouse to aim · hold left click to focus · Space toggles the lamp", faceSmall, 244, 40, rgba(colDim, a))
	}
}

func drawDay(dst *ebiten.Image, w *sim.World) {
	const px, py, pw, ph = 150, 40, 660, 470
	vector.DrawFilledRect(dst, px, py, pw, ph, rgba(color.RGBA{250, 244, 228, 255}, 0.93), false)
	ink := color.RGBA{50, 45, 40, 255}
	soft := color.RGBA{110, 100, 90, 255}
	drawText(dst, fmt.Sprintf("Morning after night %d", w.Night), faceBig, px+24, py+14, ink)
	drawText(dst, fmt.Sprintf("Docked %d · Wrecked %d · Coins %d · Lighthouse %.0f%%", w.Docked, w.Wrecked, w.Coins, w.Condition), faceSmall, px+24, py+62, soft)
	drawText(dst, "A captain's note:", faceSmall, px+24, py+92, soft)
	drawText(dst, w.Note, faceSmall, px+24, py+112, ink)

	y := float64(py + 170)
	drawText(dst, "Shop — press a number to buy", faceMid, px+24, y, ink)
	y += 30
	for i, u := range sim.Upgrades {
		c := ink
		owned := ""
		if w.Owned[i] >= u.Max {
			c, owned = soft, "  (owned)"
		} else if w.Owned[i] > 0 {
			owned = fmt.Sprintf("  (x%d)", w.Owned[i])
		}
		drawText(dst, fmt.Sprintf("%d   %s", i+1, u.Name), faceSmall, px+24, y, c)
		drawText(dst, fmt.Sprintf("%d", u.Cost), faceSmall, px+200, y, c)
		drawText(dst, u.Effect+owned, faceSmall, px+240, y, c)
		y += 22
	}
	drawText(dst, "R   Repair lighthouse", faceSmall, px+24, y, ink)
	drawText(dst, fmt.Sprintf("%d", sim.RepairCost), faceSmall, px+200, y, ink)
	drawText(dst, fmt.Sprintf("+%.0f%% condition", sim.RepairAmount), faceSmall, px+240, y, ink)
	y += 34
	if w.Message != "" {
		drawText(dst, w.Message, faceSmall, px+24, y, color.RGBA{160, 70, 40, 255})
	}
	drawText(dst, "Oil is refilled every morning.   Press Space to light the lamp →", faceMid, px+24, py+ph-36, ink)
}

func drawEnd(dst *ebiten.Image, title, body string, c color.RGBA) {
	vector.DrawFilledRect(dst, 0, 0, sim.Width, sim.Height, color.RGBA{0, 0, 0, 150}, false)
	drawCentered(dst, title, faceBig, sim.Height/2-70, c)
	drawCentered(dst, body, faceMid, sim.Height/2-10, colText)
	drawCentered(dst, "Click or press Space to play again", faceSmall, sim.Height/2+30, colDim)
}
