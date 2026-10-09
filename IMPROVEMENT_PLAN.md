# Improvement Plan

**Created:** 2025-10-09
**Goal:** Address the issues identified in the project review, each phase independently
verifiable, worked in sequence. Every phase ends with a green `cd server && go test -v`
and an updated context doc (`.claude/docs/CURRENT_STATE.md`) per `CLAUDE.md`.

---

## Phase 0 — Quality & hygiene (START HERE)

**Objective:** Establish a clean, green baseline before adding features.

- [x] **0.1** Run `gofmt` on all non-compliant files:
      `event_bus.go`, `game_mode.go`, `statistics.go`,
      `testutil/scenario.go`, `testutil/scenario_runner.go`, `testutil/test_server.go`.
      Verify with `gofmt -l .` → empty output. ✅ (2025-10-09)
- [x] **0.2** Replace `interface{}` with `any` across `server/` for consistency
      (`main.go`, `rule_engine.go`, `event_bus.go`, `statistics.go`, `game_mode.go`).
      ✅ (2025-10-09)
- [x] **0.3** Remove the dead empty directory `server/cmd/maps/`. ✅ (2025-10-09)
- [x] **0.4** Refresh stale docs to match code: ✅ (2025-10-09)
      - `.claude/docs/CURRENT_STATE.md` — add event bus / game modes / rule engine /
        statistics systems; correct test count (51); remove `server/cmd/scenario-viz`
        and `server/cmd/maps` references.
      - `Claude.md` — fix "12 tests" → actual count; note new subsystems.
      - `TESTING.md` — add new test files/counts and the mode/rule/statistics tests.
      - `.claude/docs/README.md` — index update.

**Verify:** `gofmt -l .` empty, `go vet ./...` clean, `go test -v` all pass.

---

## Phase 1 — Wire game modes into the running game (PLUMBING ONLY — approved 2025-10-09)

**Objective:** Wire the game-mode layer into the live loop end-to-end (load mode,
check win conditions each tick, broadcast `game_over`, client UI). The real JSON
modes load/display but only become *winnable* in Phase 2 — the rule engine cannot
yet resolve `player:"self"/"any"` per player. Until then, lose conditions are
**deliberately not evaluated** in the live loop (self-referencing rules like
`money_current <= 0` resolve against player 0's empty stats and would
false-positive at tick 1). Plumbing is proven with a purpose-built test mode.

- [x] **1.1** `GameServer`: add `gameMode`, `gameModeDef`, `gameOver`, `winnerId`
      fields; `setGameMode()`; `buildMatchContext()`; `main()` loads
      `game_modes/resource_race.json` (default, approved) and fires
      `OnMatchCreated/OnMatchStarting/OnMatchStarted`. ✅ (2025-10-09)
- [x] **1.2** `gameTick()` calls `s.checkMatchEnd()` **after** unlocking (the rule
      engine + `determineWinner` take `s.mu` internally → deadlock if called under
      lock). `endMatch()` is idempotent: sets `gameOver`/`winnerId`, fires
      `OnMatchEnding`/`OnMatchEnded`, emits `match.ended`, broadcasts `game_over`.
      Guards: `handleHello` rejects joins, `handleInput` drops, `processCommand`
      (and the tick input loop) ignore commands once `gameOver`.
      Lose-condition evaluation deferred to Phase 2 (see above). ✅ (2025-10-09)
- [x] **1.3** Add `MsgGameOver` + `GameOverMessage{winnerId, conditionId,
      conditionName, objectiveText, victoryMessage, defeatMessage}`; extend
      `WelcomeMessage` with `gameModeName`, `objectiveText`, `showTimer`,
      `showMoney` (approved delivery mechanism); update `NETWORK_PROTOCOL.md`. ✅ (2025-10-09)
- [x] **1.4** Client: `NetworkManager.gd` handles `game_over` (new
      `game_over_received` + `mode_info_received` signals); `GameController.gd`
      shows objective on connect, victory/defeat on end, disables
      move/build/attack when `match_over`; `Main.tscn` gains a `GameStatus` label. ✅ (2025-10-09, headless --check-only parse clean)
- [x] **1.5** Tests (`server/game_mode_plumbing_test.go`): `TestGameServerGameModePlumbing`
      (purpose-built mode winning on `match_duration_ticks`), `TestCheckMatchEndNoMode`,
      `TestGameOverGuardsCommands`, `TestBuildGameOverMessage`, `TestEndMatchIdempotent`;
      `broadcastMessage` gains a nil-conn guard for unstarted test servers. ✅ (2025-10-09)

**Verify:** `go test -v` green; objective text visible in client; a test-mode win
ends the match (manual Godot check left to the user per project convention).
**Risk:** event ordering / locking — full `Emit`-outside-lock refactor deferred to Phase 4.

---

## Phase 2 — Fix rule engine & game-mode logic gaps

**Objective:** Correct the stubs so declarative rules behave as documented.

- [ ] **2.1** `rule_engine.go`: resolve `player: "self" / "opponent" / "all" / <id>`
      in `StatValue.Resolve` and `ParseValue` so per-player win/lose rules work.
- [ ] **2.2** `game_mode.determineWinner` (`game_mode.go:257`): honor `condition.Winner`
      (`player_who_triggered`, `highest_money`, `highest_stat`) and select deterministically
      (no map-iteration ordering).
- [ ] **2.3** `executePeriodicCheck` (`game_mode.go:271`): implement `grant_resources` and
      `increment_global_stat` actions; wire the `match_duration_ticks` counter.
- [ ] **2.4** Make `CheckLoseConditions` and `GetObjectiveText` player-aware.

**Verify:** extend `rule_engine_test.go` + `game_mode_test.go` with per-player and
periodic-check cases; `go test -v` all pass.

---

## Phase 3 — Combat expansion

**Objective:** Support unit combat and complete the statistics they feed.

- [ ] **3.1** `handleAttackCommand` (`main.go:1941`): allow attacking units (not just
      `generator`), with range/HP handling and unit-vs-unit targeting.
- [ ] **3.2** `statistics.go`: populate `damage_taken`, `attacks_received`, and
      `units_killed` / `units_lost` in event handlers.
- [ ] **3.3** Update `NETWORK_PROTOCOL.md` (attackable types) and the combat stats docs.

**Verify:** new unit tests for unit combat + stat assertions; `go test -v` all pass.

---

## Phase 4 — Concurrency & correctness hardening

**Objective:** Prevent deadlocks/races as the mode layer grows.

- [ ] **4.1** Avoid `eventBus.Emit` while holding `s.mu` (used at `main.go:425-449`,
      `1878`, `1912`, etc.); document event ordering or queue events for after-unlock.
- [ ] **4.2** Run `go test -race ./...`; fix any races; add a targeted test for the
      win-condition path under concurrent load.

**Verify:** `go vet ./...`, `go test -race ./...` clean.

---

## Out of scope (deferred)
- Phase 5 optional features: delta compression (`baselineTick`/`LastAckTick`), fog of war,
  unit balancing, visual scenario editor.

## Progress tracker
- [x] Review complete (2025-10-09)
- [x] Phase 0 — hygiene (done 2025-10-09: gofmt, `any`, dead code, docs; 51 tests green)
- [x] Phase 1 — wire game modes (done 2025-10-09: mode loaded in main(), win check in
      tick loop, `game_over` protocol + client HUD, 5 new tests; 56 tests green.
      Note: shipped JSON modes fire only after Phase 2 per-player rule resolution)
- [ ] Phase 2 — rule engine fixes
- [ ] Phase 3 — combat expansion
- [ ] Phase 4 — concurrency hardening
