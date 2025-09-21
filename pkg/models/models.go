package models

type Name string

const (
	Standard  Name = "A" // Original telemetry format
	Addendum1 Name = "B" // Adds steering wheel data and translational envelope
	Addendum2 Name = "~" // Adds throttle input and brake output data and more (unknown)
)

// Coordinate represents a coordinate in 3D space
type Coordinate struct {
	X float32
	Y float32
	Z float32
}

// CoordinateNorm is a normalised, reduced precision coordinate in 3D space
// Primarily used for location matching
type CoordinateNorm struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
	Z int16 `json:"z"`
}

// CornerSet represents individual values at each corner or wheel of a vehicle
type CornerSet struct {
	FrontLeft  float32
	FrontRight float32
	RearLeft   float32
	RearRight  float32
}

// RotationalEnvelope represents the rotational orientation of a body
type RotationalEnvelope struct {
	Pitch float32
	Yaw   float32
	Roll  float32
}

// TranslationalEnvelope represents the acceleration of a body along 3 axes
type TranslationalEnvelope struct {
	Sway  float32
	Heave float32
	Surge float32
}

// Vector represents the velocity of a body along 3 axes
type Vector struct {
	X float32
	Y float32
	Z float32
}
