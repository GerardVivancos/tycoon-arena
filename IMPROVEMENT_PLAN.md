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
check win conditions each tick, broadcast `game_over`, client UI). At Phase 1
time the real JSON modes load/display but only became *winnable* in Phase 2 —
the rule engine could not yet resolve `player:"self"/"any"` per player, so lose
conditions were **deliberately not evaluated** in the live loop (self-referencing
rules would false-positive at tick 1). Plumbing was proven with a purpose-built
test mode. **(Phase 2 has since enabled per-player resolution and live lose
evaluation — see below.)**

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
Shipped modes are now genuinely winnable/losable in the live loop.

- [x] **2.1** `rule_engine.go`: `RuleContext` gains `CurrentPlayer` + `Players`;
      `StatValue` gains `PlayerRef` and `Resolve` handles `"self"` /
      `"opponent"` (max excluding self) / `"any"` (aggregate max) / numeric id /
      `""` (legacy direct id). `ParseValue` maps the JSON `player` selector and
      rejects unknown selectors. ✅ (2025-10-09)
- [x] **2.2** `game_mode.go`: removed the map-iteration `determineWinner` stub.
      `CheckWinConditions` now honors `condition.Winner` — `player_who_triggered`
      iterates sorted players (first self-rule pass wins), `highest_money` /
      `highest_stat` evaluate the rule once then pick the max-holder via
      `playerWithHighestStat` (ties → lowest id, deterministic). ✅ (2025-10-09)
- [x] **2.3** `executePeriodicCheck` implements `grant_resources` (owner building
      count × `amount_per_building`, via new `grantMoneyToPlayer` + `countBuildings`
      helpers) and `increment_global_stat`; `gameTick()` now calls
      `mode.OnTick(ctx)` after unlock (adjacent to `checkMatchEnd`). ✅ (2025-10-09)
- [x] **2.4** `CheckLoseConditions(ctx, playerID)` sets `CurrentPlayer` so
      self-referencing rules work; `checkMatchEnd` evaluates lose conditions and
      crowns the lone survivor (`last_player_standing`), skipping solo matches.
      `handleHello` seeds `money_current` so bankruptcy can't false-positive. ✅ (2025-10-09)
- [x] **2.5** JSON modes: per-player win rules moved from `"any"` to `"self"`
      (`resource_race` money, `survival` buildings); removed the redundant
      `grant_resources` periodic checks from all three modes (core loop already
      pays `$10/s`/generator — double-count avoided). `timed_race` keeps its
      `increment_global_stat` tick counter. ✅ (2025-10-09)
- [x] **2.6** Tests (+6, 62 green): `TestStatValuePerPlayerResolution`
      (self/opponent/any/id); rewrote `TestGameModeWinCondition` /
      `TestSurvivalGameMode` / `TestTimedRaceGameMode` for real clients +
      per-player stats; added `TestLoseConditionLastPlayerStanding`,
      `TestLoseConditionFreshPlayerNotBroke`, `TestPeriodicGrantResources`,
      `TestPeriodicIncrementGlobalStat`, `TestResourceRaceEndToEnd` (one
      `gameTick()` ends the match). ✅ (2025-10-09)

**Verify:** `gofmt -l .` empty, `go vet ./...` clean, `go test -count=1 ./...`
62/62 green (2025-10-09).

---

## Phase 3 — Combat expansion

**Objective:** Support unit combat and complete the statistics they feed.
Decisions: directed attack-move with range/cooldown (1A), attack command carries
`unitIds` (2A), per-type combat table as server constants (3A), buildings are
non-combatant, plus a declarative scenario.

- [x] **3.1** `handleAttackCommand`: now *assigns* a target (`AttackTargetId`) to
      the named `unitIds` (or the client's own units) instead of applying one-shot
      damage; validates attackable types (`worker`/`player`/`generator`) and enemy
      ownership. New `processCombat()` (in `gameTick`, after formations) moves
      attackers into range (`pathUnitToAttackRange`/`nearestAttackTile`) and applies
      damage via `applyDamage()` on a per-type cooldown (`unitCombatStats`:
      worker dmg 25 / range 1 / interval 20). `handleMoveCommand` clears
      `AttackTargetId`; `Entity` gains `AttackTargetId`/`NextAttackTick`. ✅ (2025-10-09)
- [x] **3.2** `statistics.go`: `damage_taken` + `attacks_received` now land on the
      target's owner (carried as `targetOwnerId` in `entity.damaged`/`unit.attacked`);
      `units_killed`/`units_lost` fire for unit deaths via `entity.destroyed`. ✅ (2025-10-09)
- [x] **3.3** `NETWORK_PROTOCOL.md`: attack `unitIds` + combat semantics; constants
      table gains AttackRange/AttackInterval; validation rules updated. ✅ (2025-10-09)
- [x] **3.4** Tests: `combat_test.go` (+7: kill, chase, cooldown, out-of-range,
      non-combatant building, no self-attack, destroy building) and new scenario
      `maps/scenarios/unit_combat.json`. Test-adapter fixes: `Tick()` now runs
      `processCombat()`, `AttackTarget` sends `unitIds`, and destroyed buildings are
      recorded so `exists:false` is verifiable. ✅ (2025-10-09)

**Verify:** `gofmt -l .` empty, `go vet ./...` clean, `go test -count=1 ./...`
69/69 green (2025-10-09).

**Follow-up (not in scope):** auto-acquire/defensive AI, ranged unit types,
building attackers, and a "no units remaining" lose rule for `survival`.

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
- [x] Phase 2 — rule engine fixes (done 2025-10-09: per-player rule resolution
      self/opponent/any/id; winner strategies player_who_triggered/highest_money/
      highest_stat; periodic grant_resources + increment_global_stat wired into
      tick; live lose conditions + lone-survivor; modes winnable/losable; +6 tests,
      62 green)
- [x] Phase 3 — combat expansion (done 2025-10-09: unit-vs-unit + unit-vs-building
      attack-move with range/cooldown via processCombat; per-type combat table;
      damage_taken/attacks_received/units_killed/units_lost wired; attack `unitIds`;
      client sends unitIds + enemy-unit targeting; +7 tests + unit_combat scenario,
      69 green. Note: harness fixed to record destroyed buildings & run combat on tick)
- [ ] Phase 4 — concurrency hardening
