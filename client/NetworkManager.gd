extends Node

signal connected_to_server(client_id: int, tick_rate: int)
signal snapshot_received(snapshot: Dictionary)
signal disconnected_from_server()

var udp_socket: PacketPeerUDP
var server_address: String = "127.0.0.1"
var server_port: int = 8080
var is_connected: bool = false
var client_id: int = -1
var tick_rate: int = 20
var current_tick: int = 0
var sequence: int = 0
var heartbeat_interval: float = 2.0  # seconds
var heartbeat_timer: float = 0.0

func _ready():
	udp_socket = PacketPeerUDP.new()
	udp_socket.bind(0)  # Bind to any available port
	set_process(true)

func connect_to_server(player_name: String):
	print("Connecting to server at %s:%d" % [server_address, server_port])
	udp_socket.connect_to_host(server_address, server_port)

	# Send hello message
	var hello_msg = {
		"type": "hello",
		"data": {
			"clientVersion": "1.0",
			"playerName": player_name
		}
	}
	send_message(hello_msg)

func send_message(message: Dictionary):
	var json_string = JSON.stringify(message)
	var buffer = json_string.to_utf8_buffer()
	udp_socket.put_packet(buffer)

func send_input(commands: Array):
	if not is_connected:
		return

	sequence += 1
	var input_msg = {
		"type": "input",
		"data": {
			"tick": current_tick,
			"clientId": client_id,
			"sequence": sequence,
			"commands": commands
		}
	}
	send_message(input_msg)

func send_ping():
	if not is_connected:
		return

	var ping_msg = {
		"type": "ping",
		"data": {}
	}
	send_message(ping_msg)

func _process(delta):
	# Handle heartbeat
	if is_connected:
		heartbeat_timer += delta
		if heartbeat_timer >= heartbeat_interval:
			heartbeat_timer = 0.0
			send_ping()

	# Check for incoming packets
	while udp_socket.get_available_packet_count() > 0:
		var packet = udp_socket.get_packet()
		var json_string = packet.get_string_from_utf8()
		var json = JSON.new()
		var parse_result = json.parse(json_string)

		if parse_result == OK:
			var message = json.data
			handle_message(message)

func handle_message(message: Dictionary):
	match message.get("type", ""):
		"welcome":
			handle_welcome(message.get("data", {}))
		"snapshot":
			handle_snapshot(message.get("data", {}))
		"pong":
			pass  # Handle ping/pong if needed

func handle_welcome(data: Dictionary):
	client_id = data.get("clientId", -1)
	tick_rate = data.get("tickRate", 20)
	var heartbeat_ms = data.get("heartbeatInterval", 2000)
	heartbeat_interval = heartbeat_ms / 1000.0  # Convert to seconds
	is_connected = true
	heartbeat_timer = 0.0  # Reset timer
	print("Connected! Client ID: %d, Tick Rate: %d, Heartbeat: %.1fs" % [client_id, tick_rate, heartbeat_interval])
	connected_to_server.emit(client_id, tick_rate)

func handle_snapshot(data: Dictionary):
	current_tick = data.get("tick", 0)
	snapshot_received.emit(data)

func disconnect_from_server():
	is_connected = false
	client_id = -1
	udp_socket.close()
	disconnected_from_server.emit()