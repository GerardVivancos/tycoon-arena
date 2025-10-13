extends CharacterBody2D

var entity_id: int = -1
var owner_id: int = -1
var is_local_player: bool = false
var player_name: String = "Player"
var health: int = 100
var max_health: int = 100
var is_selected: bool = false

# For interpolation
var target_position: Vector2
var interpolation_speed: float = 10.0

# Selection ring references
var selection_ring: Polygon2D = null
var outer_selection_ring: Polygon2D = null

func _ready():
	# Create isometric player visual
	create_isometric_sprite()

func set_selected(selected: bool):
	is_selected = selected
	# Update selection visual using direct references
	if selection_ring:
		selection_ring.visible = selected
	if outer_selection_ring:
		outer_selection_ring.visible = selected

func setup(id: int, owner: int, pos: Vector2, is_local: bool = false):
	entity_id = id
	owner_id = owner
	position = pos
	target_position = pos
	is_local_player = is_local

	# Set metadata for selection system
	set_meta("entity_id", id)
	set_meta("owner_id", owner)

	create_isometric_sprite()

func create_isometric_sprite():
	# Clear old visuals
	for child in get_children():
		if child.name in ["Shadow", "Body", "ColorRect", "SelectionRing", "OuterSelectionRing"]:
			child.queue_free()

	var base_color = Color(0, 1, 0.5, 1) if is_local_player else Color(0, 0.5, 1, 1)

	# Selection ring - outer dark ring for contrast (hidden by default)
	outer_selection_ring = Polygon2D.new()
	outer_selection_ring.name = "OuterSelectionRing"
	var outer_points = PackedVector2Array()
	for i in range(20):
		var angle = i * PI * 2 / 20
		outer_points.append(Vector2(cos(angle) * 20, sin(angle) * 10 + 10))
	outer_selection_ring.polygon = outer_points
	outer_selection_ring.color = Color(0, 0, 0, 0.8)  # Dark outline
	outer_selection_ring.visible = is_selected
	outer_selection_ring.z_index = 0
	add_child(outer_selection_ring)

	# Selection ring - bright yellow inner ring (hidden by default)
	selection_ring = Polygon2D.new()
	selection_ring.name = "SelectionRing"
	var ring_points = PackedVector2Array()
	for i in range(20):
		var angle = i * PI * 2 / 20
		ring_points.append(Vector2(cos(angle) * 18, sin(angle) * 9 + 10))
	selection_ring.polygon = ring_points
	selection_ring.color = Color(1, 1, 0, 1.0)  # Bright yellow, fully opaque
	selection_ring.visible = is_selected
	selection_ring.z_index = 1
	add_child(selection_ring)

	# Shadow (ellipse at base)
	var shadow = Polygon2D.new()
	shadow.name = "Shadow"
	var shadow_points = PackedVector2Array()
	for i in range(16):
		var angle = i * PI * 2 / 16
		shadow_points.append(Vector2(cos(angle) * 12, sin(angle) * 6 + 10))
	shadow.polygon = shadow_points
	shadow.color = Color(0, 0, 0, 0.3)
	shadow.z_index = -1
	add_child(shadow)

	# Body (circle with slight vertical offset for height)
	var body = Polygon2D.new()
	body.name = "Body"
	var body_points = PackedVector2Array()
	for i in range(16):
		var angle = i * PI * 2 / 16
		body_points.append(Vector2(cos(angle) * 10, sin(angle) * 10))
	body.polygon = body_points
	body.color = base_color
	add_child(body)

func update_from_snapshot(pos: Vector2, hp: int, max_hp: int):
	health = hp
	max_health = max_hp

	# Update health bar
	$HealthBar.value = (float(health) / float(max_health)) * 100.0

	# Smooth interpolation to new position (server-authoritative)
	target_position = pos

func _physics_process(delta):
	# Smooth interpolation to target position (from server snapshots)
	if position.distance_to(target_position) > 1.0:
		position = position.lerp(target_position, interpolation_speed * delta)

func set_player_name(name: String):
	player_name = name
	$PlayerNameLabel.text = name
