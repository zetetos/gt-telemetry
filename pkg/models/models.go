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
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// CoordinateNorm is a normalised, reduced precision coordinate in 3D space
// Primarily used for location matching.
type CoordinateNorm struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
	Z int16 `json:"z"`
}

// CornerSet represents individual values at each corner or wheel of a vehicle.
type CornerSet struct {
	FrontLeft  float32 `json:"frontLeft"`
	FrontRight float32 `json:"frontRight"`
	RearLeft   float32 `json:"rearLeft"`
	RearRight  float32 `json:"rearRight"`
}

// RotationalEnvelope represents the rotational orientation of a body.
type RotationalEnvelope struct {
	Pitch float32 `json:"pitch"`
	Yaw   float32 `json:"yaw"`
	Roll  float32 `json:"roll"`
}

// TranslationalEnvelope represents the acceleration of a body along 3 axes.
type TranslationalEnvelope struct {
	Sway  float32 `json:"sway"`
	Heave float32 `json:"heave"`
	Surge float32 `json:"surge"`
}

// Vector represents the velocity of a body along 3 axes.
type Vector struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
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
