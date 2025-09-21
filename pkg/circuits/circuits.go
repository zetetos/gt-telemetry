package circuits

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Coordinate struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
	Z int16 `json:"z"`
}

// CircuitInfo represents information about a specific race circuit
type CircuitInfo struct {
	ID        string
	Name      string
	Region    string
	Length    int
	StartLine Coordinate
}

// CircuitInventory represents the complete JSON structure from the embedded circuit inventory data
type CircuitInventory struct {
	Coordinates map[string][]string
	StartLines  map[string][]string
	Circuits    map[string]CircuitInfo
}

// CircuitDB provides an object and methods to access circuit information from the embedded inventory
type CircuitDB struct {
	inventory *CircuitInventory
}

//go:embed circuits.json
var baseInventoryJSON []byte

// newDB creates a new CircuitDB instance by loading the circuit inventory from embedded JSON data
func NewDB(inventoryJSON []byte) (*CircuitDB, error) {
	inventory := CircuitInventory{}

	if inventoryJSON == nil {
		inventoryJSON = baseInventoryJSON
	}

	err := json.Unmarshal([]byte(inventoryJSON), &inventory)
	if err != nil {
		return &CircuitDB{}, fmt.Errorf("unmarshall circuit inventory JSON: %w", err)
	}

	// Populate start line lookup tables
	inventory.StartLines = make(map[string][]string)
	for _, circuit := range inventory.Circuits {
		normalisedCoordinate := NormaliseStartLineCoordinate(circuit.StartLine)
		key := CoordinateToKey(normalisedCoordinate)

		inventory.StartLines[key] = append(inventory.StartLines[key], circuit.ID)
	}

	return &CircuitDB{
		inventory: &inventory,
	}, nil
}

// GetCircuitsAtCoordinate returns the list of circuits at a given coordinate
func (c *CircuitDB) GetCircuitsAtCoordinate(coordinate Coordinate) (circuitIDs []string, found bool) {
	if c.inventory == nil {
		return nil, false
	}

	normalisedPos := NormaliseCircuitCoordinate(coordinate)
	key := CoordinateToKey(normalisedPos)

	circuitIDs, found = c.inventory.Coordinates[key]

	return circuitIDs, found
}

// GetCircuitsAtStartLine returns the list of circuits at a given start line coordinate
func (c *CircuitDB) GetCircuitsAtStartLine(coordinate Coordinate) (circuitIDs []string, found bool) {
	if c.inventory == nil {
		return nil, false
	}

	normalisedPos := NormaliseStartLineCoordinate(coordinate)
	key := CoordinateToKey(normalisedPos)

	circuitIDs, found = c.inventory.StartLines[key]

	return circuitIDs, found
}

// GetCircuitByID retrieves a CircuitInfo by its ID
func (c *CircuitDB) GetCircuitByID(circuitID string) (circuit CircuitInfo, found bool) {
	if c.inventory == nil {
		return CircuitInfo{}, false
	}

	circuit, found = c.inventory.Circuits[circuitID]
	circuit.ID = circuitID

	return circuit, found
}

// GetAllCircuitIDs returns all available circuit IDs
func (c *CircuitDB) GetAllCircuitIDs() (circuitIDs []string) {
	if c.inventory == nil {
		return nil
	}

	circuitIDs = make([]string, 0, len(c.inventory.Circuits))
	for circuitID := range c.inventory.Circuits {
		circuitIDs = append(circuitIDs, circuitID)
	}

	return circuitIDs
}

// GetCircuitsInRegion returns all circuits in a given region
func (c *CircuitDB) GetCircuitsInRegion(region string) (circuits map[string]CircuitInfo) {
	if c.inventory == nil {
		return nil
	}

	circuits = make(map[string]CircuitInfo)
	for circuitID, circuitInfo := range c.inventory.Circuits {
		if circuitInfo.Region == region {
			circuits[circuitID] = circuitInfo
		}
	}

	return circuits
}

// NormaliseStartLineCoordinate normalises a start line coordinate to reduce precision for location matching
func NormaliseStartLineCoordinate(coordinate Coordinate) (normalised struct{ X, Y, Z int16 }) {
	normalised = struct{ X, Y, Z int16 }{
		X: int16(coordinate.X/32) * 32,
		Y: int16(coordinate.Y/4) * 4,
		Z: int16(coordinate.Z/32) * 32,
	}

	return normalised
}

// NormaliseCircuitCoordinate normalises a circuit coordinate to reduce precision for location matching
func NormaliseCircuitCoordinate(coordinate Coordinate) (normalised struct{ X, Y, Z int16 }) {
	normalised = struct{ X, Y, Z int16 }{
		X: int16(coordinate.X/64) * 64,
		Y: int16(coordinate.Y/8) * 8,
		Z: int16(coordinate.Z/64) * 64,
	}

	return normalised
}

func CoordinateToKey(normalised struct{ X, Y, Z int16 }) string {
	return fmt.Sprintf("x:%d,y:%d,z:%d", normalised.X, normalised.Y, normalised.Z)
}
