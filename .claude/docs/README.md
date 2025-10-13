# Documentation Index

## Project Overview
- 📘 [Main Project Instructions](../../CLAUDE.md) - Core design notes, tech stack, and project guidelines
- ⚡ [**CURRENT STATE**](CURRENT_STATE.md) - **START HERE**: Quick reference for current features and systems
- 🏗️ [System Architecture](ARCHITECTURE.md) - Technical implementation guide and handoff documentation
- 🌐 [Network Protocol](NETWORK_PROTOCOL.md) - Formal protocol specification (Quake 3 model)
- 🗺️ [Map System Design](MAP_SYSTEM.md) - Terrain, camera, and occlusion system (in progress)

## Planning Documents
- 📋 [Godot Implementation Plan](planning/GODOT_PLAN.md) - Initial planning for Godot client architecture
- 🎨 [Theme Ideas](planning/Theme%20Ideas.md) - Brainstorming for game themes

## Sprint Documentation

### Sprint 1 - Networking Core ✅
- 📝 [Sprint 1 Plan](sprints/SPRINT_1_PLAN.md) - Implementation checklist and tasks
- ✅ [Sprint 1 Complete](sprints/SPRINT_1_COMPLETE.md) - Summary of accomplishments and metrics

### Sprint 2 - Game Rules & Mechanics ✅
- ✅ [Sprint 2 Complete](sprints/SPRINT_2_COMPLETE.md) - Building, resources, combat, and network refactor

### Sprint 3 - RTS Controls & Formations ✅
- ✅ [Sprint 3 Progress](sprints/SPRINT_3_PROGRESS.md) - Multi-unit control, formations, isometric rendering, drag-to-select

### Map System - Phases 1-3 ✅
- ✅ [Map System Phases 1-3 Complete](sprints/MAP_SYSTEM_PHASES_1-3_COMPLETE.md) - File-based maps, camera controls, terrain rendering

### Future Work
- Sprint 4+ - Win conditions, more unit types, pathfinding (upcoming)

## Quick Links
- **Server Code**: `/server/main.go`
- **Client Code**: `/client/` (Godot project)
- **Test Client**: `/test_client.go`
- **Launch Scripts**:
  - `/launch_client.sh` - Single client
  - `/launch_all.sh` - Server + multiple clients with color-coded logs

## Organization Structure
```
.claude/
├── docs/
│   ├── README.md (this file)
│   ├── CURRENT_STATE.md ⭐ START HERE
│   ├── ARCHITECTURE.md
│   ├── NETWORK_PROTOCOL.md
│   ├── MAP_SYSTEM.md
│   ├── planning/
│   │   ├── GODOT_PLAN.md
│   │   └── Theme Ideas.md
│   └── sprints/
│       ├── SPRINT_1_PLAN.md
│       ├── SPRINT_1_COMPLETE.md
│       ├── SPRINT_2_COMPLETE.md
│       ├── SPRINT_3_PROGRESS.md
│       └── MAP_SYSTEM_PHASES_1-3_COMPLETE.md
└── settings.local.json
```