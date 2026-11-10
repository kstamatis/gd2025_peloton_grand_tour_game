# Game Design Document (GDD): Peloton Grand Tour

> **Game Title**: Peloton Grand Tour: Breakaway
> **Target 2D Genre**: Arcade Peloton Racer & Tactical Stamina Breakaway Action
> **Camera Projection & Spatial Model**: 2D Side-Profile Multi-Lane Road (`4` distinct road lanes with depth sorting & scrolling parallax Alpine/Coastal/Parisian stages)
> **Simulation Model**: Real-Time Continuous (`60 TPS` fixed timestep in Ebitengine)
> **Target Platform**: Desktop (macOS/Windows/Linux) & WebAssembly (Browser / Game Dev Studio Playtest Tab)
> **Virtual Canvas & Aspect Ratio**: 16:9 Widescreen (`1280x720` for 2D HD sprites, parallax scenery, and crisp telemetry HUD)
> **Author / Lead Designer**: User & Antigravity Game Studio

---

## 1. Executive Summary & Elevator Pitch
- **Elevator Pitch**: *Peloton Grand Tour: Breakaway* is a high-speed 2D arcade cycling game where you ride through a 3-stage Grand Tour, tucking into rival slipstreams to conserve stamina, shifting gears across flat sprints and steep mountain gradients, and launching explosive aero breakaways to capture the Yellow Jersey (`Maillot Jaune`).
- **Core Inspiration**: *OutRun*, *Excitebike*, *Pro Cycling Manager* (arcade breakaway tension), and classic 16-bit SEGA arcade racers.
- **Target Mood & Pacing**: Energetic synthwave-infused European Grand Tour broadcast energy—balancing tense tactical drafting in the pack with high-adrenaline final-kilometer sprint finishes.

---

## 2. Core Gameplay Loop, Simulation & Mechanics
- **Primary Gameplay Loop**:
  1. **Read the Road & Pack (`ActionNavigate`)**: Switch between 4 horizontal road lanes (`Lane 0` top/inner to `Lane 3` bottom/outer) to avoid road hazards (cobblestone patches, wet slicks, dropped bidons) and position directly behind a rival rider (`0.8m` to `3.5m` wheel gap).
  2. **Draft in the Slipstream (Systemic Challenge)**: Riding in the wind drains Stamina proportionally to aerodynamic drag ($F_{\text{drag}} \propto v^2$) and road gradient ($\sin\theta$). Tucking into a rival's slipstream cone cuts wind resistance by **65%** and regenerates Stamina while displaying golden slipstream wind ribbons.
  3. **Attack, Hydrate & Sprint (Reward/Progression)**: Consume collected **Energy Gels / Bidons** (`X` / `Shift`) to restore **+35% Stamina**, trigger an **Aero Sprint Surge** (`Space` / `Z`) on mountain climbs or the final `500m` red kite (`Flamme Rouge`), and cross the finish arch in **1st Place**!
- **Spatial & Simulation Rules**:
  - **Movement & Interaction Topology**: Continuous horizontal road progression ($x$ in meters from `0m` to stage distance `2500m`/`3000m`/`3500m`) paired with 4 discrete/smooth-interpolated road lanes ($y \in \{470, 525, 580, 635\}$ virtual pixels, depth-sorted back-to-front).
  - **Player Abilities & Actions**:
    - **Cadence Pace (`Left` / `Right`)**: Adjust cruising effort (Coast/Recover, Tempo, Hard Pace) and brake cleanly around tight cobbles.
    - **Lane Switch (`Up` / `Down`)**: Smoothly steer across the 4 road lanes to catch slipstreams or pass slower riders.
    - **Aero Sprint Surge (`Space` / `Z`)**: Drops into an aggressive standing/aero sprint (`sprint` animation tag), boosting top speed by **+38%** and spawning speed-line particles at higher stamina burn.
    - **Consume Bidon / Energy Gel (`Shift` / `X`)**: Uses 1 stored Energy Gel charge (starts with 2 per stage, more can be grabbed from roadside feed zones) for an instant **+35 Stamina** and brief lactic recovery boost.
  - **Systemic Challenges, Rivals & 3 Grand Tour Stages**:
    - **Stage 1 — Côte d'Azur Coastal Sprint (`2,400m`)**: Rolling coastal roads, high-speed peloton trains, and a flat boulevard sprint finish.
    - **Stage 2 — Col du Galibier Alpine Climb (`2,800m`)**: Steep `6%` to `11%` mountain gradients where gravity slows the pack, climbers launch breakaway surges, and pacing Stamina is critical.
    - **Stage 3 — Champs-Élysées Cobblestone Finale (`3,200m`)**: Rough pavé cobble sectors that rattle speed and stamina unless navigated cleanly, culminating in a frantic sprint for the General Classification crown.
    - **Rival AI Archetypes**: *Domestiques* (steady pace-setters ideal for drafting), *Rouleurs* (launch mid-stage breakaways), *Grimpeurs* (surge on steep gradients), and *Sprinters* (explosive burst inside the final `600m`).
- **Win & Loss Conditions**:
  - **Stage Win / Grand Tour Victory**: Finish **1st Place** (or lowest cumulative General Classification time across all 3 stages) without bonking.
  - **Loss Condition**:
    - **Bonk / Exhaustion**: Player Stamina drops to `0` AND speed falls more than `180m` behind the peloton broom wagon (`Voiture Balai`), or
    - **Eliminated on Stage**: Finishing outside the podium cutoff required to advance.

---

## 3. Controls & Input Mapping Scheme
- **Primary Input Devices**: Keyboard, Gamepad, Mouse/Pointer
- **Default Control Scheme**:

| Logical Action | Keyboard / Mouse | Gamepad Button | Touch / Pointer |
| :--- | :--- | :--- | :--- |
| `ActionLaneUp / Down` | `W`/`S` or `Up`/`Down` Arrows | Left Stick Y / D-Pad Up-Down | Click/Drag Target Lane |
| `ActionPaceUp / Coast` | `D`/`A` or `Right`/`Left` Arrows | Left Stick X / D-Pad Left-Right | Hold Right/Left Side |
| `ActionSprint` (Aero Surge) | `Space` / `Z` / Left Click | Button South (`A` / `Cross`) | Hold Sprint Button |
| `ActionGel` (Energy Gel/Bidon) | `X` / `Shift` / `E` / Right Click | Button East (`B` / `Circle`) | Tap Gel Button |
| `ActionPause` | `Escape` / `P` | Start / Options | Pause Icon Button |

*(Note: `F8` and backtick `` ` `` are kept strictly unbound for the Game Dev Studio 10s DVR scrubber.)*

---

## 4. Visual Art Direction & Sprite Pipeline (`sprite-design` + `nano-banana`)
- **Aesthetic Direction & Color Palette**: Crisp **Hybrid 2D HD** (`1280x720` widescreen canvas) combining vibrant cel-shaded 2D HD cyclist sprites and scenic European tour backdrops (Path A `nano-banana` + `sprite-design` normalize, when Vertex AI / Gemini API credentials are active) with razor-sharp procedural `.sprite` SDF/pixel DSL assets (`sprite-design` Path B) using the `genesis` and `cyber` Retro-HD palettes and 4-step material ramps (`Shadow -> Base -> Light -> Specular`).
- **Sprite Resolution Tier**:
  - `[x]` **Native 2D HD (`256x256` per frame)** for hero/rival cyclists (or high-detail procedural `.sprite` cyclist rigs and road props).
- **Selected Visual Pipeline Mode**: **Path C: Hybrid Visual Pipeline** (`nano-banana` + `sprite-design` normalize for 2D HD character sheets/backdrops + pure-code `.sprite` SDF DSL for cyclist rigs, energy gels, bidons, cobble hazards, and finish arches).
- **Entity Animation State Tags (`AnimationController`)**:
  - **Player & Rival Cyclist (`cyclist_pro.sprite` / `cyclist_rival.sprite`)**:
    - `idle`: Track-stand / waiting at the start line (`4` frames, `pingpong`)
    - `cruise`: Seated aero pedaling cadence (`4` frames, `loop`)
    - `sprint`: Out-of-the-saddle standing power sprint (`4` frames, `loop`)
    - `tuck`: Deep aerodynamic supertuck descent/drafting pose (`4` frames, `loop`)
    - `tired`: Fatigued shoulders-bobbing bonk struggle (`4` frames, `loop`)
    - `victory`: Hands-raised no-hands finish line celebration (`4` frames, `pingpong`)
  - **Roadside Pickups & Props (`energy_gel.sprite`, `bidon_bottle.sprite`, `cobble_hazard.sprite`)**:
    - `spin` / `idle`: Bobbing animated collectible with specular glint (`4` frames, `loop`)

---

## 5. Audio & Soundscape Pipeline (`procedural-sfx` + `procedural-music`)
- **Selected Audio Pipeline Mode**: **Pure-Code Procedural (`.sfx` + `.music`)** synthesized in memory at `44.1 kHz` 16-bit stereo PCM (`len(pcm) &^ 3` 4-byte aligned).
- **Sound Effects (`procedural-sfx` `.sfx` DSL)**:
  - **Gameplay & Cycling Cues**:
    - `freewheel_click.sfx`: Crisp mechanical ratchet clicks when coasting or tucking into a slipstream.
    - `draft_whoosh.sfx`: Aerodynamic wind-tunnel lock-on whoosh when entering a rival's slipstream cone.
    - `gear_shift.sfx`: Snappy derailleur metallic clack when shifting effort up or down.
    - `sprint_surge.sfx`: Punchy upward synth + wind burst when triggering an Aero Sprint attack.
    - `gel_pickup.sfx`: Bright two-tone hydration/energy gel chime (`coin_pickup` + `gem_shimmer` layer).
    - `cobble_rattle.sfx`: Textured stone vibration impact when riding over rough pavé cobbles.
    - `fanfare_finish.sfx`: Uplifting 4-note podium celebration stinger when crossing the finish line.
  - **UI & Menus**:
    - `ui_select.sfx` & `ui_confirm.sfx`: Clean non-fatiguing menu navigation ticks.
- **Background Music (`procedural-music` `.music` DSL)**:
  - **Score**: `grand_tour_theme.music` — Original 10-channel SEGA YM2612 4-Op FM + SN76489 PSG synthwave/euro-tour racing anthem (`tempo allegro 138`, `4/4`, `A Minor / C Major`, `>45s` seamless loop with sub-bass & slap-bass anchored in `C1-C2`).
  - **Synthesis Engine**: `procedural-music` Default (Pure-Go YM2612 FM + SN76489 PSG) for 100% WebAssembly and desktop portability.

---

## 6. Scene Flow & UI / HUD Layout
- **Scene State Machine**: `Boot` -> `Intro` -> `Title Screen` (with 10s idle `Demo / Attract Mode`) -> `Stage Briefing / Gameplay` -> `Pause Overlay` -> `Stage Podium / Victory / Game Over` -> `Title Screen`
- **HUD & UI Regions (`ebiten/v2/text/v2` + Premultiplied `color.RGBA` Widgets)**:
  - **Top-Left (Rider Telemetry Panel)**: Speed (`km/h`), Cadence (`RPM`), Current Gear (`53x11` to `39x28`), Stamina Gauge (`0..100%` with color-coded Draft Recovery vs. Red-Zone Sprint burn), and Stored Energy Gels (`[X] Gel x2`).
  - **Top-Center (Stage Elevation Profile & Minimap Ribbon)**: Live course profile showing current gradient (`0.0%` to `+11.0%`), upcoming mountain climbs, feed zones, `1 km` Flamme Rouge marker, and dot positions of the Player (Yellow/Cyan) vs. Rival Peloton.
  - **Top-Right (Race Standings & General Classification)**: Live Position (`1st / 12`), Gap to Leader (`+0.0s` or `-2.4s`), Stage Timer, and Current Stage (`Stage 1/3: Côte d'Azur`).
  - **Bottom / Contextual Banner**: Slipstream Draft Lock indicator (`⚡ DRAFTING -65% WIND DRAG | +STAMINA`), Gradient Alert (`⛰️ COL DU GALIBIER +9.5%`), and Flamme Rouge Final Sprint callouts.

---

## 7. Ebitengine Architecture & Studio Workflow (`ebitengineer` + `game-dev-studio`)
- **Core Runtime Subsystems (`ebitengineer`)**:
  - **Coordinate & Collision System**: 2.5D lane-based road coordinates (`RoadMetersX`, `LaneY`) with AABB slipstream wake cones behind every rider, roadside pickup overlaps, and cobble sector hazard strips.
  - **Zero-Allocation Game Loop**: Pre-allocated rival/pickup/particle pools, pre-sliced `GridSpriteSheet` frames, pre-allocated Kage speed-line/vignette shader uniforms (`//kage:unit pixels`), and 4-byte aligned (`&^ 3`) PCM audio slices inside `Update()` and `Draw()`.
- **Interactive Workbench Verification (`game-dev-studio`)**:
  - **Sprite tab in Game Dev Studio / Sprite Editor (`sidecar://agy-game-studio/studio/sprite`)**: Inspect `.sprite` and `.aseprite` animation tags, frame alignment, and 4-step color ramps.
  - **Sound tab in Game Dev Studio / Sound Editor (`sidecar://agy-game-studio/studio/sound`)**: Audition `.sfx` and `.music` cues, verify `< 260 Hz` bass balance, and check loop points.
  - **Playtest tab in Game Dev Studio (`sidecar://agy-game-studio/studio/playtest`)**: Run the live 60 FPS WebAssembly build, scrub the 10-second DVR buffer, and capture annotated frame/GIF feedback for rapid tuning.
