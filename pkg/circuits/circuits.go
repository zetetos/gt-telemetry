package circuits

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/zetetos/gt-telemetry/pkg/models"
)

const (
	circuitCoorindateResolutionX = 16
	circuitCoorindateResolutionY = 2
	circuitCoorindateResolutionZ = 16

	startLineCoorindateResolutionX = 16
	startLineCoorindateResolutionY = 2
	startLineCoorindateResolutionZ = 16
)

// CircuitInfo represents information about a specific race circuit
type CircuitInfo struct {
	ID                    string                `json:"id"`
	Name                  string                `json:"name"`
	Variation             string                `json:"variation"`
	Country               string                `json:"country"`
	Default               bool                  `json:"default"`
	Length                int                   `json:"length"`
	StartLine             models.CoordinateNorm `json:"startline"`
	UniqueCoordinateCount int                   `json:"unique_coordinate_count"`
}

// CircuitInventory represents the complete JSON structure from the embedded circuit inventory data
type CircuitInventory struct {
	Coordinates map[string]string      `json:"coordinates"`
	StartLines  map[string][]string    `json:"-"`
	Circuits    map[string]CircuitInfo `json:"circuits"`
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
		key := CoordinateNormToKey(circuit.StartLine)
		inventory.StartLines[key] = append(inventory.StartLines[key], circuit.ID)
	}

	return &CircuitDB{
		inventory: &inventory,
	}, nil
}

// GetCircuitAtCoordinate returns the circuit at a given coordinate (single value)
func (c *CircuitDB) GetCircuitAtCoordinate(coordinate models.Coordinate, coordType models.CoordinateType) (circuitID string, found bool) {
	if c.inventory == nil {
		return "", false
	}

	var normalisedPos models.CoordinateNorm
	if coordType == models.CoordinateTypeStartLine {
		normalisedPos = NormaliseStartLineCoordinate(coordinate)
	} else {
		normalisedPos = NormaliseCircuitCoordinate(coordinate)
	}
	key := CoordinateNormToKey(normalisedPos)

	if coordType == models.CoordinateTypeStartLine {
		circuitIDs, found := c.inventory.StartLines[key]
		if found && len(circuitIDs) == 1 {
			return circuitIDs[0], true
		}
		return "", false
	} else {
		circuitID, found := c.inventory.Coordinates[key]
		return circuitID, found
	}
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

// NormaliseStartLineCoordinate normalises a start line coordinate to reduce precision for location matching
func NormaliseStartLineCoordinate(coordinate models.Coordinate) (normalised models.CoordinateNorm) {
	// The FIA rules state that the starting grid has a min width of 15 meters.
	// 32m resolution should provide sufficient accuracy for most tracks.
	return models.CoordinateNorm{
		X: int16(coordinate.X/startLineCoorindateResolutionX) * startLineCoorindateResolutionX,
		Y: int16(coordinate.Y/startLineCoorindateResolutionY) * startLineCoorindateResolutionY,
		Z: int16(coordinate.Z/startLineCoorindateResolutionZ) * startLineCoorindateResolutionZ,
	}
}

// NormaliseCircuitCoordinate normalises a circuit coordinate to reduce precision for location matching
func NormaliseCircuitCoordinate(coordinate models.Coordinate) (normalised models.CoordinateNorm) {
	// Track map resultion is lower to reduce file size.
	// 64m resolution should be sufficient for most tracks.
	// Y (vertical) resolution is higher since elevation changes are much smaller than X/Z.
	return models.CoordinateNorm{
		X: int16(coordinate.X/circuitCoorindateResolutionX) * circuitCoorindateResolutionX,
		Y: int16(coordinate.Y/circuitCoorindateResolutionY) * circuitCoorindateResolutionY,
		Z: int16(coordinate.Z/circuitCoorindateResolutionZ) * circuitCoorindateResolutionZ,
	}
}

func CoordinateNormToKey(normalised models.CoordinateNorm) string {
	return fmt.Sprintf("x:%d,y:%d,z:%d", normalised.X, normalised.Y, normalised.Z)
}
