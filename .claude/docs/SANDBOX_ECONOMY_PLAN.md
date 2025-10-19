# Sandbox Economy System - Implementation Plan

**Created:** 2025-10-20
**Goal:** Transform the game into a sandbox economy with materials, construction, and worker operations
**Prerequisites:** Code reorganization + handoff documentation

---

## Table of Contents
1. [Phase 0: Code Reorganization & Documentation](#phase-0-code-reorganization--documentation)
2. [Phase 1: Resource System](#phase-1-resource-system)
3. [Phase 2: Construction System](#phase-2-construction-system)
4. [Phase 3: Worker Assignment & Operations](#phase-3-worker-assignment--operations)
5. [Phase 4: Production Loops](#phase-4-production-loops)
6. [Phase 5: UI & Polish](#phase-5-ui--polish)

---

## Phase 0: Code Reorganization & Documentation

**Why First:** Clean foundation for the user to take over server development

### 0.1 Server Handoff Documentation

**Create `.claude/docs/SERVER_HANDOFF.md`:**
- System overview diagram
- Server architecture (Quake 3 model, tick-based)
- Code organization (current + planned)
- Key systems guide (networking, pathfinding, formations, combat)
- How to add new features (step-by-step)
- Testing guide
- Common pitfalls & debugging tips
- Performance considerations

**Update Existing Docs:**
- `.claude/docs/ARCHITECTURE.md` - Mark outdated sections, add current state
- `.claude/docs/CURRENT_STATE.md` - Update with reorganization + new economy
- `.claude/docs/README.md` - Add link to SERVER_HANDOFF.md

**Files:**
- `.claude/docs/SERVER_HANDOFF.md` (NEW)
- `.claude/docs/ARCHITECTURE.md` (UPDATE)
- `.claude/docs/CURRENT_STATE.md` (UPDATE)

---

### 0.2 Server Code Reorganization

**Current:** Everything in `server/main.go` (1952 lines)

**Target Structure:**
```
server/
├── main.go                    # Entry point (~50 lines)
├── go.mod
├── game/
│   ├── server.go             # GameServer struct + Start()
│   ├── tick.go               # gameTick() + game loop
│   ├── commands.go           # Command handlers (move, build, attack, assign)
│   ├── buildings.go          # Building logic, construction, production
│   ├── resources.go          # Resource management (money, materials)
│   └── workers.go            # Worker assignment, construction work
├── movement/
│   ├── pathfinding.go        # A* algorithm
│   ├── formations.go         # Formation calculations
│   └── movement.go           # updateEntityMovement(), collision
├── network/
│   ├── protocol.go           # Message types + constants
│   ├── handlers.go           # Message handling
│   └── serialization.go      # JSON helpers
├── types/
│   ├── entity.go             # Entity, Client, Player structs
│   ├── message.go            # Message structs
│   └── map.go                # MapData, TerrainType, Feature
└── testutil/                  # (Already exists)
    ├── scenario.go
    ├── scenario_runner.go
    └── test_server.go
```

**Migration Strategy:**
1. Create package structure
2. Move types first (no dependencies)
3. Move network layer
4. Move game logic (depends on types + network)
5. Update tests
6. Verify `go test ./...` passes

**Estimated Time:** 2-4 hours

---

### 0.3 Client Code Reorganization

**Current:**
- `GameController.gd` (877 lines - too large!)
- `NetworkManager.gd` (good size)
- `Player.gd` (good size)

**Target Structure:**
```
client/
├── project.godot
├── Main.tscn
├── core/
│   ├── GameController.gd      # Core game loop, scene management (~200 lines)
│   ├── NetworkManager.gd      # (Keep as-is)
│   └── Constants.gd           # Shared constants (NEW)
├── input/
│   ├── InputHandler.gd        # Mouse/keyboard input (~150 lines)
│   └── SelectionManager.gd    # Unit/building selection (~150 lines)
├── rendering/
│   ├── IsometricRenderer.gd   # tile_to_iso, iso_to_tile (~50 lines)
│   ├── TerrainRenderer.gd     # Terrain tiles (~100 lines)
│   └── CameraController.gd    # Zoom, pan, bounds (~100 lines)
├── ui/
│   ├── UIManager.gd           # HUD updates (~150 lines)
│   ├── BuildMenu.gd           # Build UI (NEW for economy)
│   └── ResourceDisplay.gd     # Money + materials (NEW)
├── entities/
│   ├── Player.gd              # (Keep as-is)
│   ├── BuildingFactory.gd     # Create building visuals (~100 lines)
│   └── EntityManager.gd       # Entity lifecycle (~100 lines)
└── game/
    ├── FormationManager.gd    # Formation UI logic (~80 lines)
    └── EventLog.gd            # Event logging (~50 lines)
```

**Benefits:**
- Each file has single responsibility
- Easier to find code
- Easier to test individual systems
- Client development can continue independently

**Estimated Time:** 3-5 hours

---

## Phase 1: Resource System

### 1.1 Server: Dual Resource Model

**Add to `game/resources.go`:**
```go
type Resources struct {
    Money     float32
    Materials int32
    MaterialCap int32  // Storage limit
}

type Building struct {
    // ... existing fields
    AssignedWorkers []uint32  // Worker entity IDs
    ProductionRate  float32   // Materials/sec or $/sec
    ConsumptionRate float32   // Materials/sec (for shops)
}
```

**Constants:**
```go
const (
    StartingMoney     = 300.0
    StartingMaterials = 30    // Enough for 1 generator
    BaseMaterialCap   = 200
)
```

**Update `Client` struct:**
```go
type Client struct {
    // ... existing
    Resources Resources
}
```

**Update snapshots:**
```go
type PlayerSnapshot struct {
    Id        uint32
    Name      string
    Money     float32
    Materials int32     // NEW
    MaterialCap int32   // NEW
}
```

---

### 1.2 Client: Resource Display

**Create `client/ui/ResourceDisplay.gd`:**
```gdscript
extends Control

@onready var money_label = $MoneyLabel
@onready var materials_label = $MaterialsLabel

func update_resources(money: float, materials: int, material_cap: int):
    money_label.text = "$%.0f" % money
    materials_label.text = "Materials: %d / %d" % [materials, material_cap]
```

**Update UI layout:**
- Top-left corner: Money + Materials side-by-side
- Use icons ($ and cube icon)

---

## Phase 2: Construction System

### 2.1 Server: Building Definitions

**Create `game/buildings.go`:**
```go
type BuildingType string

const (
    BuildingHQ       BuildingType = "hq"
    BuildingGenerator BuildingType = "generator"
    BuildingShop      BuildingType = "shop"
    BuildingHousing   BuildingType = "housing"
    BuildingStorage   BuildingType = "storage"
)

type BuildingDefinition struct {
    Type            BuildingType
    CostMoney       float32
    CostMaterials   int32
    BuildTime       float32  // seconds
    WorkersRequired int      // to construct
    FootprintWidth  int
    FootprintHeight int
    MaxHealth       int32
    MaxWorkers      int      // for operation

    // Production (generators, shops)
    ProducesMaterials bool
    MaterialsPerSec   float32
    ProducesMoney     bool
    MoneyPerSec       float32
    ConsumesMaterials bool
    MaterialConsumptionPerSec float32
}

var BuildingDefs = map[BuildingType]BuildingDefinition{
    BuildingHQ: {
        Type: BuildingHQ,
        CostMoney: 0,  // Pre-placed
        CostMaterials: 0,
        FootprintWidth: 3,
        FootprintHeight: 3,
        MaxHealth: 500,
    },
    BuildingGenerator: {
        Type: BuildingGenerator,
        CostMoney: 50,
        CostMaterials: 20,
        BuildTime: 10.0,
        WorkersRequired: 2,
        FootprintWidth: 2,
        FootprintHeight: 2,
        MaxHealth: 100,
        MaxWorkers: 3,
        ProducesMaterials: true,
        MaterialsPerSec: 5.0,  // per worker
    },
    BuildingShop: {
        Type: BuildingShop,
        CostMoney: 100,
        CostMaterials: 50,
        BuildTime: 15.0,
        WorkersRequired: 2,
        FootprintWidth: 2,
        FootprintHeight: 2,
        MaxHealth: 100,
        MaxWorkers: 1,
        ConsumesMaterials: true,
        MaterialConsumptionPerSec: 10.0,
        ProducesMoney: true,
        MoneyPerSec: 20.0,
    },
    BuildingHousing: {
        Type: BuildingHousing,
        CostMoney: 80,
        CostMaterials: 30,
        BuildTime: 10.0,
        WorkersRequired: 1,
        FootprintWidth: 2,
        FootprintHeight: 2,
        MaxHealth: 80,
    },
    BuildingStorage: {
        Type: BuildingStorage,
        CostMoney: 40,
        CostMaterials: 20,
        BuildTime: 5.0,
        WorkersRequired: 1,
        FootprintWidth: 1,
        FootprintHeight: 1,
        MaxHealth: 50,
    },
}
```

---

### 2.2 Server: Construction State Machine

**Add to Entity:**
```go
type ConstructionState string

const (
    ConstructionNone       ConstructionState = ""
    ConstructionPlanned    ConstructionState = "planned"
    ConstructionInProgress ConstructionState = "building"
    ConstructionComplete   ConstructionState = "complete"
)

type Entity struct {
    // ... existing
    ConstructionState     ConstructionState
    ConstructionProgress  float32  // 0.0 to 1.0
    ConstructionWorkers   []uint32 // Worker IDs assigned to build
    OperatingWorkers      []uint32 // Worker IDs operating building (NEW)
}
```

**Construction Logic (`game/buildings.go`):**
```go
func (s *GameServer) tickConstruction(deltaTime float32) {
    for _, entity := range s.entities {
        if entity.ConstructionState != ConstructionInProgress {
            continue
        }

        def := BuildingDefs[BuildingType(entity.Type)]

        // Check workers present
        workersPresent := 0
        for _, workerID := range entity.ConstructionWorkers {
            worker := s.entities[workerID]
            if worker != nil && worker.TileX == entity.TileX && worker.TileY == entity.TileY {
                workersPresent++
            }
        }

        if workersPresent == 0 {
            continue  // No progress without workers
        }

        // Progress construction
        progressPerTick := (1.0 / def.BuildTime) * deltaTime
        entity.ConstructionProgress += progressPerTick

        if entity.ConstructionProgress >= 1.0 {
            entity.ConstructionState = ConstructionComplete
            entity.ConstructionProgress = 1.0
            entity.ConstructionWorkers = nil
            log.Printf("Building %d construction complete!", entity.Id)
        }
    }
}
```

---

### 2.3 Server: Build Command Update

**Update `handleBuildCommand()` in `game/commands.go`:**
```go
func (s *GameServer) handleBuildCommand(cmd Command, client *Client) {
    buildData := cmd.Data.(map[string]interface{})
    buildingType := BuildingType(buildData["buildingType"].(string))
    tileX := int(buildData["tileX"].(float64))
    tileY := int(buildData["tileY"].(float64))

    def := BuildingDefs[buildingType]

    // Validate resources
    if client.Resources.Money < def.CostMoney {
        return  // Not enough money
    }
    if client.Resources.Materials < def.CostMaterials {
        return  // Not enough materials
    }

    // Validate bounds + collision (existing logic)
    // ...

    // Deduct resources
    client.Resources.Money -= def.CostMoney
    client.Resources.Materials -= def.CostMaterials

    // Create building entity in "planned" state
    entityId := s.nextId
    s.nextId++

    building := &Entity{
        Id:                  entityId,
        OwnerId:             client.Id,
        Type:                string(buildingType),
        TileX:               tileX,
        TileY:               tileY,
        Health:              def.MaxHealth,
        MaxHealth:           def.MaxHealth,
        FootprintWidth:      def.FootprintWidth,
        FootprintHeight:     def.FootprintHeight,
        ConstructionState:   ConstructionPlanned,
        ConstructionProgress: 0.0,
    }

    s.entities[entityId] = building
    log.Printf("Client %d placed %s at (%d,%d) - awaiting workers",
        client.Id, buildingType, tileX, tileY)
}
```

---

### 2.4 New Command: Assign Workers to Construction

**Add to protocol:**
```go
type AssignConstructionCommand struct {
    BuildingId uint32   `json:"buildingId"`
    WorkerIds  []uint32 `json:"workerIds"`
}
```

**Handler:**
```go
func (s *GameServer) handleAssignConstructionCommand(cmd Command, client *Client) {
    data := cmd.Data.(map[string]interface{})
    buildingId := uint32(data["buildingId"].(float64))
    workerIdsInterface := data["workerIds"].([]interface{})

    building := s.entities[buildingId]
    if building == nil || building.OwnerId != client.Id {
        return
    }

    if building.ConstructionState != ConstructionPlanned {
        return  // Already building or complete
    }

    def := BuildingDefs[BuildingType(building.Type)]

    // Validate workers
    validWorkers := []uint32{}
    for _, wid := range workerIdsInterface {
        workerId := uint32(wid.(float64))
        worker := s.entities[workerId]
        if worker != nil && worker.OwnerId == client.Id && worker.Type == "worker" {
            validWorkers = append(validWorkers, workerId)
        }
    }

    if len(validWorkers) < def.WorkersRequired {
        return  // Not enough workers
    }

    // Assign workers and start construction
    building.ConstructionWorkers = validWorkers[:def.WorkersRequired]
    building.ConstructionState = ConstructionInProgress

    // Move workers to building site
    for _, workerId := range building.ConstructionWorkers {
        worker := s.entities[workerId]
        path := s.findPath(worker.TileX, worker.TileY, building.TileX, building.TileY, worker.Id)
        if len(path) > 0 {
            worker.Path = path
            worker.PathIndex = 0
        }
    }

    log.Printf("Construction started on building %d with %d workers", buildingId, len(building.ConstructionWorkers))
}
```

---

## Phase 3: Worker Assignment & Operations

### 3.1 Server: Worker Assignment to Buildings

**Add command:**
```go
type AssignWorkersCommand struct {
    BuildingId uint32   `json:"buildingId"`
    WorkerIds  []uint32 `json:"workerIds"`
}
```

**Handler (`game/workers.go`):**
```go
func (s *GameServer) handleAssignWorkersCommand(cmd Command, client *Client) {
    data := cmd.Data.(map[string]interface{})
    buildingId := uint32(data["buildingId"].(float64))
    workerIdsInterface := data["workerIds"].([]interface{})

    building := s.entities[buildingId]
    if building == nil || building.OwnerId != client.Id {
        return
    }

    if building.ConstructionState != ConstructionComplete {
        return  // Building not complete
    }

    def := BuildingDefs[BuildingType(building.Type)]
    if def.MaxWorkers == 0 {
        return  // This building doesn't need workers (e.g., Housing, Storage)
    }

    // Validate and assign workers
    validWorkers := []uint32{}
    for _, wid := range workerIdsInterface {
        workerId := uint32(wid.(float64))
        worker := s.entities[workerId]
        if worker != nil && worker.OwnerId == client.Id && worker.Type == "worker" {
            validWorkers = append(validWorkers, workerId)
            if len(validWorkers) >= def.MaxWorkers {
                break
            }
        }
    }

    building.OperatingWorkers = validWorkers

    // Move workers to building
    for _, workerId := range building.OperatingWorkers {
        worker := s.entities[workerId]
        path := s.findPath(worker.TileX, worker.TileY, building.TileX, building.TileY, worker.Id)
        if len(path) > 0 {
            worker.Path = path
            worker.PathIndex = 0
        }
    }

    log.Printf("Assigned %d workers to building %d", len(building.OperatingWorkers), buildingId)
}
```

---

### 3.2 Server: Production Tick

**Add to `gameTick()` in `game/tick.go`:**
```go
func (s *GameServer) gameTick() {
    // ... existing tick logic

    // Tick construction
    s.tickConstruction(deltaTime)

    // Tick production
    s.tickProduction(deltaTime)

    // ... rest of tick
}
```

**Production Logic (`game/buildings.go`):**
```go
func (s *GameServer) tickProduction(deltaTime float32) {
    for _, entity := range s.entities {
        if entity.ConstructionState != ConstructionComplete {
            continue
        }

        def := BuildingDefs[BuildingType(entity.Type)]
        client := s.clients[entity.OwnerId]
        if client == nil {
            continue
        }

        // Count workers present at building
        workersPresent := 0
        for _, workerID := range entity.OperatingWorkers {
            worker := s.entities[workerID]
            if worker != nil && worker.TileX == entity.TileX && worker.TileY == entity.TileY {
                workersPresent++
            }
        }

        // Generator: Produces materials (per worker)
        if def.ProducesMaterials && workersPresent > 0 {
            materialsProduced := int32(def.MaterialsPerSec * float32(workersPresent) * deltaTime)
            if client.Resources.Materials + materialsProduced <= client.Resources.MaterialCap {
                client.Resources.Materials += materialsProduced
            } else {
                client.Resources.Materials = client.Resources.MaterialCap
            }
        }

        // Shop: Consumes materials, produces money
        if def.ConsumesMaterials && def.ProducesMoney && workersPresent > 0 {
            materialsNeeded := int32(def.MaterialConsumptionPerSec * deltaTime)
            if client.Resources.Materials >= materialsNeeded {
                client.Resources.Materials -= materialsNeeded
                client.Resources.Money += def.MoneyPerSec * deltaTime
            }
            // If not enough materials, shop doesn't produce
        }
    }
}
```

---

## Phase 4: Starting State & HQ

### 4.1 Server: Spawn HQ on Connect

**Update `handleHello()` in `network/handlers.go`:**
```go
func (s *GameServer) handleHello(hello HelloMessage, clientAddr *net.UDPAddr) {
    // ... existing client creation

    // Spawn HQ building
    hqId := s.nextId
    s.nextId++

    def := BuildingDefs[BuildingHQ]

    hq := &Entity{
        Id:                hqId,
        OwnerId:           clientId,
        Type:              string(BuildingHQ),
        TileX:             spawnBaseTileX,
        TileY:             spawnBaseTileY,
        Health:            def.MaxHealth,
        MaxHealth:         def.MaxHealth,
        FootprintWidth:    def.FootprintWidth,
        FootprintHeight:   def.FootprintHeight,
        ConstructionState: ConstructionComplete,
        ConstructionProgress: 1.0,
    }

    s.entities[hqId] = hq

    // Spawn workers near HQ (offset by HQ footprint)
    workerSpawnX := spawnBaseTileX + def.FootprintWidth + 1
    // ... rest of worker spawning

    client.Resources = Resources{
        Money: StartingMoney,       // $300
        Materials: StartingMaterials, // 30
        MaterialCap: BaseMaterialCap, // 200
    }
}
```

---

## Phase 5: Client UI

### 5.1 Build Menu

**Create `client/ui/BuildMenu.gd`:**
```gdscript
extends VBoxContainer

signal build_requested(building_type: String)

@onready var generator_button = $GeneratorButton
@onready var shop_button = $ShopButton
@onready var housing_button = $HousingButton
@onready var storage_button = $StorageButton

var player_money: float = 0.0
var player_materials: int = 0

var building_costs = {
    "generator": {"money": 50, "materials": 20},
    "shop": {"money": 100, "materials": 50},
    "housing": {"money": 80, "materials": 30},
    "storage": {"money": 40, "materials": 20}
}

func _ready():
    generator_button.pressed.connect(_on_build_pressed.bind("generator"))
    shop_button.pressed.connect(_on_build_pressed.bind("shop"))
    housing_button.pressed.connect(_on_build_pressed.bind("housing"))
    storage_button.pressed.connect(_on_build_pressed.bind("storage"))
    update_buttons()

func update_resources(money: float, materials: int):
    player_money = money
    player_materials = materials
    update_buttons()

func update_buttons():
    for building_type in building_costs:
        var button = get_node(building_type.capitalize() + "Button")
        var cost = building_costs[building_type]
        var can_afford = player_money >= cost.money and player_materials >= cost.materials

        button.disabled = not can_afford
        button.text = "%s\n$%d  %dM" % [building_type.capitalize(), cost.money, cost.materials]

        if not can_afford:
            button.modulate = Color(0.5, 0.5, 0.5)
        else:
            button.modulate = Color(1, 1, 1)

func _on_build_pressed(building_type: String):
    build_requested.emit(building_type)
```

---

### 5.2 Worker Assignment UI

**When clicking a building:**
- Show building panel with:
  - Building name
  - Health bar
  - Construction progress (if building)
  - Assigned workers (if operational)
  - "Assign Workers" button
  - "Unassign Workers" button

**Implementation:**
- Detect building click (already exists)
- Show side panel with building info
- "Assign Workers" button:
  - Selects building
  - Next unit selection assigns workers to building
  - Send `assignWorkers` command

---

## Testing Strategy

### Phase 0 Testing
- [ ] Server compiles after reorganization
- [ ] All existing tests pass (`go test ./...`)
- [ ] Client loads after reorganization
- [ ] Existing gameplay still works

### Phase 1 Testing
- [ ] Resources display correctly
- [ ] Snapshots include materials

### Phase 2 Testing
- [ ] Buildings can be placed (deducts resources)
- [ ] Buildings show construction progress
- [ ] Workers can be assigned to construction
- [ ] Construction completes after time + workers

### Phase 3 Testing
- [ ] Workers can be assigned to generators
- [ ] Generators produce materials when workers present
- [ ] Shops consume materials and produce money
- [ ] No production when workers not present

### Phase 4 Testing
- [ ] HQ spawns on connect
- [ ] Players start with correct resources
- [ ] Full production chain: Generator → Materials → Shop → Money

---

## Estimated Timeline

| Phase | Task | Est. Time |
|-------|------|-----------|
| 0.1 | Server handoff docs | 2-3 hours |
| 0.2 | Server reorganization | 3-4 hours |
| 0.3 | Client reorganization | 3-5 hours |
| 1.1-1.2 | Resource system | 2-3 hours |
| 2.1-2.4 | Construction system | 4-5 hours |
| 3.1-3.2 | Worker assignment + production | 3-4 hours |
| 4.1 | Starting state | 1 hour |
| 5.1-5.2 | Client UI | 3-4 hours |
| **Total** | | **21-31 hours** |

**Recommended Approach:** Complete Phase 0 first, then user takes over Phase 1-5 with occasional assistance.

---

## Success Criteria

### Phase 0 Complete When:
- ✅ Server code split into logical packages
- ✅ Client code split into modules
- ✅ All tests still passing
- ✅ Comprehensive handoff documentation exists
- ✅ User can understand server architecture from docs

### Full System Complete When:
- ✅ Player starts with HQ, $300, 30 materials, 5 workers
- ✅ Can place generator ($50 + 20M), assign 2 workers to build
- ✅ Workers walk to construction site, building progresses
- ✅ After 10 seconds, generator completes
- ✅ Can assign workers to generator, materials accumulate
- ✅ Can build shop ($100 + 50M), assign worker
- ✅ Shop consumes materials, produces money
- ✅ Full economic loop functions
- ✅ UI clearly shows money, materials, build options
- ✅ Build menu grays out unaffordable buildings

---

## Next Steps

1. **Start with Phase 0.1** - Create SERVER_HANDOFF.md (I can help)
2. **Then Phase 0.2** - Reorganize server (I can do this)
3. **Then Phase 0.3** - Reorganize client (I can do this)
4. **Handoff** - User reviews handoff docs
5. **Phases 1-5** - User implements with my assistance as needed

**Ready to begin?** Let me know and I'll start with the handoff documentation.
