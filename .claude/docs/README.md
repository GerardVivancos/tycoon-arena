# Documentation Index

## Project Overview
- 📘 [Main Project Instructions](../../Claude.md) - Core design notes, tech stack, and project guidelines
- 🏗️ [System Architecture](ARCHITECTURE.md) - Technical implementation guide and handoff documentation

## Planning Documents
- 📋 [Godot Implementation Plan](planning/GODOT_PLAN.md) - Initial planning for Godot client architecture
- 🎨 [Theme Ideas](planning/Theme%20Ideas.md) - Brainstorming for game themes

## Sprint Documentation

### Sprint 1 - Networking Core
- 📝 [Sprint 1 Plan](sprints/SPRINT_1_PLAN.md) - Implementation checklist and tasks
- ✅ [Sprint 1 Complete](sprints/SPRINT_1_COMPLETE.md) - Summary of accomplishments and metrics

### Future Sprints
- Sprint 2 - Game Rules & Mechanics (upcoming)
- Sprint 3 - Polish & Iteration (planned)

## Quick Links
- **Server Code**: `/server/main.go`
- **Client Code**: `/client/` (Godot project)
- **Test Client**: `/test_client.go`
- **Launch Script**: `/launch_client.sh`

## Organization Structure
```
.claude/
├── docs/
│   ├── README.md (this file)
│   ├── planning/
│   │   ├── GODOT_PLAN.md
│   │   └── Theme Ideas.md
│   └── sprints/
│       ├── SPRINT_1_PLAN.md
│       └── SPRINT_1_COMPLETE.md
└── settings.local.json
```