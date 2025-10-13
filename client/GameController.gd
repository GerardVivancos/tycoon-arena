extends Node2D

@onready var network_manager = $NetworkManager
@onready var entities_container = $Entities
@onready var camera = $Camera2D
@onready var connection_label = $UI/ConnectionStatus
@onready var fps_label = $UI/FPS
@onready var player_list_label = $UI/PlayerList
@onready var money_label = $UI/MoneyLabel
@onready var build_button = $UI/BuildButton
@onready var attack_button = $UI/AttackButton
@onready var selection_label = $UI/SelectionLabel
@onready var formation_label = $UI/FormationLabel
@onready var box_formation_button = $UI/BoxFormationButton
@onready var line_formation_button = $UI/LineFormationButton
@onready var spread_formation_button = $UI/SpreadFormationButton
@onready var event_log = $UI/EventLog

# Tile system (from server via handshake)
var tile_size: int
var arena_tiles_width: int
var arena_tiles_height: int

# Rendering - Isometric
const PIXELS_PER_TILE = 32  # Visual size of each tile (for top-down, kept for reference)
const ISO_TILE_WIDTH = 64   # Width of isometric diamond
const ISO_TILE_HEIGHT = 32  # Height of isometric diamond
const ISO_OFFSET_X = 640    # Screen offset to center the map (adjusted for larger maps)
const ISO_OFFSET_Y = 200    # Adjusted for 40×30 map

var player_scene = preload("res://Player.tscn")
var entities: Dictionary = {}  # entity_id -> unit/building node
var local_client_id: int = -1
var local_money: float = 0.0
var players_data: Dictionary = {}
var selected_units: Array[int] = []  # Entity IDs of selected units
var selected_target_id: int = -1  # Target for attacks (buildings)
var selected_building: Node2D = null
var event_messages: Array = []

# Drag selection
var is_dragging: bool = false
var drag_start_pos: Vector2 = Vector2.ZERO
var drag_current_pos: Vector2 = Vector2.ZERO
const DRAG_THRESHOLD: float = 5.0  # Minimum pixels to count as drag vs click

# Formation system
var current_formation: String = "box"  # Options: "box", "line", "spread"

# Camera system
var camera_zoom_min: float = 0.5
var camera_zoom_max: float = 2.0
var camera_zoom_step: float = 0.1
var camera_pan_speed: float = 500.0  # pixels per second
var camera_bounds: Rect2  # Set after map loads

func _ready():
	# Connect network signals
	network_manager.connected_to_server.connect(_on_connected_to_server)
	network_manager.snapshot_received.connect(_on_snapshot_received)
	network_manager.disconnected_from_server.connect(_on_disconnected_from_server)

	# Connect UI signals
	build_button.pressed.connect(_on_build_button_pressed)
	attack_button.pressed.connect(_on_attack_button_pressed)
	box_formation_button.pressed.connect(func(): set_formation("box"))
	line_formation_button.pressed.connect(func(): set_formation("line"))
	spread_formation_button.pressed.connect(func(): set_formation("spread"))

	# Initialize formation display (box is default)
	box_formation_button.text = "► Box (1)"
	line_formation_button.text = "Line (2)"
	spread_formation_button.text = "Spread (3)"

	# Initialize camera
	camera.zoom = Vector2(1.0, 1.0)

	# Auto-connect on start
	network_manager.connect_to_server("Player" + str(randi() % 1000))

func _on_connected_to_server(client_id: int, tick_rate: int, tile_sz: int, tiles_w: int, tiles_h: int):
	local_client_id = client_id
	tile_size = tile_sz
	arena_tiles_width = tiles_w
	arena_tiles_height = tiles_h
	connection_label.text = "Connected (ID: %d)" % client_id
	print("Connected with client ID: %d, Arena: %dx%d tiles" % [client_id, tiles_w, tiles_h])

	# Calculate camera bounds based on map size in isometric space
	# Isometric maps form a diamond, so we need all 4 corners
	var north = tile_to_iso(0, 0)                     # Top corner
	var east = tile_to_iso(float(tiles_w), 0)        # Right corner
	var south = tile_to_iso(float(tiles_w), float(tiles_h))  # Bottom corner
	var west = tile_to_iso(0, float(tiles_h))        # Left corner

	# Find actual bounding box of the diamond
	var min_x = min(north.x, min(east.x, min(south.x, west.x)))
	var max_x = max(north.x, max(east.x, max(south.x, west.x)))
	var min_y = min(north.y, min(east.y, min(south.y, west.y)))
	var max_y = max(north.y, max(east.y, max(south.y, west.y)))

	# Get actual viewport size (handles resizable window)
	var viewport_size = get_viewport().get_visible_rect().size
	var viewport_width = viewport_size.x
	var viewport_height = viewport_size.y
	var half_viewport_w = viewport_width / 2.0
	var half_viewport_h = viewport_height / 2.0

	# Camera bounds: camera center can move so edges of map align with viewport edges
	camera_bounds = Rect2(
		min_x + half_viewport_w,
		min_y + half_viewport_h,
		(max_x - min_x) - viewport_width,
		(max_y - min_y) - viewport_height
	)

	queue_redraw()  # Trigger grid drawing

# Convert tile coordinates to isometric screen position
func tile_to_iso(tile_x: float, tile_y: float) -> Vector2:
	# Isometric projection:
	# iso_x = (tile_x - tile_y) * half_width
	# iso_y = (tile_x + tile_y) * half_height
	var iso_x = (tile_x - tile_y) * (ISO_TILE_WIDTH / 2.0)
	var iso_y = (tile_x + tile_y) * (ISO_TILE_HEIGHT / 2.0)
	return Vector2(iso_x + ISO_OFFSET_X, iso_y + ISO_OFFSET_Y)

# Convert isometric screen position to tile coordinates
func iso_to_tile(screen_pos: Vector2) -> Vector2i:
	# Remove offset
	var dx = screen_pos.x - ISO_OFFSET_X
	var dy = screen_pos.y - ISO_OFFSET_Y

	# Inverse isometric projection
	# From: iso_x = (tx - ty) * w/2, iso_y = (tx + ty) * h/2
	# Solve: tx = (dx/(w/2) + dy/(h/2)) / 2
	#        ty = (dy/(h/2) - dx/(w/2)) / 2
	var tile_x = (dx / (ISO_TILE_WIDTH / 2.0) + dy / (ISO_TILE_HEIGHT / 2.0)) / 2.0
	var tile_y = (dy / (ISO_TILE_HEIGHT / 2.0) - dx / (ISO_TILE_WIDTH / 2.0)) / 2.0

	# Use floor instead of round - tile (x,y) owns all points where x <= tile_x < x+1
	return Vector2i(int(floor(tile_x)), int(floor(tile_y)))

# Camera control functions
func zoom_camera(delta_zoom: float):
	var new_zoom = camera.zoom.x + delta_zoom
	new_zoom = clamp(new_zoom, camera_zoom_min, camera_zoom_max)
	camera.zoom = Vector2(new_zoom, new_zoom)

	# After zoom, re-clamp camera position to adjusted bounds
	pan_camera(Vector2.ZERO)

func pan_camera(offset: Vector2):
	camera.position += offset / camera.zoom.x  # Adjust for zoom level

	# Clamp to map bounds (if bounds are set), accounting for current zoom level
	if camera_bounds.has_area():
		# Get actual viewport size (handles resizable window)
		var viewport_size = get_viewport().get_visible_rect().size
		var viewport_width = viewport_size.x
		var viewport_height = viewport_size.y

		# Effective viewport size in world coordinates changes with zoom
		var effective_half_w = (viewport_width / 2.0) / camera.zoom.x
		var effective_half_h = (viewport_height / 2.0) / camera.zoom.x

		# Allow 20% padding from viewport edges (keep map edge at least 20% from window edge)
		var edge_padding_x = viewport_width * 0.20  # 20% of viewport width
		var edge_padding_y = viewport_height * 0.20  # 20% of viewport height

		# Recalculate bounds for current zoom
		var half_w = viewport_width / 2.0
		var half_h = viewport_height / 2.0
		var min_x = camera_bounds.position.x - (half_w - effective_half_w) - edge_padding_x
		var max_x = camera_bounds.end.x + (half_w - effective_half_w) + edge_padding_x
		var min_y = camera_bounds.position.y - (half_h - effective_half_h) - edge_padding_y
		var max_y = camera_bounds.end.y + (half_h - effective_half_h) + edge_padding_y

		camera.position.x = clamp(camera.position.x, min_x, max_x)
		camera.position.y = clamp(camera.position.y, min_y, max_y)

func _on_snapshot_received(snapshot: Dictionary):
	var entities_data = snapshot.get("entities", [])
	players_data = snapshot.get("players", {})

	# Update local money
	if str(local_client_id) in players_data:
		var player_data = players_data[str(local_client_id)]
		local_money = player_data.get("money", 0.0)
		money_label.text = "Money: $%.0f" % local_money

	# Track which entities are in the snapshot
	var current_entity_ids = {}

	for entity_data in entities_data:
		var entity_id = int(entity_data.get("id", -1))
		var owner_id = int(entity_data.get("ownerId", -1))
		var entity_type = entity_data.get("type", "")
		var tile_x = int(entity_data.get("tileX", 0))
		var tile_y = int(entity_data.get("tileY", 0))
		var target_tile_x = int(entity_data.get("targetTileX", 0))
		var target_tile_y = int(entity_data.get("targetTileY", 0))
		var move_progress = float(entity_data.get("moveProgress", 0.0))
		var health = int(entity_data.get("health", 100))
		var max_health = int(entity_data.get("maxHealth", 100))
		var footprint_width = int(entity_data.get("footprintWidth", 0))
		var footprint_height = int(entity_data.get("footprintHeight", 0))

		current_entity_ids[entity_id] = true

		# Calculate interpolated tile position
		var interp_tile_x = lerp(float(tile_x), float(target_tile_x), move_progress)
		var interp_tile_y = lerp(float(tile_y), float(target_tile_y), move_progress)

		# Convert to isometric screen position
		var screen_pos = tile_to_iso(interp_tile_x, interp_tile_y)

		if entity_type == "player" or entity_type == "worker":
			if entity_id in entities:
				# Update existing unit
				var unit = entities[entity_id]
				unit.update_from_snapshot(screen_pos, health, max_health)
			else:
				# Create new unit
				var unit = player_scene.instantiate()
				var is_local = (owner_id == local_client_id)
				unit.setup(entity_id, owner_id, screen_pos, is_local)

				if is_local:
					unit.set_player_name("Worker")
				else:
					unit.set_player_name("Enemy")

				entities_container.add_child(unit)
				entities[entity_id] = unit
				print("Spawned %s entity %d at tile (%d, %d)" % [entity_type, entity_id, tile_x, tile_y])

		elif entity_type == "generator":
			if entity_id in entities:
				# Update existing building
				var building = entities[entity_id]
				update_building_health(building, health, max_health)
			else:
				# Create new building at tile corner in isometric space
				var building_pos = tile_to_iso(float(tile_x), float(tile_y))
				var building = create_building(entity_id, owner_id, building_pos, footprint_width, footprint_height, health, max_health)
				entities_container.add_child(building)
				entities[entity_id] = building
				print("Spawned generator %d at tile (%d, %d)" % [entity_id, tile_x, tile_y])

	# Remove entities that are no longer in the snapshot
	for entity_id in entities.keys():
		if not (entity_id in current_entity_ids):
			var entity = entities[entity_id]
			entity.queue_free()
			entities.erase(entity_id)
			# Remove from selection if it was selected
			if entity_id in selected_units:
				selected_units.erase(entity_id)
			# Clear target if it was the selected target
			if entity_id == selected_target_id:
				selected_target_id = -1
				selected_building = null
				selection_label.text = ""
			print("Removed entity %d" % entity_id)

	# Update player list
	update_player_list()

func _on_disconnected_from_server():
	connection_label.text = "Disconnected"
	local_client_id = -1
	selected_units.clear()
	selected_target_id = -1
	selected_building = null

	# Clear all entities
	for entity in entities.values():
		entity.queue_free()
	entities.clear()

func _process(delta):
	# Update FPS
	fps_label.text = "FPS: %d" % Engine.get_frames_per_second()

	# WASD/Arrow key camera panning
	var pan_direction = Vector2.ZERO

	if Input.is_key_pressed(KEY_W) or Input.is_key_pressed(KEY_UP):
		pan_direction.y -= 1
	if Input.is_key_pressed(KEY_S) or Input.is_key_pressed(KEY_DOWN):
		pan_direction.y += 1
	if Input.is_key_pressed(KEY_A) or Input.is_key_pressed(KEY_LEFT):
		pan_direction.x -= 1
	if Input.is_key_pressed(KEY_D) or Input.is_key_pressed(KEY_RIGHT):
		pan_direction.x += 1

	if pan_direction != Vector2.ZERO:
		pan_camera(pan_direction.normalized() * camera_pan_speed * delta)

func _input(event):
	# Handle scroll/zoom in _input (before _unhandled_input)
	if event is InputEventMouseButton:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			zoom_camera(camera_zoom_step)
			get_viewport().set_input_as_handled()
			return
		elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			zoom_camera(-camera_zoom_step)
			get_viewport().set_input_as_handled()
			return

	# Handle attack with Q key
	if event is InputEventKey and event.pressed and not event.echo and event.keycode == KEY_Q:
		_on_attack_button_pressed()

	# Handle formation hotkeys
	if event is InputEventKey and event.pressed and not event.echo:
		match event.keycode:
			KEY_1:
				set_formation("box")
			KEY_2:
				set_formation("line")
			KEY_3:
				set_formation("spread")

func _draw():
	if tile_size == 0:
		return

	# Draw isometric diamond grid
	for x in range(arena_tiles_width):
		for y in range(arena_tiles_height):
			# Get the 4 corners of this tile in isometric space
			var p1 = tile_to_iso(float(x), float(y))
			var p2 = tile_to_iso(float(x + 1), float(y))
			var p3 = tile_to_iso(float(x + 1), float(y + 1))
			var p4 = tile_to_iso(float(x), float(y + 1))

			# Draw diamond outline
			draw_line(p1, p2, Color(0, 1, 0, 0.3), 1.0)
			draw_line(p2, p3, Color(0, 1, 0, 0.3), 1.0)
			draw_line(p3, p4, Color(0, 1, 0, 0.3), 1.0)
			draw_line(p4, p1, Color(0, 1, 0, 0.3), 1.0)

	# Draw origin marker at tile (0,0) center
	draw_circle(tile_to_iso(0, 0), 5.0, Color(1, 0, 0, 1.0))

	# Draw selection box while dragging
	if is_dragging:
		var rect = make_rect(drag_start_pos, drag_current_pos)
		# Draw semi-transparent fill
		draw_rect(rect, Color(0, 1, 0, 0.2))
		# Draw outline
		draw_rect(rect, Color(0, 1, 0, 0.8), false, 2.0)

func _unhandled_input(event):
	# Mouse wheel / trackpad zoom (works even when not connected)
	if event is InputEventMouseButton:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			zoom_camera(camera_zoom_step)
			get_viewport().set_input_as_handled()
		elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			zoom_camera(-camera_zoom_step)
			get_viewport().set_input_as_handled()

	# Trackpad pinch/magnify gesture for zoom
	elif event is InputEventMagnifyGesture:
		# factor > 1.0 means zoom in (pinch out), < 1.0 means zoom out (pinch in)
		var zoom_change = (event.factor - 1.0) * 0.5  # Scale the gesture
		zoom_camera(zoom_change)
		get_viewport().set_input_as_handled()

	if not network_manager.is_connected or tile_size == 0:
		return

	# Left mouse button pressed - start drag
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		is_dragging = true
		drag_start_pos = get_local_mouse_position()
		drag_current_pos = drag_start_pos

	# Mouse motion - update drag
	elif event is InputEventMouseMotion and is_dragging:
		drag_current_pos = get_local_mouse_position()
		queue_redraw()  # Redraw to show selection box

	# Left mouse button released - finish selection
	elif event is InputEventMouseButton and not event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		if is_dragging:
			is_dragging = false
			queue_redraw()  # Clear selection box

			var drag_distance = drag_start_pos.distance_to(drag_current_pos)

			if drag_distance < DRAG_THRESHOLD:
				# Small drag = single click selection
				var clicked_entity_id = get_entity_at_position(drag_start_pos)
				print("Clicked entity ID: ", clicked_entity_id)

				if clicked_entity_id != -1:
					var entity = entities.get(clicked_entity_id)
					var entity_owner = entity.get_meta("owner_id", -1) if entity else -1
					print("Entity owner: ", entity_owner, " local_client_id: ", local_client_id)
					if entity and entity.has_method("get_meta") and entity_owner == local_client_id:
						# This is our unit - select it
						selected_units.clear()
						selected_units.append(clicked_entity_id)
						update_selection_visual()
						log_event("Selected unit %d" % clicked_entity_id)
						print("Selected unit: ", clicked_entity_id)
					else:
						print("Not our unit or invalid entity")
				else:
					# Clicked empty space - deselect all
					selected_units.clear()
					update_selection_visual()
			else:
				# Large drag = box selection
				var rect = make_rect(drag_start_pos, drag_current_pos)
				var selected_entity_ids = get_entities_in_rect(rect)

				if selected_entity_ids.size() > 0:
					selected_units = selected_entity_ids
					update_selection_visual()
					log_event("Selected %d units" % selected_units.size())
					print("Box selected units: ", selected_units)
				else:
					# No units in box - deselect all
					selected_units.clear()
					update_selection_visual()

	# Right click to move selected units
	elif event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_RIGHT:
		if selected_units.is_empty():
			log_event("No units selected")
			return

		# Convert isometric click to tile coordinates
		var click_pos = get_local_mouse_position()
		var tile_coords = iso_to_tile(click_pos)

		print("Move command: units ", selected_units, " -> tile: ", tile_coords)

		# Send move command with selected unit IDs and formation
		var commands = [{
			"type": "move",
			"data": {
				"unitIds": selected_units,
				"targetTileX": tile_coords.x,
				"targetTileY": tile_coords.y,
				"formation": current_formation
			}
		}]
		network_manager.send_input(commands)
		log_event("Moving %d units to tile (%d, %d) in %s formation" % [selected_units.size(), tile_coords.x, tile_coords.y, current_formation])

func get_entity_at_position(screen_pos: Vector2) -> int:
	# Check all entities to see if click is within their bounds
	for entity_id in entities:
		var entity = entities[entity_id]
		if entity and entity is Node2D:
			# Simple distance check (works for circular units)
			# Both screen_pos and entity.position are in local coordinates
			var distance = entity.position.distance_to(screen_pos)
			if distance < 15:  # Click radius
				return entity_id
	return -1

func update_selection_visual():
	# Clear all selection visuals first
	for entity_id in entities:
		var entity = entities[entity_id]
		if entity and entity.has_method("set_selected"):
			entity.set_selected(false)

	# Show selection for selected units
	for unit_id in selected_units:
		if unit_id in entities:
			var entity = entities[unit_id]
			if entity and entity.has_method("set_selected"):
				entity.set_selected(true)

func set_formation(formation: String):
	# Update formation
	current_formation = formation

	# Update UI
	var formation_name = formation.capitalize()
	formation_label.text = "Formation: " + formation_name
	log_event("Formation changed to: " + formation_name)

	# Update button highlighting (simple text-based)
	box_formation_button.text = "Box (1)" if formation != "box" else "► Box (1)"
	line_formation_button.text = "Line (2)" if formation != "line" else "► Line (2)"
	spread_formation_button.text = "Spread (3)" if formation != "spread" else "► Spread (3)"

	# If units are selected, re-form them in place if they're close together
	if selected_units.size() > 1:
		# Calculate center position of selected units (in tiles)
		var center_tile_x: float = 0.0
		var center_tile_y: float = 0.0
		var max_distance: float = 0.0

		for unit_id in selected_units:
			if unit_id in entities:
				var entity = entities[unit_id]
				var screen_pos = entity.position
				var tile_pos = iso_to_tile(screen_pos)
				center_tile_x += tile_pos.x
				center_tile_y += tile_pos.y

		center_tile_x /= selected_units.size()
		center_tile_y /= selected_units.size()

		# Check if all units are within threshold distance (5 tiles)
		const REFORM_THRESHOLD: float = 5.0
		var all_close: bool = true
		for unit_id in selected_units:
			if unit_id in entities:
				var entity = entities[unit_id]
				var screen_pos = entity.position
				var tile_pos = iso_to_tile(screen_pos)
				var distance = Vector2(tile_pos.x - center_tile_x, tile_pos.y - center_tile_y).length()
				if distance > REFORM_THRESHOLD:
					all_close = false
					break

		# If all units are close, send move command to center with new formation
		if all_close:
			var center_tile = Vector2i(int(round(center_tile_x)), int(round(center_tile_y)))

			# Validate bounds
			if center_tile.x >= 0 and center_tile.x < arena_tiles_width and center_tile.y >= 0 and center_tile.y < arena_tiles_height:
				var commands = [{
					"type": "move",
					"data": {
						"unitIds": selected_units,
						"targetTileX": center_tile.x,
						"targetTileY": center_tile.y,
						"formation": current_formation
					}
				}]
				network_manager.send_input(commands)
				log_event("Re-forming %d units in %s formation" % [selected_units.size(), formation_name])

func make_rect(from: Vector2, to: Vector2) -> Rect2:
	# Create normalized rectangle (handles dragging left/up)
	var pos = Vector2(min(from.x, to.x), min(from.y, to.y))
	var size = Vector2(abs(to.x - from.x), abs(to.y - from.y))
	return Rect2(pos, size)

func get_entities_in_rect(rect: Rect2) -> Array[int]:
	# Find all owned units within the selection rectangle
	var selected: Array[int] = []
	for entity_id in entities:
		var entity = entities[entity_id]
		if entity and entity is Node2D:
			# Check if this is our unit (not a building)
			var entity_owner = entity.get_meta("owner_id", -1) if entity.has_method("get_meta") else -1
			if entity_owner != local_client_id:
				continue

			# Skip buildings (only select workers/units)
			if entity.has_meta("entity_id") and not entity.has_method("set_selected"):
				continue

			# Check if entity position is within rectangle
			if rect.has_point(entity.position):
				selected.append(entity_id)
	return selected

func update_player_list():
	var text = "Players:\n"
	for player_id_str in players_data:
		var player_data = players_data[player_id_str]
		var player_id = int(player_data.get("id", -1))  # JSON→int conversion
		var player_name = player_data.get("name", "Unknown")
		var money = player_data.get("money", 0.0)

		if player_id == local_client_id:
			text += "• You: $%.0f\n" % money
		else:
			text += "• %s: $%.0f\n" % [player_name, money]
	player_list_label.text = text

func update_building_health(building: Node2D, health: int, max_health: int):
	building.set_meta("health", health)
	building.set_meta("max_health", max_health)
	if building.has_meta("health_bar"):
		var health_bar = building.get_meta("health_bar")
		health_bar.value = (float(health) / float(max_health)) * 100.0

func create_building(entity_id: int, owner_id: int, pos: Vector2, footprint_w: int, footprint_h: int, health: int, max_health: int) -> Node2D:
	var building = Node2D.new()
	building.position = pos
	building.set_meta("entity_id", entity_id)
	building.set_meta("owner_id", owner_id)
	building.set_meta("health", health)
	building.set_meta("max_health", max_health)

	# Visual size based on isometric tile footprint
	var visual_width = footprint_w * ISO_TILE_WIDTH
	var visual_height = footprint_h * ISO_TILE_HEIGHT
	var building_height = 40.0  # 3D height in pixels

	# Use Polygon2D to draw isometric box
	var iso_box = Polygon2D.new()
	var base_color = Color(1, 0.8, 0, 1) if owner_id == local_client_id else Color(0.8, 0.4, 0, 1)

	# Draw as isometric box (top face + two visible sides)
	# Top face (diamond shape)
	var top_points = PackedVector2Array([
		Vector2(visual_width / 2, 0),                                    # Top
		Vector2(visual_width, visual_height / 2),                        # Right
		Vector2(visual_width / 2, visual_height),                        # Bottom
		Vector2(0, visual_height / 2)                                    # Left
	])
	iso_box.polygon = top_points
	iso_box.color = base_color.lightened(0.2)
	building.add_child(iso_box)

	# Right face (darker)
	var right_face = Polygon2D.new()
	var right_points = PackedVector2Array([
		Vector2(visual_width, visual_height / 2),
		Vector2(visual_width, visual_height / 2 + building_height),
		Vector2(visual_width / 2, visual_height + building_height),
		Vector2(visual_width / 2, visual_height)
	])
	right_face.polygon = right_points
	right_face.color = base_color.darkened(0.2)
	building.add_child(right_face)

	# Left face (even darker)
	var left_face = Polygon2D.new()
	var left_points = PackedVector2Array([
		Vector2(0, visual_height / 2),
		Vector2(visual_width / 2, visual_height),
		Vector2(visual_width / 2, visual_height + building_height),
		Vector2(0, visual_height / 2 + building_height)
	])
	left_face.polygon = left_points
	left_face.color = base_color.darkened(0.4)
	building.add_child(left_face)

	# Health bar (above the building)
	var health_bar = ProgressBar.new()
	health_bar.position = Vector2(0, -15)
	health_bar.size = Vector2(visual_width, 8)
	health_bar.max_value = 100
	health_bar.value = (float(health) / float(max_health)) * 100.0
	health_bar.show_percentage = false
	building.add_child(health_bar)
	building.set_meta("health_bar", health_bar)

	# Label (below the building)
	var label = Label.new()
	label.text = "Generator"
	label.position = Vector2(0, visual_height + building_height + 2)
	building.add_child(label)

	# Make clickable (click area covers the whole building including height)
	var input_area = Area2D.new()
	input_area.input_pickable = true
	var collision = CollisionShape2D.new()
	var shape = RectangleShape2D.new()
	shape.size = Vector2(visual_width, visual_height + building_height)
	collision.shape = shape
	collision.position = Vector2(visual_width / 2, (visual_height + building_height) / 2)
	input_area.add_child(collision)
	building.add_child(input_area)
	input_area.input_event.connect(_on_building_clicked.bind(entity_id))

	# Selection highlight (outline around base)
	var highlight = Polygon2D.new()
	var highlight_points = PackedVector2Array([
		Vector2(visual_width / 2 - 2, -2),
		Vector2(visual_width + 2, visual_height / 2 - 2),
		Vector2(visual_width / 2 + 2, visual_height + 2),
		Vector2(-2, visual_height / 2 + 2)
	])
	highlight.polygon = highlight_points
	highlight.color = Color(1, 1, 0, 0.5)
	highlight.visible = false
	highlight.z_index = -1
	building.add_child(highlight)
	building.set_meta("highlight", highlight)

	return building

func _on_building_clicked(viewport, event, shape_idx, entity_id):
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		# Clear previous selection
		if selected_building != null and selected_building.has_meta("highlight"):
			var old_highlight = selected_building.get_meta("highlight")
			old_highlight.visible = false

		# Set new target
		selected_target_id = entity_id
		if entity_id in entities:
			selected_building = entities[entity_id]
			if selected_building.has_meta("highlight"):
				var highlight = selected_building.get_meta("highlight")
				highlight.visible = true

			var owner_id = selected_building.get_meta("owner_id") if selected_building.has_meta("owner_id") else -1
			if owner_id == local_client_id:
				selection_label.text = "Selected: Your Generator #%d" % entity_id
				log_event("Selected your generator #%d" % entity_id)
			else:
				selection_label.text = "Selected: Enemy Generator #%d" % entity_id
				log_event("Selected enemy generator #%d - press Q to attack!" % entity_id)

# Removed - infer events from snapshot changes instead

func log_event(message: String):
	event_messages.append(message)
	if event_messages.size() > 10:
		event_messages.pop_front()

	var log_text = "Events:\n"
	for msg in event_messages:
		log_text += "• " + msg + "\n"
	event_log.text = log_text

	# Auto-scroll to bottom
	event_log.scroll_to_line(event_log.get_line_count() - 1)

func _on_build_button_pressed():
	if not network_manager.is_connected or tile_size == 0 or selected_units.is_empty():
		log_event("No units selected to build!")
		return

	# Build near the first selected unit
	var first_unit_id = selected_units[0]
	if not (first_unit_id in entities):
		return

	var first_unit = entities[first_unit_id]
	var unit_tile = iso_to_tile(first_unit.position)

	# Build 3 tiles to the right
	var build_tile_x = unit_tile.x + 3
	var build_tile_y = unit_tile.y

	# Client-side validation
	if not can_build_at_tile(build_tile_x, build_tile_y):
		return

	# Send build command
	var commands = [{
		"type": "build",
		"data": {
			"buildingType": "generator",
			"tileX": build_tile_x,
			"tileY": build_tile_y
		}
	}]
	network_manager.send_input(commands)

	# Client-side prediction: assume success (will be corrected by snapshot if wrong)
	log_event("Building generator at tile (%d, %d)..." % [build_tile_x, build_tile_y])

func can_build_at_tile(tile_x: int, tile_y: int) -> bool:
	var building_footprint_w = 2  # Generators are 2x2
	var building_footprint_h = 2

	# Check money
	if local_money < 50:
		log_event("Not enough money to build!")
		return false

	# Check bounds
	if tile_x < 0 or tile_x + building_footprint_w > arena_tiles_width or \
	   tile_y < 0 or tile_y + building_footprint_h > arena_tiles_height:
		log_event("Can't build out of bounds!")
		return false

	# TODO: Check collision with existing buildings (tile-based)
	# For now, just allow it - server will reject if invalid

	return true

func _on_attack_button_pressed():
	if not network_manager.is_connected or selected_target_id == -1:
		log_event("No target selected!")
		return

	# Check if target entity exists and is not owned by us
	if not (selected_target_id in entities):
		log_event("Target no longer exists!")
		return

	var target = entities[selected_target_id]
	var target_owner = target.get_meta("owner_id") if target.has_meta("owner_id") else -1

	if target_owner == local_client_id:
		log_event("Can't attack your own buildings!")
		return

	var commands = [{
		"type": "attack",
		"data": {
			"targetId": selected_target_id
		}
	}]
	network_manager.send_input(commands)
	log_event("Attacking entity %d..." % selected_target_id)
