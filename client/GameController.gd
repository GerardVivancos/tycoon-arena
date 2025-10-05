extends Node2D

@onready var network_manager = $NetworkManager
@onready var entities_container = $Entities
@onready var connection_label = $UI/ConnectionStatus
@onready var fps_label = $UI/FPS
@onready var player_list_label = $UI/PlayerList
@onready var money_label = $UI/MoneyLabel
@onready var build_button = $UI/BuildButton
@onready var attack_button = $UI/AttackButton
@onready var selection_label = $UI/SelectionLabel
@onready var event_log = $UI/EventLog

# Tile system (from server via handshake)
var tile_size: int
var arena_tiles_width: int
var arena_tiles_height: int

# Rendering - Isometric
const PIXELS_PER_TILE = 32  # Visual size of each tile (for top-down, kept for reference)
const ISO_TILE_WIDTH = 64   # Width of isometric diamond
const ISO_TILE_HEIGHT = 32  # Height of isometric diamond
const ISO_OFFSET_X = 400    # Screen offset to center the map
const ISO_OFFSET_Y = 100

var player_scene = preload("res://Player.tscn")
var entities: Dictionary = {}  # entity_id -> Player node or Building node
var local_player: Node = null
var local_client_id: int = -1
var local_money: float = 0.0
var players_data: Dictionary = {}
var selected_entity_id: int = -1
var selected_building: Node2D = null
var event_messages: Array = []

func _ready():
	# Connect network signals
	network_manager.connected_to_server.connect(_on_connected_to_server)
	network_manager.snapshot_received.connect(_on_snapshot_received)
	network_manager.disconnected_from_server.connect(_on_disconnected_from_server)

	# Connect UI signals
	build_button.pressed.connect(_on_build_button_pressed)
	attack_button.pressed.connect(_on_attack_button_pressed)

	# Auto-connect on start
	network_manager.connect_to_server("Player" + str(randi() % 1000))

func _on_connected_to_server(client_id: int, tick_rate: int, tile_sz: int, tiles_w: int, tiles_h: int):
	local_client_id = client_id
	tile_size = tile_sz
	arena_tiles_width = tiles_w
	arena_tiles_height = tiles_h
	connection_label.text = "Connected (ID: %d)" % client_id
	print("Connected with client ID: %d, Arena: %dx%d tiles" % [client_id, tiles_w, tiles_h])
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

		if entity_type == "player":
			if entity_id in entities:
				# Update existing entity
				var player = entities[entity_id]
				player.update_from_snapshot(screen_pos, health, max_health)
			else:
				# Create new entity
				var player = player_scene.instantiate()
				var is_local = (owner_id == local_client_id)
				player.setup(entity_id, owner_id, screen_pos, is_local)

				if is_local:
					local_player = player
					player.set_player_name("You")
				else:
					player.set_player_name("Player %d" % owner_id)

				entities_container.add_child(player)
				entities[entity_id] = player
				print("Spawned player entity %d at tile (%d, %d) -> screen (%f, %f)" % [entity_id, tile_x, tile_y, screen_pos.x, screen_pos.y])

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
			if entity == local_player:
				local_player = null
			if entity_id == selected_entity_id:
				selected_entity_id = -1
				selected_building = null
				selection_label.text = "No target selected"
				log_event("Target destroyed!")
			print("Removed entity %d" % entity_id)

	# Update player list
	update_player_list()

func _on_disconnected_from_server():
	connection_label.text = "Disconnected"
	local_client_id = -1
	local_player = null

	# Clear all entities
	for entity in entities.values():
		entity.queue_free()
	entities.clear()

func _process(delta):
	# Update FPS
	fps_label.text = "FPS: %d" % Engine.get_frames_per_second()

func _input(event):
	# Handle attack with Q key
	if event is InputEventKey and event.pressed and not event.echo and event.keycode == KEY_Q:
		_on_attack_button_pressed()

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

func _unhandled_input(event):
	if not network_manager.is_connected or local_player == null or tile_size == 0:
		return

	# Click to move
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_RIGHT:
		# Convert isometric click to tile coordinates
		var click_pos = get_local_mouse_position()
		var tile_coords = iso_to_tile(click_pos)

		print("Click at local: ", click_pos, " -> tile: ", tile_coords)

		# Send move command
		var commands = [{
			"type": "move",
			"data": {
				"targetTileX": tile_coords.x,
				"targetTileY": tile_coords.y
			}
		}]
		network_manager.send_input(commands)
		log_event("Moving to tile (%d, %d)" % [tile_coords.x, tile_coords.y])

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

		# Set new selection
		selected_entity_id = entity_id
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

func _on_build_button_pressed():
	if not network_manager.is_connected or local_player == null or tile_size == 0:
		return

	# Build near the player - convert player isometric position to tiles
	var player_tile = iso_to_tile(local_player.position)

	# Build 3 tiles to the right
	var build_tile_x = player_tile.x + 3
	var build_tile_y = player_tile.y

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
	if not network_manager.is_connected or selected_entity_id == -1:
		return

	# Check if selected entity exists and is not owned by us
	if not (selected_entity_id in entities):
		log_event("No valid target selected!")
		return

	var target = entities[selected_entity_id]
	var target_owner = target.get_meta("owner_id") if target.has_meta("owner_id") else -1

	if target_owner == local_client_id:
		log_event("Can't attack your own buildings!")
		return

	var commands = [{
		"type": "attack",
		"data": {
			"targetId": selected_entity_id
		}
	}]
	network_manager.send_input(commands)
	log_event("Attacking entity %d..." % selected_entity_id)
