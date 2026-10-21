# Tide Keeper — Game Design Document

Oct 2, 2026 · @Erick

## Overview

Tide Keeper is a calm-but-tense, single-screen management game where you keep a lonely lighthouse running and guide boats safely past the rocks at night. It is non-violent, readable in seconds, and scoped to be built in 7 to 15 days.

| Item | Detail |
| --- | --- |
| Genre | Light strategy / management, arcade tension |
| Platform | PC (web build first), mobile-friendly controls later |
| Session length | 15–30 minutes per run; one night ≈ 2–3 minutes |
| Audience | Casual players who like Mini Metro, Harbor Master, cozy games |
| Player fantasy | "I am the only thing keeping these sailors alive tonight." |
| Win / lose | Survive 10 nights to win; the run ends if 3 boats wreck in one night or the lighthouse breaks down |

The hook is the rotating beam: instead of drawing paths or placing objects, you light up a moving area, and boats react to what they can see.

## Core gameplay loop

Each run alternates between a tense night phase and a calm day phase, ten times.

**Night phase (2–3 minutes, real-time).** Boats appear at the screen edges and sail toward the harbor. They can only steer around rocks they can see, so you rotate the lighthouse beam to reveal danger in time. Your lamp burns oil, and a wider or brighter beam burns it faster. Every boat that docks pays coins; every wreck costs reputation.

**Day phase (untimed).** You walk around the small island and spend the night's coins. You refill oil, repair storm damage, buy upgrades, and read a short note from a grateful (or angry) captain. Then you press "Light the lamp" to start the next night.

**Why it works.** The night creates pressure and split attention; the day gives a breather and meaningful choices. Each upgrade changes how the next night feels, which keeps the loop fresh across ten nights.

## Mechanics and controls

The whole game runs on one input: pointing the beam. Everything else is boats reacting to light.

**Controls**

- Mouse / touch: drag to rotate the beam toward the cursor.
- Hold click / tap-and-hold: focus the beam (narrower, longer reach, faster oil use).
- Right click or button: sound the foghorn (unlocked later) to slow nearby boats.
- Day phase: click buildings and shop items; no movement skills needed.

**Boat behavior**

- Boats follow a simple path toward the harbor.
- A rock inside the beam becomes "known" to every boat for a few seconds; boats steer around known rocks.
- An unlit rock in a boat's path means a crash unless you light it in time.

**Boat types**

| Boat | Speed | Reward | Twist |
| --- | --- | --- | --- |
| Fishing boat | Slow | 5 coins | Forgiving, good for learning |
| Merchant ship | Medium | 12 coins | Wide hull, needs early warning |
| Speedboat | Fast | 10 coins | Reacts late, needs quick aim |
| Ferry | Slow | 25 coins | Losing it costs 2 reputation |
| Lost rowboat | Erratic | 15 coins | Wanders; must be followed with the beam |

**Hazards**

- Rocks: static, placed per night.
- Drifting icebergs or logs: slow-moving rocks.
- Fog banks: shrink the beam's reach where they pass.
- Storms: wind pushes boats off course and can damage the lighthouse.

**Resources and failure**

- Oil: drains while the lamp is on; empty means darkness until refilled.
- Reputation: 3 hearts per night; each wreck removes one; zero ends the run.
- Lighthouse condition: storm damage lowers beam range until repaired.

## Progression

A run lasts 10 nights, and each night adds one new idea so players are never overwhelmed.

| Night | New element | Boats |
| --- | --- | --- |
| 1 | Fishing boats, a few rocks (tutorial) | 4 |
| 2 | Oil limit introduced | 6 |
| 3 | Merchant ships | 7 |
| 4 | Light fog | 8 |
| 5 | Speedboats | 9 |
| 6 | Drifting logs | 10 |
| 7 | Ferry (high stakes) | 10 |
| 8 | First storm | 11 |
| 9 | Lost rowboats, heavy fog | 12 |
| 10 | Final storm night, everything combined | 14 |

**Upgrades (bought in the day phase)**

| Upgrade | Cost | Effect |
| --- | --- | --- |
| Bigger oil tank | 20 | +50% oil capacity |
| Polished lens | 30 | +25% beam range |
| Wide reflector | 35 | Wider beam angle |
| Faster gears | 25 | Beam rotates 30% faster |
| Foghorn | 40 | Slows boats in a radius once per 20 s |
| Second lamp | 60 | A small fixed light you aim during the day |
| Storm shutters | 30 | Halves storm damage |
| Harbor buoy | 15 | Permanently marks one rock |

Costs are starting numbers for playtesting, not final balance.

## Art, audio and mood

The look is minimalist and dark, so the beam of light is the most beautiful thing on screen.

**Visual style**

- Top-down view, flat shapes or low-res pixel art.
- Night palette: deep navy sea, near-black rocks, warm yellow beam.
- Day palette: soft blues and sandy beige for contrast and relief.
- Boats get a tiny lantern each, so players spot them before the beam does.
- Wrecks show a short splash and floating debris, never people in danger.

**Audio**

- Constant layer of waves and wind; volume rises with storms.
- A low, slow piano or ambient track at night; a lighter tune in the day.
- Clear cues: a bell when a boat docks, a creak when a boat nears a rock, a heavy thud on a wreck.

**Tone and story**

Captains leave short notes each morning: thanks, complaints, small gossip about the town. Over 10 nights these hint at why the old keeper left, giving players a reason to reach the last night.

## Development schedule

The plan assumes a solo developer with 12 working days; it can be squeezed into a 7-day jam or stretched to 15.

1. **Day 1 — Prototype the beam.** Rotating cone of light, one rock, one boat on a fixed path. Gray boxes only.
2. **Day 2 — Boat AI.** Boats avoid lit rocks, crash into unlit ones. If this isn't fun yet, fix it before moving on.
3. **Day 3 — Night structure.** Spawning, docking, coins, reputation hearts, end-of-night screen.
4. **Day 4 — Oil and focus beam.** Resource drain, empty-lamp state, focus mode.
5. **Day 5 — Day phase.** Shop screen, upgrades applied to the next night, "Light the lamp" button.
6. **Days 6–7 — Content.** All boat types, fog, drifting logs, storms; build the 10 nights.
7. **Day 8 — Art pass.** Replace gray boxes, night/day palettes, beam glow.
8. **Day 9 — Audio.** Ambient layers, music, feedback sounds.
9. **Day 10 — Story notes and menus.** Captain notes, title screen, win/lose screens.
10. **Days 11–12 — Playtest and polish.** Balance costs and speeds, fix bugs, build and upload.

For a 7-day version: cut storms, the second lamp, and story notes, and run 5 nights instead of 10.

## Stretch goals, risks and references

**Stretch goals (only if time remains)**

- Endless mode with a high-score table.
- Daily seed: everyone plays the same night layout.
- Unlockable lighthouse skins and new islands.
- Mobile port with tap-to-aim controls.

**Risks and fallbacks**

| Risk | Fallback |
| --- | --- |
| Boat avoidance feels unfair or buggy | Use fixed lanes that switch when a rock is lit |
| Beam lighting effect is slow to build | Use a semi-transparent cone sprite instead of real-time lights |
| Balance takes too long | Lock to 5 nights and tune those well |
| Game feels too easy | Add more boats at once before adding new mechanics |

**References to study**

- Harbor Master and Flight Control: guiding multiple vessels under time pressure.
- Mini Metro: minimal single-screen clarity.
- Kingdom Two Crowns and Moonlighter: day/night economy loops.
