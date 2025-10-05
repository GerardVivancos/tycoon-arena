# Real-time Multiplayer Game — Design notes & tech choices

This document captures the high-level design, engineering contract, tech-stack recommendation, networking model, and an MVP plan for a realtime multiplayer game inspired by Constructor (1997). The goal: get a small, playable prototype fast while keeping the design flexible to change theme later.

## One-liner

Players run competing businesses on the same map. They expand, sabotage, and defend; actions are realtime and parallel. The server is authoritative; clients do prediction and smoothing. Theme is undecided — prototype should be theme-agnostic.

## Goals for the prototype

- Produce a playable 1v1/1vN online match where players can build/own a few entities, issue commands, and perform simple interference actions against others.
- Authoritative dedicated Linux server (hosted by us).
- Cross-platform clients: Windows (first), macOS (likely), iOS later.
- Iterate quickly: fast contrast between ideas — theme independent.
- Prevent trivial cheating by validating on server.

## High-level engineering contract (tiny)

- Inputs: per-frame player inputs (move, build, trigger interference, select target). Each input is stamped with clientId and inputTick.
- Outputs: server snapshots (tick, authoritative entity states), event messages (build success/fail, damage, ownership change), matchmaking/lobby messages.
- Error modes: dropped packets, late inputs, desynced clients, client disconnects.
- Success criteria for prototype: ability for two clients to play a 3–5 minute match end-to-end with visible build/attack interactions and minimal perceived lag (client-side prediction and interpolation) on LAN and acceptable behaviour on moderate-latency internet.

## Recommended primary stack (fastest path to playable prototype)

- Server: Go (for rapid prototyping)
  - Why: Fast to iterate and prototype with. Excellent networking stdlib, lightweight goroutines, and simple UDP server setup. Perfect for getting something working quickly to find the fun.
  - Suggested packages: net (UDP), encoding/json (simple protocol), time (tick loops), log (basic logging).

- Client: Godot 4.x (GDScript or C#)
  - Why: Godot provides a rapid editor-driven iteration loop, built-in cross-platform export (Windows, macOS, iOS), easy scene-driven UI, and networking primitives. Great for a quick playable prototype where you want to iterate on gameplay rather than low-level rendering/engine details.
  - Networking: use Godot's PacketPeerUDP / low-level UDP or the built-in high-level multiplayer (ENet) if it meets needs. Using PacketPeerUDP gives full control and the ability to implement the same protocol the server expects.

- Serialization / Protocol: JSON (for rapid prototyping) - simple, debuggable, and works everywhere. Can switch to binary later for optimization.

- Deployment/ops: Docker image for the server + systemd or container supervisor on your Linux dedicated host; use Prometheus metrics + Grafana (optional) for early instrumentation.

Why this combination:
- Rapid client iteration with Godot (edit scenes, tweak quickly) — reduces time-to-playable.
- Go server allows extremely fast prototyping and iteration cycles.
- JSON protocol is immediately debuggable and works seamlessly between Go and Godot.

## Alternative stacks (pros/cons)

1) Go server + Godot client
   - Pros: you’re proficient in Go; very fast to put a UDP/TCP server together (goroutines are lightweight). The Go ecosystem has quic-go and enet bindings. Good for shipping quickly.
   - Cons: compared to Rust you trade off some memory-safety guarantees and, depending on libraries, raw throughput/latency can vary.

2) Rust server + Bevy client (Rust)
   - Pros: single-language (Rust) across server and client. Bevy makes game code modular and is quickly improving. Good if you prefer writing client logic in Rust.
   - Cons: Bevy's export maturity for iOS/mobile historically lags, and editor-driven workflows are not as fast as Godot for artists/designers. Will likely take longer to iterate for a prototype.

3) Unity client + Rust/Go server
   - Pros: very mature cross-platform, lots of assets and tooling, excellent networking options (Mirror/Netcode). Mobile export is straightforward.
   - Cons: heavier, less
   flexible for small prototypes and licensing/size concerns for an open toolchain.

   ## Networking model (authoritative server)

   - Tick-based authoritative server loop (fixed tick rate, e.g. 20–30Hz to start). Server advances authoritative simulation each tick. Clients send inputs with inputTick id. Server validates and applies inputs deterministically.
   - Transport: UDP-based with reliability layer for important events (build/ownership changes) and unreliable for frequent position/command inputs. Consider QUIC (via quinn) for built-in connection and stream multiplexing or implement a simple reliable messaging layer over UDP (sequence numbers + ACKs + retransmit).
   - Client-side prediction: clients apply local inputs immediately, rendering predicted state. Server snapshots arrive and reconciliation corrects predicted state with smooth correction (position snaps with interpolation / lerp over a few frames).
   - Snapshot strategy: send delta-compressed snapshots keyed by tick ID. For early prototype, sending full snapshot at 10–20Hz is fine.
   - Bandwidth considerations: keep entity state compact (float32 positions, small enums for states), send only changed entities; aim for <10KB/s per client for small matches.

   ## Data Model (minimal)

   - Player: id, name, cash, ownedEntities[]
   - Entity: id, ownerId, type, position (vec2), orientation, health, stateFlags
   - Command input: tick, clientId, sequence, commandType, payload
   - Event: tick, eventType, payload

   ## MVP feature list (fast path)

   - Core networking: server accepts client connections, tick loop, input apply, snapshot broadcast, basic reliable messaging.
   - Minimal game rules: place / build 1-3 types of structures, one interference action (e.g., trespass or sabotage), simple defense (repair). Structures generate currency over time.
   - Map: small bounded arena that fits 2–6 players.
   - UI: minimal HUD — player money, selected entity, build button, interference button, health bars.
   - Replayable match: start match from lobby, play for 3–5 minutes, determine winner by money or ownership.

   ## Implementation plan & primitive schedule (1–3 sprints)

   Sprint 0 — Setup (1–3 days)
   - Create repository layout: server/, client/, proto/.
   - Add basic README, CI skeleton, and Dockerfile for server.
   - Choose protocol (protobuf) and define minimal messages in `proto/`.

   Sprint 1 — Networking core (3–7 days) [CURRENT]
   - Implement Go server skeleton: UDP accept, client registry, tick loop, basic input handling, snapshot broadcast.
   - Implement Godot client: connect, send inputs, receive snapshots, render simple entities (colored boxes), client-side prediction for movement.
   - Smoke test on LAN with 2 clients.

   Sprint 2 — Game rules + polish (7–14 days)
   - Implement building & interference actions, server validation, event reliable delivery.
   - Add simple UI in Godot to trigger build/attack.
   - Add basic interpolation + reconciliation.

   Sprint 3 — Playtesting & iteration (ongoing)
   - Tune tick rates, latencies, and gameplay. Add minimal server logging and metrics. Start thinking about theme and art.

   ## Edge cases & risks

   - Latency & packet loss: mitigate with prediction, interpolation, and an appropriate tick rate. Test on simulated high-latency conditions.
   - Cheating & desync: keep server authoritative and never trust client state. Validate build positions, collision, and resource transactions server-side.
   - Scaling: prototype should target small matches; scaling to many simultaneous matches requires matchmaking and instance sharding (stateless server workers behind a gateway), and possibly leverage a lobby/service written in Go or Rust.

   ## Suggested technologies & libraries (concrete)

   - Server (Golang): tokio, quinn (QUIC) or laminar, prost (protobuf), serde/bincode (fallback), tracing, prometheus exporter.
   - Client (Godot): Godot 4.x, GDScript or C#, use low-level UDP sockets or Godot ENet API with custom protocol mapping. For iOS export later, Godot supports it directly.
   - Protobuf tooling: protoc + prost for Rust; for Godot, use a small generated JS/TS/GD wrapper or send compact binary frames and decode manually in GDScript.

   ## Minimal proof-of-concept protocol (example)

   - Connection handshake (client -> server): Hello(clientVersion, playerName)
   - Server -> client: Welcome(clientId, tickRate)
   - Client -> server: InputBatch(tickStart, [commands])
   - Server -> client: Snapshot(tick, [entityStates])
   - Server -> client reliable: Event(tick, eventType, details)

   Keep messages compact: use integers for enums and small integer IDs.

   ## Testing & quality gates

   - Unit tests: server-side logic for command validation, resource accounting, and deterministic simulation steps where possible.
   - Integration tests: spinning up server + headless clients that send scripted inputs to verify end-to-end loop.
   - Manual playtests: instrument simulated packet loss/latency and test client reconciliation.

   ## First milestone (deliverable)

   - A Go server that runs locally and accepts two client connections via UDP.
   - A Godot client that connects, spawns a box representing the player, can move and place a structure, and sees the other player's actions.

   ## Next steps (concrete tasks)

   1. Pick primary stack (Rust server + Godot client recommended). If you prefer to prototype faster in a language you already know well, choose Go server + Godot client; both are valid.
   2. Create repo skeleton with server/, client/, proto/ and add this `Claude.md` to root.
   3. Create `proto/messages.proto` with minimal messages and generate Rust prost code.
   4. Implement server core loop and a tiny Godot scene that connects and exchanges messages.
   5. Playtest on LAN, iterate networking parameters.

   ## Current Sprint Status

   **Sprint 1 - Networking Core (✅ COMPLETE)**
   - Go server with UDP networking, movement, and bounds checking
   - Godot client with prediction and interpolation
   - Multi-client support tested and working
   - See `.claude/docs/sprints/SPRINT_1_COMPLETE.md` for details

   **Sprint 2 - Game Rules & Mechanics (✅ COMPLETE)**
   - Building placement system with collision detection
   - Resource generation ($10/sec per generator)
   - Attack/sabotage mechanics (25 damage per hit)
   - Client-side prediction and validation
   - Network architecture refactored to Quake 3 model
   - See `.claude/docs/sprints/SPRINT_2_COMPLETE.md` for details

   **Sprint 3 - Playtesting & Iteration (NEXT)**
   - Tune tick rates, latencies, and gameplay
   - Add minimal server logging and metrics
   - Start thinking about theme and art

   **Documentation**
   - All sprint and planning documentation is in `.claude/docs/`
   - See `.claude/docs/README.md` for full index
   - Network protocol: `.claude/docs/NETWORK_PROTOCOL.md`

   ## Notes on language choice — short guidance

   - If you prefer Rust and want a high-performance, safe server: pick Rust. It will take slightly longer than Go only if you're less practiced, but the long-term benefits (memory safety, performance) are significant.
   - If you want to iterate *faster* on server logic and are very productive in Go: use Go. It's perfectly capable and will reduce initial dev friction.
   - For clients and fastest playable prototype: Godot — easiest cross-platform path including iOS later. If you need lots of asset-store content or C# tooling, Unity is the fallback.

   ## Appendix: quick decision checklist

   - Host server on Linux: both Rust and Go are great; Dockerize for portability.
   - Client platforms: Godot exports to Windows/macOS/iOS. Bevy/Unity are alternatives depending on language preference.
   - Protocol: JSON (current) for rapid prototyping and debugging; binary protocol (Protobuf/MessagePack) can be added later for optimization.

   ---

   If you want, I can now:
   - generate the initial repo skeleton (server/, client/, proto/) and a minimal `proto/messages.proto` file, or
   - produce a 1-page `proto/messages.proto` and a tiny Rust server skeleton (tick loop + UDP accept) and a Godot scene that can connect and exchange Hello/Welcome messages.

   Tell me which follow-up you want and I will implement it next.
