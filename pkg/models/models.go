package models

import (
	"fmt"
)

type Name string

const (
	Unknown   Name = "unknown"
	Standard  Name = "A" // Original telemetry format
	Addendum1 Name = "B" // Adds steering wheel data and translational envelope
	Addendum2 Name = "~" // Adds throttle input and brake output data and more (unknown)
	Addendum3 Name = "C" // Adds ? TODO: determine new fields
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

type SurfaceType int

const (
	SurfaceTypeUnknown SurfaceType = iota
	SurfaceTypeTarmac
	SurfaceTypeConcrete
	SurfaceTypeGrass
	SurfaceTypeDirt
	SurfaceTypeSand
	SurfaceTypeSnow
)

var surfaceTypeName = map[SurfaceType]string{ //nolint:gochecknoglobals // helper for string representation of SurfaceType
	SurfaceTypeUnknown:  "unknown",
	SurfaceTypeTarmac:   "tarmac",
	SurfaceTypeConcrete: "concrete",
	SurfaceTypeGrass:    "grass",
	SurfaceTypeDirt:     "dirt",
	SurfaceTypeSand:     "sand",
	SurfaceTypeSnow:     "snow",
}

var surfaceTypeIDs = map[string]SurfaceType{ //nolint:gochecknoglobals // helper for parsing SurfaceType from telemetry
	"T": SurfaceTypeTarmac,
	"C": SurfaceTypeConcrete,
	"G": SurfaceTypeGrass,
	"D": SurfaceTypeDirt,
	"S": SurfaceTypeSand,
	"s": SurfaceTypeSnow,
}

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

type CornerSetValue interface {
	float32 | ~int
}

// CornerSetGeneric represents individual generic values at each corner or wheel of a vehicle.
type CornerSetGeneric[T CornerSetValue] struct {
	FrontLeft  T `json:"frontLeft"`
	FrontRight T `json:"frontRight"`
	RearLeft   T `json:"rearLeft"`
	RearRight  T `json:"rearRight"`
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

// String returns a string representation of the SurfaceType.
func (s *SurfaceType) String() string {
	if name, ok := surfaceTypeName[*s]; ok {
		return name
	}

	return surfaceTypeName[SurfaceTypeUnknown]
}

// SurfaceTypeFromID converts a surface type character representation to a SurfaceType.
// It returns SurfaceTypeUnknown if the string is not recognised.
// Valid character values are "T" for tarmac, "C" for concrete, "G" for grass, "D" for dirt, and "S" for sand.
func SurfaceTypeFromID(id string) SurfaceType {
	if surfaceType, ok := surfaceTypeIDs[id]; ok {
		return surfaceType
	}

	return SurfaceTypeUnknown
}
