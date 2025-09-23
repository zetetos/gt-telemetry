package models

type Name string

const (
	Standard  Name = "A" // Original telemetry format
	Addendum1 Name = "B" // Adds steering wheel data and translational envelope
	Addendum2 Name = "~" // Adds throttle input and brake output data and more (unknown)
)

// Coordinate represents a coordinate in 3D space
type Coordinate struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
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
	FrontLeft  float32 `json:"front_left"`
	FrontRight float32 `json:"front_right"`
	RearLeft   float32 `json:"rear_left"`
	RearRight  float32 `json:"rear_right"`
}

// RotationalEnvelope represents the rotational orientation of a body
type RotationalEnvelope struct {
	Pitch float32 `json:"pitch"`
	Yaw   float32 `json:"yaw"`
	Roll  float32 `json:"roll"`
}

// TranslationalEnvelope represents the acceleration of a body along 3 axes
type TranslationalEnvelope struct {
	Sway  float32 `json:"sway"`
	Heave float32 `json:"heave"`
	Surge float32 `json:"surge"`
}

// Vector represents the velocity of a body along 3 axes
type Vector struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}
