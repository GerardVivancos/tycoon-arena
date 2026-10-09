extends Control

# Phase 5 visual editor: paint terrain, place spawn points / scenario units &
# buildings, record simple move/attack actions, then export a map file and a
# scenario file (consumed directly by LoadMap and the scenario harness), and run
# the scenario with `go test`.

const TILE_PX := 24
const ORIGIN := Vector2(20, 48)

enum Tool { PAINT_ROCK, PAINT_GRASS, SPAWN, UNIT_WORKER, UNIT_SOLDIER, BUILDING_GENERATOR, ERASE }

var tool: int = Tool.PAINT_ROCK

var map_width: int = 20
var map_height: int = 10
var map_name: String = "custom_map.json"
var scenario_name: String = "custom_scenario.json"

var tiles: Dictionary = {}         # "x,y" -> {type, passable, height}
var spawn_points: Array = []       # [{team, x, y, radius}]
var scenario_units: Array = []     # [{id, team, type, position:[x,y], label}]
var scenario_buildings: Array = [] # [{id, team, type, position:[x,y], label}]
var actions: Array = []            # [{tick, type, ...}]

var edit_team: int = 0
var action_tick: int = 0
var next_id: int = 1
var pending_action = null          # {type:"move"|"attack"} awaiting a target click

var maps_dir: String = ""
var server_dir: String = ""

var map_name_edit: LineEdit
var scenario_name_edit: LineEdit
var team_spin: SpinBox
var tick_spin: SpinBox
var go_edit: LineEdit
var output_edit: RichTextLabel


func _ready() -> void:
	var project_dir := ProjectSettings.globalize_path("res://")
	maps_dir = project_dir.path_join("..").path_join("maps")
	server_dir = project_dir.path_join("..").path_join("server")
	_build_ui()
	_log("Editor ready. maps_dir=" + maps_dir)


func _build_ui() -> void:
	var tools := HBoxContainer.new()
	tools.position = Vector2(10, 6)
	add_child(tools)
	_add_tool_button(tools, "Rock", Tool.PAINT_ROCK)
	_add_tool_button(tools, "Grass", Tool.PAINT_GRASS)
	_add_tool_button(tools, "Spawn", Tool.SPAWN)
	_add_tool_button(tools, "Worker", Tool.UNIT_WORKER)
	_add_tool_button(tools, "Soldier", Tool.UNIT_SOLDIER)
	_add_tool_button(tools, "Generator", Tool.BUILDING_GENERATOR)
	_add_tool_button(tools, "Erase", Tool.ERASE)

	var row := HBoxContainer.new()
	row.position = Vector2(10, 640)
	add_child(row)
	row.add_child(_label("Team"))
	team_spin = SpinBox.new()
	team_spin.min_value = 0
	team_spin.max_value = 7
	team_spin.value = edit_team
	team_spin.value_changed.connect(_on_team_changed)
	row.add_child(team_spin)
	row.add_child(_label("Tick"))
	tick_spin = SpinBox.new()
	tick_spin.min_value = 0
	tick_spin.max_value = 100000
	tick_spin.value = 0
	tick_spin.value_changed.connect(_on_tick_changed)
	row.add_child(tick_spin)

	var row2 := HBoxContainer.new()
	row2.position = Vector2(10, 678)
	add_child(row2)
	row2.add_child(_label("Map"))
	map_name_edit = LineEdit.new()
	map_name_edit.text = map_name
	map_name_edit.size = Vector2(170, 24)
	row2.add_child(map_name_edit)
	row2.add_child(_label("Scenario"))
	scenario_name_edit = LineEdit.new()
	scenario_name_edit.text = scenario_name
	scenario_name_edit.size = Vector2(180, 24)
	row2.add_child(scenario_name_edit)
	row2.add_child(_label("go"))
	go_edit = LineEdit.new()
	go_edit.text = "go"
	go_edit.size = Vector2(80, 24)
	row2.add_child(go_edit)

	var btns := HBoxContainer.new()
	btns.position = Vector2(10, 712)
	add_child(btns)
	_add_button(btns, "Load Map", _on_load_map)
	_add_button(btns, "Save Map", _on_save_map)
	_add_button(btns, "Export Scenario", _on_export_scenario)
	_add_button(btns, "Run Scenario", _on_run_scenario)
	_add_button(btns, "Add Move", _on_add_move)
	_add_button(btns, "Add Attack", _on_add_attack)
	_add_button(btns, "New", _on_new)

	output_edit = RichTextLabel.new()
	output_edit.position = Vector2(560, 48)
	output_edit.size = Vector2(360, 700)
	output_edit.scroll_following = true
	add_child(output_edit)
	queue_redraw()


# ---- Input -----------------------------------------------------------------

func _gui_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		var tile = _screen_to_tile(event.position)
		if tile == null:
			return
		if pending_action != null:
			_complete_pending(int(tile[0]), int(tile[1]))
		else:
			_apply_tool(int(tile[0]), int(tile[1]))


func _screen_to_tile(pos: Vector2):
	var tx := int(floor((pos.x - ORIGIN.x) / TILE_PX))
	var ty := int(floor((pos.y - ORIGIN.y) / TILE_PX))
	if tx < 0 or ty < 0 or tx >= map_width or ty >= map_height:
		return null
	return [tx, ty]


func _apply_tool(x: int, y: int) -> void:
	var key := "%d,%d" % [x, y]
	match tool:
		Tool.PAINT_ROCK:
			tiles[key] = {"type": "rock", "passable": false, "height": 2.0}
		Tool.PAINT_GRASS:
			tiles.erase(key)
		Tool.ERASE:
			tiles.erase(key)
			spawn_points = _filter_xy(spawn_points, x, y)
			scenario_units = _filter_pos(scenario_units, x, y)
			scenario_buildings = _filter_pos(scenario_buildings, x, y)
		Tool.SPAWN:
			spawn_points.append({"team": edit_team, "x": x, "y": y, "radius": 3})
		Tool.UNIT_WORKER:
			scenario_units.append(_new_unit("worker", x, y))
		Tool.UNIT_SOLDIER:
			scenario_units.append(_new_unit("soldier", x, y))
		Tool.BUILDING_GENERATOR:
			scenario_buildings.append({"id": "b%d" % next_id, "team": edit_team, "type": "generator", "position": [x, y], "label": ""})
			next_id += 1
	queue_redraw()


func _new_unit(type_name: String, x: int, y: int) -> Dictionary:
	var id := "%s%d" % [type_name.substr(0, 1), next_id]
	next_id += 1
	return {"id": id, "team": edit_team, "type": type_name, "position": [x, y], "label": ""}


func _filter_xy(arr: Array, x: int, y: int) -> Array:
	var out := []
	for it in arr:
		if int(it["x"]) != x or int(it["y"]) != y:
			out.append(it)
	return out


func _filter_pos(arr: Array, x: int, y: int) -> Array:
	var out := []
	for it in arr:
		var p = it["position"]
		if int(p[0]) != x or int(p[1]) != y:
			out.append(it)
	return out


# ---- Actions ---------------------------------------------------------------

func _on_add_move() -> void:
	pending_action = {"type": "move"}
	_log("Click a target tile to MOVE team %d units (tick %d)" % [edit_team, action_tick])


func _on_add_attack() -> void:
	pending_action = {"type": "attack"}
	_log("Click the enemy entity tile to ATTACK with team %d units (tick %d)" % [edit_team, action_tick])


func _complete_pending(x: int, y: int) -> void:
	var kind = pending_action["type"]
	var ids := _unit_ids_for_team(edit_team)
	if ids.is_empty():
		_log("No team-%d units to issue the action" % edit_team)
		pending_action = null
		return
	if kind == "move":
		actions.append({"tick": action_tick, "type": "move", "unitIds": ids, "target": [x, y], "formation": "box"})
	elif kind == "attack":
		var tid := _entity_id_at(x, y)
		if tid == "":
			_log("No entity at (%d,%d) to attack" % [x, y])
			pending_action = null
			return
		actions.append({"tick": action_tick, "type": "attack", "unitIds": ids, "targetId": tid})
	pending_action = null
	_log("Added action: " + JSON.stringify(actions[-1]))


func _unit_ids_for_team(team: int) -> Array:
	var ids := []
	for u in scenario_units:
		if int(u["team"]) == team:
			ids.append(u["id"])
	return ids


func _entity_id_at(x: int, y: int) -> String:
	for u in scenario_units:
		var p = u["position"]
		if int(p[0]) == x and int(p[1]) == y:
			return String(u["id"])
	for b in scenario_buildings:
		var p = b["position"]
		if int(p[0]) == x and int(p[1]) == y:
			return String(b["id"])
	return ""


# ---- Load / Save / Export / Run -------------------------------------------

func _on_new() -> void:
	tiles.clear()
	spawn_points.clear()
	scenario_units.clear()
	scenario_buildings.clear()
	actions.clear()
	next_id = 1
	pending_action = null
	queue_redraw()
	_log("New scenario cleared")


func _on_load_map() -> void:
	var path := maps_dir.path_join(map_name_edit.text)
	var txt := FileAccess.get_file_as_string(path)
	if txt == "":
		_log("Could not read map: " + path)
		return
	var data = JSON.parse_string(txt)
	if typeof(data) != TYPE_DICTIONARY:
		_log("Invalid JSON in map: " + path)
		return
	map_width = int(data.get("width", map_width))
	map_height = int(data.get("height", map_height))
	tiles.clear()
	spawn_points.clear()
	var terrain: Dictionary = data.get("terrain", {})
	for t in terrain.get("tiles", []):
		var key := "%d,%d" % [int(t["x"]), int(t["y"])]
		tiles[key] = {"type": String(t.get("type", "rock")), "passable": bool(t.get("passable", false)), "height": float(t.get("height", 0))}
	for sp in data.get("spawnPoints", []):
		spawn_points.append({"team": int(sp.get("team", 0)), "x": int(sp.get("x", 0)), "y": int(sp.get("y", 0)), "radius": int(sp.get("radius", 3))})
	queue_redraw()
	_log("Loaded %s (%dx%d, %d tiles)" % [path, map_width, map_height, tiles.size()])


func _on_save_map() -> void:
	var terrain_tiles := []
	for key in tiles.keys():
		var parts = key.split(",")
		var t: Dictionary = tiles[key]
		terrain_tiles.append({"x": int(parts[0]), "y": int(parts[1]), "type": t["type"], "passable": t["passable"], "height": t["height"]})
	var map_json := {
		"version": "1.0",
		"name": map_name_edit.text.trim_suffix(".json"),
		"width": map_width,
		"height": map_height,
		"tileSize": 32,
		"terrain": {"default": {"type": "grass", "passable": true, "height": 0, "visual": "grass"}, "tiles": terrain_tiles},
		"features": [],
		"spawnPoints": spawn_points,
		"metadata": {"author": "Visual Editor"},
	}
	var path := maps_dir.path_join(map_name_edit.text)
	_write_json(path, map_json)


func _on_export_scenario() -> void:
	var scen := {
		"name": scenario_name_edit.text.trim_suffix(".json"),
		"map": map_name_edit.text,
		"description": "Created with the Phase 5 visual editor",
		"setup": {"units": scenario_units, "buildings": scenario_buildings},
		"actions": actions,
		"expectations": {"maxTicks": 300, "finalState": {"units": [], "buildings": []}},
	}
	var path := maps_dir.path_join("scenarios").path_join(scenario_name_edit.text)
	_write_json(path, scen)


func _on_run_scenario() -> void:
	var name := scenario_name_edit.text.trim_suffix(".json")
	# `go -C <dir>` runs the command in the server module directory (Go 1.20+).
	var args := ["-C", server_dir, "test", "-run", "TestAllScenarios/" + name, "-v"]
	var out: Array = []
	var code := OS.execute(go_edit.text, args, out, true)
	_log("go test exit=%d\n%s" % [code, "\n".join(out)])


func _write_json(path: String, obj) -> void:
	var f := FileAccess.open(path, FileAccess.WRITE)
	if f == null:
		_log("Failed to open for write: %s (err=%s)" % [path, str(FileAccess.get_open_error())])
		return
	f.store_string(JSON.stringify(obj, "  "))
	f.close()
	_log("Wrote " + path)


# ---- Drawing ---------------------------------------------------------------

func _draw() -> void:
	for y in range(map_height):
		for x in range(map_width):
			var key := "%d,%d" % [x, y]
			var col := Color(0.20, 0.42, 0.22)
			if tiles.has(key):
				col = Color(0.40, 0.40, 0.42)
			var origin := ORIGIN + Vector2(x * TILE_PX, y * TILE_PX)
			draw_rect(Rect2(origin, Vector2(TILE_PX - 1, TILE_PX - 1)), col)

	for sp in spawn_points:
		var c := ORIGIN + Vector2(int(sp["x"]) * TILE_PX + TILE_PX * 0.5, int(sp["y"]) * TILE_PX + TILE_PX * 0.5)
		draw_circle(c, TILE_PX * 0.33, Color(0.25, 0.45, 1.0))

	for u in scenario_units:
		var p = u["position"]
		var c := ORIGIN + Vector2(int(p[0]) * TILE_PX + TILE_PX * 0.5, int(p[1]) * TILE_PX + TILE_PX * 0.5)
		var col := Color(1.0, 0.82, 0.2)
		if int(u["team"]) != 0:
			col = Color(1.0, 0.35, 0.35)
		if String(u["type"]) == "soldier":
			col = col.darkened(0.25)
		var half := TILE_PX * 0.33
		draw_rect(Rect2(c - Vector2(half, half), Vector2(half * 2 - 2, half * 2 - 2)), col)

	for b in scenario_buildings:
		var p = b["position"]
		var c := ORIGIN + Vector2(int(p[0]) * TILE_PX, int(p[1]) * TILE_PX)
		draw_rect(Rect2(c, Vector2(TILE_PX * 2 - 2, TILE_PX * 2 - 2)), Color(0.62, 0.35, 0.12))


# ---- Small UI helpers ------------------------------------------------------

func _label(txt: String) -> Label:
	var l := Label.new()
	l.text = txt
	return l


func _add_button(parent: Node, text: String, cb: Callable) -> void:
	var b := Button.new()
	b.text = text
	parent.add_child(b)
	b.pressed.connect(cb)


func _add_tool_button(parent: Node, text: String, t: int) -> void:
	var b := Button.new()
	b.text = text
	parent.add_child(b)
	b.pressed.connect(_select_tool.bind(t, text))


func _select_tool(t: int, label: String) -> void:
	tool = t
	_log("Tool: " + label)


func _on_team_changed(v) -> void:
	edit_team = int(v)


func _on_tick_changed(v) -> void:
	action_tick = int(v)


func _log(msg: String) -> void:
	if output_edit != null:
		output_edit.append_text(msg + "\n")
	else:
		print(msg)
