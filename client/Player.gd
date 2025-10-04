extends CharacterBody2D

var entity_id: int = -1
var owner_id: int = -1
var is_local_player: bool = false
var player_name: String = "Player"
var health: int = 100
var max_health: int = 100

# For interpolation
var target_position: Vector2
var interpolation_speed: float = 10.0

# For prediction (local player only)
var predicted_position: Vector2
var last_server_position: Vector2
var input_buffer: Array = []

func _ready():
	if is_local_player:
		$ColorRect.color = Color(0, 1, 0.5, 1)  # Green for local player
		predicted_position = position
		last_server_position = position

func setup(id: int, owner: int, pos: Vector2, is_local: bool = false):
	entity_id = id
	owner_id = owner
	position = pos
	target_position = pos
	is_local_player = is_local

	if is_local:
		$ColorRect.color = Color(0, 1, 0.5, 1)  # Green
		predicted_position = pos
		last_server_position = pos
	else:
		$ColorRect.color = Color(0, 0.5, 1, 1)  # Blue

func update_from_snapshot(pos: Vector2, hp: int, max_hp: int):
	health = hp
	max_health = max_hp

	# Update health bar
	$HealthBar.value = (float(health) / float(max_health)) * 100.0

	if is_local_player:
		# Reconciliation for local player
		last_server_position = pos
		var prediction_error = pos - predicted_position

		# If error is significant, correct it
		if prediction_error.length() > 2.0:
			predicted_position = pos
			position = pos
		target_position = predicted_position
	else:
		# Simple interpolation for other players
		target_position = pos

func apply_input(movement: Vector2, delta_time: float):
	if not is_local_player:
		return

	# Apply predicted movement
	var speed = 200.0  # Units per second (match server)
	predicted_position += movement * speed * delta_time

	# Clamp to arena bounds (match server)
	predicted_position.x = clamp(predicted_position.x, 0, 800)
	predicted_position.y = clamp(predicted_position.y, 0, 600)

	target_position = predicted_position

	# Store input for potential reconciliation
	input_buffer.append({
		"movement": movement,
		"position": predicted_position
	})

	# Keep only recent inputs
	if input_buffer.size() > 60:  # About 3 seconds at 20Hz
		input_buffer.pop_front()

func _physics_process(delta):
	# Smooth interpolation to target position
	if not is_local_player or position.distance_to(target_position) > 1.0:
		position = position.lerp(target_position, interpolation_speed * delta)

func set_player_name(name: String):
	player_name = name
	$PlayerNameLabel.text = name