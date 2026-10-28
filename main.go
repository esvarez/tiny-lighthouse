// Command tide-keeper is a proof of concept of the Tide Keeper game: rotate a
// lighthouse beam so boats can see the rocks and reach the harbor safely.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/esvarez/tiny-lighthouse/internal/sim"
)

type Game struct {
	world    *sim.World
	touchIDs []ebiten.TouchID
	newTouch []ebiten.TouchID
	touchAt  map[ebiten.TouchID]time.Time

	// Debug mode (F1 or -debug): a left click spawns a boat of spawnKind at
	// the cursor, keys 1-5 pick the kind and Shift focuses the beam.
	debug     bool
	spawnKind sim.BoatKind
}

func (g *Game) Update() error {
	g.world.Update(1.0/float64(ebiten.TPS()), g.input())
	return nil
}

func (g *Game) input() sim.Input {
	x, y := ebiten.CursorPosition()
	in := sim.Input{
		Aim:   sim.Vec{X: float64(x), Y: float64(y)},
		Focus: ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		Horn:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyF),
		Click: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
	}
	space := inpututil.IsKeyJustPressed(ebiten.KeySpace)
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.debug = !g.debug
	}
	switch g.world.Phase {
	case sim.PhaseNight:
		in.ToggleLamp = space
		if g.debug {
			for i := range sim.Specs {
				if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
					g.spawnKind = sim.BoatKind(i)
				}
			}
			in.Focus = ebiten.IsKeyPressed(ebiten.KeyShift)
			if in.Click {
				in.Spawn = int(g.spawnKind) + 1
			}
		}
	case sim.PhaseDay:
		in.Start = space || enter
		in.Repair = inpututil.IsKeyJustPressed(ebiten.KeyR)
		for i := 0; i < len(sim.Upgrades); i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				in.Buy = i + 1
			}
		}
	default:
		in.Start = space || enter || in.Click
	}

	// Touch: aim with the first finger, holding it focuses the beam.
	g.touchIDs = ebiten.AppendTouchIDs(g.touchIDs[:0])
	g.newTouch = inpututil.AppendJustPressedTouchIDs(g.newTouch[:0])
	if len(g.touchIDs) > 0 {
		id := g.touchIDs[0]
		tx, ty := ebiten.TouchPosition(id)
		in.Aim = sim.Vec{X: float64(tx), Y: float64(ty)}
		if len(g.newTouch) > 0 && g.newTouch[0] == id {
			g.touchAt[id] = time.Now()
			in.Click = true
			if g.world.Phase != sim.PhaseNight && g.world.Phase != sim.PhaseDay {
				in.Start = true
			}
		}
		in.Focus = time.Since(g.touchAt[id]) > 300*time.Millisecond
	}
	return in
}

func (g *Game) Layout(int, int) (int, int) { return sim.Width, sim.Height }

func main() {
	night := flag.Int("night", 1, "night to start on (for playtesting)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed for the night layouts")
	debug := flag.Bool("debug", false, "start in debug mode (F1 toggles it in game)")
	flag.Parse()

	ebiten.SetWindowSize(sim.Width, sim.Height)
	ebiten.SetWindowTitle("Tide Keeper")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	g := &Game{
		world:   sim.New(*seed, *night),
		touchAt: map[ebiten.TouchID]time.Time{},
		debug:   *debug,
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
