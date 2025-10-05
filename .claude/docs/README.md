# Documentation Index

## Project Overview
- 📘 [Main Project Instructions](../../CLAUDE.md) - Core design notes, tech stack, and project guidelines
- 🏗️ [System Architecture](ARCHITECTURE.md) - Technical implementation guide and handoff documentation
- 🌐 [Network Protocol](NETWORK_PROTOCOL.md) - Formal protocol specification (Quake 3 model)

## Planning Documents
- 📋 [Godot Implementation Plan](planning/GODOT_PLAN.md) - Initial planning for Godot client architecture
- 🎨 [Theme Ideas](planning/Theme%20Ideas.md) - Brainstorming for game themes

## Sprint Documentation

### Sprint 1 - Networking Core ✅
- 📝 [Sprint 1 Plan](sprints/SPRINT_1_PLAN.md) - Implementation checklist and tasks
- ✅ [Sprint 1 Complete](sprints/SPRINT_1_COMPLETE.md) - Summary of accomplishments and metrics

### Sprint 2 - Game Rules & Mechanics ✅
- ✅ [Sprint 2 Complete](sprints/SPRINT_2_COMPLETE.md) - Building, resources, combat, and network refactor

### Future Sprints
- Sprint 3 - Playtesting & Iteration (upcoming)

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
│   ├── ARCHITECTURE.md
│   ├── NETWORK_PROTOCOL.md
│   ├── planning/
│   │   ├── GODOT_PLAN.md
│   │   └── Theme Ideas.md
│   └── sprints/
│       ├── SPRINT_1_PLAN.md
│       ├── SPRINT_1_COMPLETE.md
│       └── SPRINT_2_COMPLETE.md
└── settings.local.json
```