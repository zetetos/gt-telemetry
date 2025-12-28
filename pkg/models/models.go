package models

import (
	"fmt"
)

type Name string

const (
	Standard  Name = "A" // Original telemetry format
	Addendum1 Name = "B" // Adds steering wheel data and translational envelope
	Addendum2 Name = "~" // Adds throttle input and brake output data and more (unknown)
)

type GameState int

const (
	GameStateUnknown GameState = iota
	GameStateMainMenu
	GameStateRaceMenu
	GameStateLive
	GameStateReplay
)

type RaceType int

const (
	RaceTypeUnknown RaceType = iota
	RaceTypeSprint
	RaceTypeEndurance
	RaceTypeTimeTrial
)

type CoordinateType int

const (
	CoordinateTypeStartLine CoordinateType = iota
	CoordinateTypeCircuit
)

// Coordinate represents a coordinate in 3D space.
type Coordinate struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
}

// CoordinateNorm is a normalised, reduced precision coordinate in 3D space
// Primarily used for location matching.
type CoordinateNorm struct {
	X int16 `json:"X"`
	Y int16 `json:"Y"`
	Z int16 `json:"Z"`
}

// CornerSet represents individual values at each corner or wheel of a vehicle.
type CornerSet struct {
	FrontLeft  float32 `json:"FrontLeft"`
	FrontRight float32 `json:"FrontRight"`
	RearLeft   float32 `json:"RearLeft"`
	RearRight  float32 `json:"RearRight"`
}

// RotationalEnvelope represents the rotational orientation of a body.
type RotationalEnvelope struct {
	Pitch float32 `json:"Pitch"`
	Yaw   float32 `json:"Yaw"`
	Roll  float32 `json:"Roll"`
}

// TranslationalEnvelope represents the acceleration of a body along 3 axes.
type TranslationalEnvelope struct {
	Sway  float32 `json:"Sway"`
	Heave float32 `json:"Heave"`
	Surge float32 `json:"Surge"`
}

// Vector represents the velocity of a body along 3 axes.
type Vector struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
}

// Normalise normalises the Coordinate to reduce precision for location matching.
func (c *Coordinate) Normalise(resX, resY, resZ int16) CoordinateNorm {
	return CoordinateNorm{
		X: int16(c.X/float32(resX)) * resX,
		Y: int16(c.Y/float32(resY)) * resY,
		Z: int16(c.Z/float32(resZ)) * resZ,
	}
}

// String returns a string representation of the CoordinateNorm of the form "x:<X>,y:<Y>,z:<Z>".
func (c *CoordinateNorm) String() string {
	return fmt.Sprintf("x:%d,y:%d,z:%d", c.X, c.Y, c.Z)
}
