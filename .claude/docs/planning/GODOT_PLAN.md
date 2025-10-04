# Godot Learning Plan — 7-day focused path to a networked prototype

Goal
----
Get you from no Godot experience to a minimal, networked Godot client that can connect to the prototype Go server, send input batches, and render remote snapshots. Prioritized for speed-to-playable while teaching the core concepts you'll use daily.

Prerequisites
-------------
- A development machine (you mentioned macOS). For Windows builds later you can export from macOS.
- Basic programming experience (you have Go experience; Godot + GDScript is quick to pick up).

Install (macOS / zsh)
---------------------
- Install Godot 4.x (non-Mono recommended for fastest start):

```bash
brew install --cask godot
```

- If you want C# support (optional):

```bash
brew install --cask godot-mono
```

- Install protoc (optional, only if using Protobuf locally):

```bash
brew install protobuf
```

- Optional tools: a text editor (VS Code), Git, and Docker (for running your Go server in a container).

Core concepts to learn (priority order)
-------------------------------------
1. Editor & workflow (Scenes, Nodes, Inspector, FileSystem)
2. GDScript basics (syntax, lifecycle methods, signals)
3. Scenes & instancing (PackedScene, instancing players/entities)
4. Input & physics loop (_process vs _physics_process, InputMap)
5. UI & Control nodes (HUD, buttons, labels)
6. Signals and messaging (connect, emit)
7. Networking: PacketPeerUDP (low-level) and ENetMultiplayerPeer (mid-level)
8. Client-side prediction & interpolation (network smoothing and reconciliation)
9. Exporting projects (install export templates and build a Windows export)

Which networking API to pick for the prototype
---------------------------------------------
- PacketPeerUDP (raw UDP): maximum control; you'll implement framing, reliability, and NAT handling yourself. Best if you want protocol parity with a Go UDP server.
- ENet / ENetMultiplayerPeer: provides reliable/unreliable channel semantics; easier than raw UDP and good for many realtime games.
- RPC/High-level Godot multiplayer: not recommended for tick-based authoritative server patterns where you need precise input batching and reconciliation.

Recommendation: start with PacketPeerUDP for parity with the Go server skeleton. Later, if you want simpler code and don't need absolute control, move to ENet.

Daily learning plan (7 days)
---------------------------
Day 0 — Setup (couple hours)
- Install Godot and open the editor.
- Create a new project `realtime-game-client` and commit to Git.
- Install export templates (Editor -> Manage Export Templates).

Day 1 — Editor fundamentals & GDScript (3–4 hours)
- Complete the official "Your first 2D game" tutorial or equivalent.
- Learn GDScript syntax: variables, functions, classes, signals, extends, typing basics.
- Exercise: create a simple Player scene that moves with arrow keys.

Day 2 — Scenes, instancing, and UI (3–4 hours)
- Learn PackedScene, instancing scenes at runtime, and saving/loading scenes.
- Create a HUD with a label showing player money and a build button.
- Exercise: spawn a second Player from code (simulate a remote player) and control its transform with a script.

Day 3 — Input model & physics loops (3–5 hours)
- Learn InputMap, _process (render), and _physics_process (fixed-step physics).
- Implement local prediction: apply input locally and store input history for later reconciliation.
- Exercise: implement local movement with history of input frames.

Day 4 — Networking basics (4–6 hours)
- Learn how to use PacketPeerUDP (or ENet) to send and receive bytes.
- Implement a basic network script that sends a Hello JSON packet and listens for a Welcome reply.
- Exercise: run a headless Go echo server (I can scaffold) and verify Godot receives Welcome.

Day 5 — Tick-based input batching & snapshot handling (4–8 hours)
- Implement a fixed-tick sender: send InputBatch at 20–30Hz (tick id, sequence, inputs).
- Implement snapshot processing: receive Snapshot messages and update or spawn remote entities.
- Exercise: show the remote player's box moving based on snapshots; implement interpolation between snapshot positions.

Day 6 — Reconciliation and smoothing (4–8 hours)
- Implement reconciliation: when server snapshot conflicts with predicted local state, roll forward stored inputs and smooth-correct visually.
- Implement interpolation for remote entities (buffer snapshots and interpolate by tick offset).
- Exercise: simulate packet loss and latency and verify smoothing and correction behavior.

Day 7 — Polish, export, and connect to Go server (4–8 hours)
- Connect to the real Go prototype server endpoint. Exchange Hello/Welcome, send inputs, receive authoritative snapshots.
- Add minimal HUD controls for build/interfere commands.
- Export a Windows build and test two clients connect to the Linux server (or run one exported and one in-editor).

Mini-project: Networked movement proof-of-life
--------------------------------------------
Deliverable: two running Godot clients and a Go server where each client can move a box and see the other's movement with interpolation and server reconciliation.

Implementation hints for your Godot scripts
-----------------------------------------
- Use PoolByteArray and get_packet()/put_packet() to send and receive raw bytes.
- For early debugging, JSON is fine (serialize to UTF-8); for real network tests use compact binary.
- Keep messages small: use int32 for ids/ticks, float32 for positions, and small enums for commands.
- Keep an input history buffer per client (vector of {tick, input}) to reapply during reconciliation.
- For interpolation: buffer last N snapshots (N = 2–5) and render remote entities by interpolating between the last two snapshots aligning to a fixed render delay (e.g., 100ms).

Sample message framing (compact, minimal)
----------------------------------------
- Frame header: 1-byte messageType, 4-byte tick (uint32), 2-byte payload length (uint16)
- MessageType examples: 1=Hello, 2=Welcome, 3=InputBatch, 4=Snapshot, 5=Event

Protobuf vs custom binary
-------------------------
- Protobuf: good for cross-language structs and versioning. Requires extra tooling to use in Godot (third-party libs or manual decoding).
- Custom binary: less setup, faster to implement in GDScript via PoolByteArray and byte reads/writes. Recommended for prototype.

Testing & debugging tips
------------------------
- Use Godot's remote debugger (Debug -> Remote) to inspect running nodes in a client instance.
- Print/log packet sizes and tick IDs during development.
- Simulate latency and packet loss by adding an artificial delay/packet drop layer in your Go test server or in a small local proxy.
- Use small map and few entities to keep snapshot size small while tuning.

Exporting and iOS notes
-----------------------
- To export to Windows from macOS: install export templates and create a Windows Desktop export preset; you can produce .exe builds directly from macOS.
- For iOS later you'll need Xcode and an Apple developer account; Godot supports iOS export but has extra setup (provisioning, code signing).

Resources (docs & tutorials)
---------------------------
- Godot official docs: https://docs.godotengine.org/en/stable/
- Networking: https://docs.godotengine.org/en/stable/tutorials/networking/high_level_multiplayer.html and PacketPeerUDP docs
- GDQuest tutorials (client prediction & interpolation): https://www.gdquest.com/
- HeartBeast tutorials (2D basics): https://www.youtube.com/c/HeartBeast
- Example articles: "client-side prediction in Godot" (search) and ENet usage in Godot docs

Checklist (ready-to-run)
------------------------
- [ ] Godot 4.x installed and project created
- [ ] Export templates installed
- [ ] Basic Player scene with local movement
- [ ] Network script that sends Hello and receives Welcome from server
- [ ] Input batching at fixed tick rate
- [ ] Snapshot handling + interpolation for remote players
- [ ] Reconciliation logic for local player
- [ ] Exported Windows build that can connect to server

Next engineering step I can do for you
-------------------------------------
- Scaffold a minimal Godot project (GDScript) that implements Hello/Welcome over UDP, InputBatch sender at 20Hz, and Snapshot consumer that spawns/moves boxes for remote players.
- Or scaffold the Go server + proto and a headless Go client first (if you want to validate networking before the Godot client).

If you want me to scaffold the Godot project now, tell me whether you prefer raw UDP (PacketPeerUDP) or ENet for the first skeleton. I suggest raw UDP for full control and parity with a custom Go server.
