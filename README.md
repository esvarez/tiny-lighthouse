# Tide Keeper

A small lighthouse game: boats sail toward the lighthouse in the middle of the
bay, and you point its rotating beam to guide them safely past the rocks. This is a proof of concept written in Go with
[Ebitengine](https://ebitengine.org), based on
[the game design document](docs/game-design.md).

## Run

```sh
go run .                 # desktop
go run . -night 8        # jump to a later night for playtesting
make serve               # web build at http://localhost:8080
go test ./...            # game rule tests
```

On Linux, Ebitengine needs the X11/GL headers
(`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev`).

## How to play

**Night**
- Move the mouse to aim the beam. Hold left click to focus it: narrower, longer reach, more oil.
- Boats come in from every edge and sail toward the lighthouse. A boat in or near the beam turns toward it
  and follows the light home, so sweep the beam to lead boats around rocks.
- A rock inside the beam becomes known to every boat for 5 seconds (orange ring), and nearby captains remember it.
  Boats steer around rocks they know and crash into rocks they don't.
  Light rocks early: captains need a moment to react and turn gradually, so a rock lit too late is hit anyway.
- Space turns the lamp off and on to save oil (oil is limited from night 2).
- Right click (or F) sounds the foghorn once you have bought it.
- Each night you have 3 hearts. A wreck costs one (the ferry costs two). Run out and the run ends.

**Day**
- Read the captain's note, press 1–8 to buy upgrades, R to repair storm damage.
- With the second lamp, click the sea to place it.
- Space starts the next night. Survive all 10 nights to win.

## What's in the POC

- All 10 nights from the progression table, each introducing one new idea.
- All five boat types (fishing, merchant, speedboat, ferry, lost rowboat).
- Hazards: rocks, drifting logs, fog banks that shorten the beam, storms with wind and lightning damage.
- Oil, focus mode, lighthouse condition, coins and reputation hearts.
- All eight upgrades, and a short captain's note each morning.

Not included yet: audio, art pass, title or settings menus.

## Code layout

- `internal/sim` holds the game rules and has no rendering dependencies, so it is tested headlessly.
  `TestNightsArePlayable` runs a simple bot through every night to catch layouts that stall or are impossible.
- `main.go` maps mouse, keyboard and touch input to `sim.Input`.
- `draw.go` renders everything with flat vector shapes.
