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

var player_scene = preload("res://Player.tscn")
var entities: Dictionary = {}  # entity_id -> Player node or Building node
var local_player: Node = null
var local_client_id: int = -1
var input_timer: float = 0.0
var input_send_rate: float = 0.05  # Send inputs 20 times per second (50ms)
var local_money: float = 0.0
var players_data: Dictionary = {}
var selected_entity_id: int = -1
var selected_building: Node2D = null  # Reference to selected building for visual feedback
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

func _on_connected_to_server(client_id: int, tick_rate: int):
	local_client_id = client_id
	connection_label.text = "Connected (ID: %d)" % client_id
	print("Connected with client ID: %d" % client_id)

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
		# JSON has no integer type - all numbers are floats
		# Convert to int for type-safe dictionary keys and comparisons
		var entity_id = int(entity_data.get("id", -1))
		var owner_id = int(entity_data.get("ownerId", -1))
		var entity_type = entity_data.get("type", "")
		var x = entity_data.get("x", 0.0)
		var y = entity_data.get("y", 0.0)
		var health = entity_data.get("health", 100)
		var max_health = entity_data.get("maxHealth", 100)
		var width = entity_data.get("width", 0.0)
		var height = entity_data.get("height", 0.0)

		current_entity_ids[entity_id] = true

		if entity_type == "player":
			if entity_id in entities:
				# Update existing entity
				var player = entities[entity_id]
				player.update_from_snapshot(Vector2(x, y), health, max_health)
			else:
				# Create new entity
				var player = player_scene.instantiate()
				var is_local = (owner_id == local_client_id)
				player.setup(entity_id, owner_id, Vector2(x, y), is_local)

				if is_local:
					local_player = player
					player.set_player_name("You")
				else:
					player.set_player_name("Player %d" % owner_id)

				entities_container.add_child(player)
				entities[entity_id] = player
				print("Spawned player entity %d at (%f, %f)" % [entity_id, x, y])

		elif entity_type == "generator":
			if entity_id in entities:
				# Update existing building
				var building = entities[entity_id]
				update_building_health(building, health, max_health)
			else:
				# Create new building
				var building = create_building(entity_id, owner_id, Vector2(x, y), width, height, health, max_health)
				entities_container.add_child(building)
				entities[entity_id] = building
				print("Spawned generator %d at (%f, %f)" % [entity_id, x, y])

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

	# Handle input and send to server
	input_timer += delta
	if input_timer >= input_send_rate:
		input_timer = 0.0
		handle_input(input_send_rate)

func handle_input(delta_time: float):
	if not network_manager.is_connected or local_player == null:
		return

	# Attack with Q key
	if Input.is_action_just_pressed("ui_focus_prev"):  # Q key
		_on_attack_button_pressed()

	var movement = Vector2.ZERO

	# Get input
	if Input.is_action_pressed("ui_up"):
		movement.y -= 1
	if Input.is_action_pressed("ui_down"):
		movement.y += 1
	if Input.is_action_pressed("ui_left"):
		movement.x -= 1
	if Input.is_action_pressed("ui_right"):
		movement.x += 1

	# Normalize diagonal movement
	if movement.length() > 0:
		movement = movement.normalized()

		# Apply client-side prediction
		local_player.apply_input(movement, delta_time)

		# Send input to server
		var commands = [{
			"type": "move",
			"data": {
				"deltaX": movement.x * 200.0 * delta_time,  # Match server speed
				"deltaY": movement.y * 200.0 * delta_time
			}
		}]
		network_manager.send_input(commands)

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

func create_building(entity_id: int, owner_id: int, pos: Vector2, width: float, height: float, health: int, max_health: int) -> Node2D:
	var building = Node2D.new()
	building.position = pos
	building.set_meta("entity_id", entity_id)
	building.set_meta("owner_id", owner_id)
	building.set_meta("health", health)
	building.set_meta("max_health", max_health)

	# Visual representation
	var rect = ColorRect.new()
	rect.size = Vector2(width, height)
	rect.color = Color(1, 0.8, 0, 1) if owner_id == local_client_id else Color(0.8, 0.4, 0, 1)
	rect.mouse_filter = Control.MOUSE_FILTER_IGNORE  # Let clicks pass through to Area2D!
	building.add_child(rect)

	# Health bar
	var health_bar = ProgressBar.new()
	health_bar.position = Vector2(0, -10)
	health_bar.size = Vector2(width, 8)
	health_bar.max_value = 100
	health_bar.value = (float(health) / float(max_health)) * 100.0
	health_bar.show_percentage = false
	building.add_child(health_bar)
	building.set_meta("health_bar", health_bar)

	# Label
	var label = Label.new()
	label.text = "Generator"
	label.position = Vector2(0, height + 2)
	building.add_child(label)

	# Make clickable
	var input_area = Area2D.new()
	input_area.input_pickable = true  # Required for input_event to work!
	var collision = CollisionShape2D.new()
	var shape = RectangleShape2D.new()
	shape.size = Vector2(width, height)
	collision.shape = shape
	collision.position = Vector2(width / 2, height / 2)
	input_area.add_child(collision)
	building.add_child(input_area)
	input_area.input_event.connect(_on_building_clicked.bind(entity_id))

	# Selection highlight (hidden by default)
	var highlight = ColorRect.new()
	highlight.size = Vector2(width + 4, height + 4)
	highlight.position = Vector2(-2, -2)
	highlight.color = Color(1, 1, 0, 0.5)  # Yellow highlight
	highlight.visible = false
	highlight.z_index = -1
	highlight.mouse_filter = Control.MOUSE_FILTER_IGNORE  # Don't block clicks
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
	if not network_manager.is_connected or local_player == null:
		return

	# Build near the player
	var build_pos = local_player.position + Vector2(60, 0)

	# Client-side validation (matches server logic)
	if not can_build_at(build_pos):
		return

	# Send build command
	var commands = [{
		"type": "build",
		"data": {
			"buildingType": "generator",
			"x": build_pos.x,
			"y": build_pos.y
		}
	}]
	network_manager.send_input(commands)

	# Client-side prediction: assume success (will be corrected by snapshot if wrong)
	log_event("Building generator...")

func can_build_at(pos: Vector2) -> bool:
	var building_size = 40.0

	# Check money
	if local_money < 50:
		log_event("Not enough money to build!")
		return false

	# Check bounds
	if pos.x < 0 or pos.x + building_size > 800 or pos.y < 0 or pos.y + building_size > 600:
		log_event("Can't build out of bounds!")
		return false

	# Check collision with existing buildings
	for entity_id in entities:
		var entity = entities[entity_id]
		if entity.has_meta("entity_id") and entity.has_meta("owner_id"):
			# It's a building
			var entity_pos = entity.position
			var entity_width = 40.0  # All generators are 40x40
			var entity_height = 40.0

			# AABB collision check
			if pos.x < entity_pos.x + entity_width and \
			   pos.x + building_size > entity_pos.x and \
			   pos.y < entity_pos.y + entity_height and \
			   pos.y + building_size > entity_pos.y:
				log_event("Can't build here - overlapping!")
				return false

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
