# Sprint 1 Implementation Plan - Networking Core

## Current Status
- ✅ Go server with basic UDP networking, tick loop, and client handling exists
- ✅ Test Go client can connect and send commands
- ⚠️ Godot project initialized but no scenes/scripts yet

## Implementation Tasks

### 1. Server Improvements (server/main.go)
- [ ] Add delta movement instead of absolute positioning
- [ ] Implement proper movement speed and validation
- [ ] Add arena bounds checking
- [ ] Improve client timeout handling
- [ ] Add structured logging for debugging

### 2. Godot Client - Core Structure
**Main.tscn**
- [ ] Create main scene with game viewport
- [ ] Add camera setup
- [ ] Create UI layer for HUD
- [ ] Add entity container node

### 3. Godot Client - Networking
**NetworkManager.gd**
- [ ] Implement UDP connection handling
- [ ] Create JSON message serialization/deserialization
- [ ] Add message queue system
- [ ] Implement connection state management
- [ ] Handle hello/welcome handshake

### 4. Godot Client - Entities
**Player.tscn + Player.gd**
- [ ] Create simple colored box sprite (32x32 pixels)
- [ ] Implement position interpolation
- [ ] Add health bar display
- [ ] Add owner indicator (different color for local player)

### 5. Godot Client - Game Logic
**GameController.gd**
- [ ] Handle WASD/arrow key input
- [ ] Send input commands to server at fixed rate
- [ ] Process server snapshots
- [ ] Implement client-side prediction
- [ ] Add reconciliation when server updates arrive
- [ ] Spawn/despawn entities based on snapshots

### 6. Godot Client - UI
**UI.tscn**
- [ ] Connection status indicator
- [ ] FPS counter
- [ ] Ping display
- [ ] Connected players list
- [ ] Debug info toggle (tick, position, etc.)

### 7. Testing Checklist
- [ ] Server runs and accepts connections
- [ ] Single client can connect and move
- [ ] Two clients can see each other
- [ ] Movement is smooth with prediction
- [ ] Disconnection handled gracefully
- [ ] Reconnection works

## Next Steps After Sprint 1
- Sprint 2: Add building mechanics and interference actions
- Sprint 3: Polish, metrics, and theme exploration

## Technical Notes
- Using JSON for protocol (rapid prototyping)
- UDP with reliability layer for important events
- Fixed tick rate: 20Hz
- Client-side prediction with server reconciliation
- Target: <10KB/s bandwidth per client