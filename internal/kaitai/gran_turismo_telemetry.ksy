meta:
  id: gran_turismo_telemetry
  title: Gran Turismo Telemetry
  license: CC0-1.0
  endian: le
  bit-endian: le
seq:
  - id: header
    type: header
    -doc: File header section
  - id: map_position_coordinates
    type: coordinate
    -doc: Positional coordinates of vehicle on map in meters
  - id: velocity_vector
    type: vector
    -doc: Vehicle velocity vector in meters per second
  - id: rotational_envelope
    type: rotational_envelope
    -doc: Body rotation around axes (-1 to 1)
  - id: heading
    type: f4
    -doc: Orientation to North from 0.0(south) to 1.0(north).
  - id: angular_velocity_vector
    type: vector
    -doc: Angular velocity vector in radians per second (-1 to +1)
  - id: ride_height
    type: f4
    -doc: Vehicle ride height in meters
  - id: engine_rpm
    type: f4
    -doc: Engine speed in RPM
  - id: oiv
    type: f4
    -doc: Seed value for Salsa20 cipher, ignored
  - id: fuel_level
    type: f4
    -doc: Fuel remaining (0.0 to 1.0)
  - id: fuel_capacity
    type: f4
    -doc: Total fuel capacity (0.0 to 1.0)
  - id: ground_speed
    type: f4
    -doc: Vehicle ground speed in meters per second
  - id: manifold_pressure
    type: f4
    -doc: Manifold pressure in bar, only populated when turbo present (subtract 1 for boost pressure as negative values report vacuum)
  - id: oil_pressure
    type: f4
    -doc: Oil pressure
  - id: water_temperature
    type: f4
    -doc: Water temperature in celsius
  - id: oil_temperature
    type: f4
    -doc: Oil temperature in celsius
  - id: tyre_temperature
    type: corner_set
    -doc: Tyre temperatures in celsius
  - id: sequence_id
    type: u4
    -doc: Packet sequence ID
  - id: current_lap
    type: u2
    -doc: Current lap number
  - id: race_laps
    type: u2
    -doc: Total laps in race
  - id: best_laptime
    type: s4
    -doc: Personal best lap time for this session in milliseconds (-1ms when not set)
  - id: last_laptime
    type: s4
    -doc: Last lap time in milliseconds (-1ms when not set)
  - id: time_of_day
    type: u4
    -doc: Current time of day on track in milliseconds
  - id: starting_position
    type: s2
    -doc: Starting position at the beginning of the race (-1 when race starts)
  - id: race_entrants
    type: s2
    -doc: Total number of entrants at the beginning of the race (-1 when race starts)
  - id: rev_light_rpm_min
    type: u2
    -doc: Minimum engine RPM at which the shift light activates
  - id: rev_light_rpm_max
    type: u2
    -doc: Maximum engine RPM at which the shift light activates
  - id: calculated_max_speed
    type: u2
    -doc: Calculated maximum speed of the vehicle in kilometers per hour
  - id: flags
    type: flags
    -doc: Various flags for the current state of play and instrument cluster lights
  - id: transmission_gear
    type: transmission_gear
    -doc: Transmission gear selection
  - id: throttle_output
    type: u1
    -doc: Throttle output value after TCS applied (0 to 255)
  - id: brake_input
    type: u1
    -doc: Brake input value from controller (0 to 255)
  - id: ignore_1
    size: 1
    -doc: Field 0x93 is empty and ignored
  - id: road_plane_vector
    type: vector
    -doc: Road plane vector
  - id: road_plane_distance
    type: u4
    -doc: Road plane distance
  - id: wheel_radians_per_second
    type: corner_set
    -doc: Individual wheel rotational speed in radians per second
  - id: tyre_radius
    type: corner_set
    -doc: Individual tyre radius in meters
  - id: suspension_height
    type: corner_set
    -doc: Individual suspension height at each corner in meters
  - id: reserved
    size: 32
    -doc: Reserved data, currently unused
  - id: clutch_actuation
    type: f4
    -doc: Clutch actuation (0.0 to 1.0)
  - id: clutch_engagement
    type: f4
    -doc: Clutch engagement (0.0 to 1.0)
  - id: cluch_output_rpm
    type: f4
    -doc: Rotational speed on the output side of the clutch in rpm
  - id: transmission_top_speed_ratio
    type: f4
    -doc: Ratio between vehicle top speed and wheel rotation speed (can calculate rpm at top speed and differential ratio)
  - id: transmission_gear_ratio
    type: gear_ratio
    -doc: Gear ratios for each gear in the transmission
  - id: vehicle_id
    type: u4
    -doc: ID of the vehicle
  - id: steering_wheel_angle_radians
    type: f4
    if: has_section_b
    -doc: Steering wheel angular position in radians
  - id: steering_wheel_force_feedback
    type: f4
    if: has_section_b
    -doc: Steering wheel force feedback signal (unverified)
  - id: translational_envelope
    type: translational_envelope
    if: has_section_b
    -doc: Body forces along axes (-1 to 1)
  - id: throttle_input
    type: u1
    if: has_section_tilde
    -doc: Throttle input value from controller (0-255)
  - id: brake_output
    type: u1
    if: has_section_tilde
    -doc: Brake output value after ABS applied (0-255) (live sessions only)
  - id: unknown0x13e
    type: u1
    if: has_section_tilde
    -doc: Unknown value, possibly bitfield related to EV
  - id: unknown0x13f
    type: u1
    if: has_section_tilde
    -doc: Unknown value, possibly bitfield
  - id: unknown0x140
    type: f4
    if: has_section_tilde
    -doc: Unknown value, possibly FL torque vectoring
  - id: unknown0x144
    type: f4
    if: has_section_tilde
    -doc: Unknown value, possibly FR torque vectoring
  - id: unknown0x148
    type: f4
    if: has_section_tilde
    -doc: Unknown value, possibly RL torque vectoring
  - id: unknown0x14c
    type: f4
    if: has_section_tilde
    -doc: Unknown value, possibly RR torque vectoring
  - id: energy_recovery
    type: f4
    if: has_section_tilde
    -doc: Energy recovery value in FIXME
  - id: unknown0x154
    type: f4
    if: has_section_tilde
    -doc: Unknown value, something related to vehicle motion
types:
  header:
    doc: |
      Magic file header
      0x30 0x53 0x37 0x47 = GT Sport and GT7
      0x47 0x37 0x53 0x30 = GT6
    seq:
      - id: magic
        type: u4
        valid:
          any-of:
          - 810760007
          - 1194808112

  vector:
    doc: 3D direction vector
    seq:
      - id: vector_x
        type: f4
      - id: vector_y
        type: f4
      - id: vector_z
        type: f4
  coordinate:
    doc: 3D position coordinates
    seq:
      - id: coordinate_x
        type: f4
      - id: coordinate_y
        type: f4
      - id: coordinate_z
        type: f4
  translational_envelope:
    doc: 6DOF translational envelope
    seq:
      - id: sway
        type: f4
      - id: heave
        type: f4
      - id: surge
        type: f4
  rotational_envelope:
    doc: 6DOF rotational envelope
    seq:
      - id: pitch
        type: f4
      - id: yaw
        type: f4
      - id: roll
        type: f4
  corner_set:
    doc: Data set representing each wheel or suspension component at the corners of the vehicle
    seq:
      - id: front_left
        type: f4
      - id: front_right
        type: f4
      - id: rear_left
        type: f4
      - id: rear_right
        type: f4
  flags:
    doc: Various flags for the current state of play and instrument cluster indicators
    seq:
      - id: live
        type: b1
      - id: game_paused
        type: b1
      - id: loading
        type: b1
      - id: in_gear
        type: b1
      - id: has_turbo
        type: b1
      - id: rev_limiter_alert
        type: b1
      - id: hand_brake_active
        type: b1
      - id: headlights_active
        type: b1
      - id: high_beam_active
        type: b1
      - id: low_beam_active
        type: b1
      - id: asm_active
        type: b1
      - id: tcs_active
        type: b1
      - id: flag13
        type: b1
      - id: flag14
        type: b1
      - id: flag15
        type: b1
      - id: flag16
        type: b1
  transmission_gear:
    doc: |
      Transmission gear selection information.
      0 = reverse, 15 = neutral
    seq:
      - id: current
        type: b4
      - id: suggested
        type: b4
  gear_ratio:
    doc: Gear ratios for each gear in the transmission
    seq:
      - id: gear
        type: f4
        repeat: expr
        repeat-expr: 8
instances:
  packet_size:
    doc: The total size in bytes of the telemetry packet
    value: _io.size
  header_is_gt6:
    doc: True when the telemetry data is sent from Gran Turismo 6
    value: header.magic == 810760007
  header_is_gt7:
    doc: True when the telemetry data is sent from Gran Turismo 7 or Sport
    value: header.magic == 1194808112
  has_section_a:
    doc: True when the telemetry data contains data requested with format "A"
    value: _io.size >= 296
  has_section_b:
    doc: True when the telemetry data contains data requested with format "B"
    value: _io.size > 296
  has_section_tilde:
    doc: True when the telemetry data contains data requested with format "~"
    value: _io.size > 316