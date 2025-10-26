Repository Guidelines
=====================

Project Structure & Module Organization
---------------------------------------
- `server/`: Go authoritative server (`main.go`, unit + scenario tests under `server/`).
- `client/`: Godot 4 project (`GameController.gd`, scenes, networking scripts).
- `maps/`: JSON maps and declarative scenarios (`maps/scenarios/` for automated tests).
- Root scripts: `launch_all.sh`, `launch_client.sh`, `test_client.go` (CLI sample client).
- Documentation: `.claude/docs/` for design notes; `Claude.md`, `TESTING.md` for quick reference.

Build, Test, and Development Commands
-------------------------------------
- `cd server && go run main.go`: Launch the server locally.
- `./launch_all.sh 2`: Start server plus two Godot clients (requires macOS Godot path).
- `cd server && go test -v`: Run all Go unit and scenario tests.
- `cd client && /Applications/Godot_mono.app/.../Godot --path .`: Open client in the editor.
- `go run test_client.go`: Exercise the JSON protocol with the CLI sample.

Coding Style & Naming Conventions
---------------------------------
- Go: run `gofmt`; prefer camelCase for locals, PascalCase for exported symbols; keep files under 2k lines when feasible.
- GDScript: follow Godot 4 defaults (tabs = 4 spaces), snake_case for variables, PascalCase for classes.
- JSON maps/scenarios: indent with two spaces; keep keys lowercase with camelCase values.
- Log messages should include entity IDs or client IDs for traceability.

Testing Guidelines
------------------
- Unit tests live in `server/game_test.go`; new tests should start with `Test` and group by feature.
- Scenario tests reside in `server/scenario_test.go` and auto-discover JSON in `maps/scenarios/`.
- When adding scenarios, include a short description in the JSON and regenerate SVG visuals if relevant.
- Run `go test -v` before submitting; add targeted `go test -run <Name>` commands to PR notes for focused changes.

Commit & Pull Request Guidelines
--------------------------------
- Use conventional-style messages where practical (`feat:`, `fix:`, `docs:`); reference modules (`server`, `client`, `maps`).
- Keep commits scoped to one intent; accompany each intent with `change.md` (not checked in) documenting rationale for reviewers.
- PRs should summarize gameplay impact, testing performed, and include screenshots or GIFs if the client UI changes.
- Link to relevant docs in `.claude/docs/` when implementation diverges from recorded plans; flag follow-up work with TODOs.
