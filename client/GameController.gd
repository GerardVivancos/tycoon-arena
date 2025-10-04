extends Node2D

@onready var network_manager = $NetworkManager
@onready var entities_container = $Entities
@onready var connection_label = $UI/ConnectionStatus
@onready var fps_label = $UI/FPS
@onready var player_list_label = $UI/PlayerList

var player_scene = preload("res://Player.tscn")
var entities: Dictionary = {}  # entity_id -> Player node
var local_player: Node = null
var local_client_id: int = -1
var input_timer: float = 0.0
var input_send_rate: float = 0.05  # Send inputs 20 times per second (50ms)

func _ready():
	# Connect network signals
	network_manager.connected_to_server.connect(_on_connected_to_server)
	network_manager.snapshot_received.connect(_on_snapshot_received)
	network_manager.disconnected_from_server.connect(_on_disconnected_from_server)

	# Auto-connect on start
	network_manager.connect_to_server("Player" + str(randi() % 1000))

func _on_connected_to_server(client_id: int, tick_rate: int):
	local_client_id = client_id
	connection_label.text = "Connected (ID: %d)" % client_id
	print("Connected with client ID: %d" % client_id)

func _on_snapshot_received(snapshot: Dictionary):
	var entities_data = snapshot.get("entities", [])

	# Track which entities are in the snapshot
	var current_entity_ids = {}

	for entity_data in entities_data:
		var entity_id = entity_data.get("id", -1)
		var owner_id = entity_data.get("ownerId", -1)
		var entity_type = entity_data.get("type", "")
		var x = entity_data.get("x", 0.0)
		var y = entity_data.get("y", 0.0)
		var health = entity_data.get("health", 100)
		var max_health = entity_data.get("maxHealth", 100)

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

	# Remove entities that are no longer in the snapshot
	for entity_id in entities.keys():
		if not (entity_id in current_entity_ids):
			var entity = entities[entity_id]
			entity.queue_free()
			entities.erase(entity_id)
			if entity == local_player:
				local_player = null
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
	for entity_id in entities:
		var player = entities[entity_id]
		if player.owner_id == local_client_id:
			text += "• You (ID: %d)\n" % player.owner_id
		else:
			text += "• Player %d\n" % player.owner_id
	player_list_label.text = text