package vehicles

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrVehicleNotFound = errors.New("no vehicle found with id")
)

// Vehicle represents information about a specific vehicle.
type Vehicle struct {
	CarID                 int     `json:"CarID"`
	Manufacturer          string  `json:"Manufacturer"`
	Model                 string  `json:"Model"`
	Year                  int     `json:"Year"`
	OpenCockpit           bool    `json:"OpenCockpit"`
	CarType               string  `json:"CarType"`
	Category              string  `json:"Category"`
	Drivetrain            string  `json:"Drivetrain"`
	Aspiration            string  `json:"Aspiration"`
	Length                int     `json:"Length"`
	Width                 int     `json:"Width"`
	Height                int     `json:"Height"`
	Wheelbase             int     `json:"Wheelbase"`
	TrackFront            int     `json:"TrackFront"`
	TrackRear             int     `json:"TrackRear"`
	EngineLayout          string  `json:"EngineLayout"`
	EngineBankAngle       float32 `json:"EngineBankAngle"`
	EngineCrankPlaneAngle float32 `json:"EngineCrankPlaneAngle"`
}

// VehicleInventory represents the complete JSON structure from the embedded vehicle inventory data.
type VehicleInventory map[string]Vehicle

// VehicleDB provides an object and methods to access vehicle information from the embedded inventory.
type VehicleDB struct {
	inventory VehicleInventory
}

//go:embed vehicles.json
var baseInventoryJSON []byte

// NewDB creates a new VehicleDB instance by loading the vehicle inventory from embedded JSON data.
func NewDB(inventoryJSON []byte) (*VehicleDB, error) {
	inventory := VehicleInventory{}

	if inventoryJSON == nil {
		inventoryJSON = baseInventoryJSON
	}

	err := json.Unmarshal(inventoryJSON, &inventory)
	if err != nil {
		return &VehicleDB{}, fmt.Errorf("unmarshall vehicle inventory JSON: %w", err)
	}

	return &VehicleDB{
		inventory: inventory,
	}, nil
}

// GetVehicleByID retrieves a Vehicle from the inventory by its CarID.
func (i *VehicleDB) GetVehicleByID(id int) (Vehicle, error) {
	vehicle, ok := i.inventory[strconv.Itoa(id)]
	if !ok {
		return Vehicle{}, fmt.Errorf("%w: %d", ErrVehicleNotFound, id)
	}

	return vehicle, nil
}

// ExpandedAspiration provides a human-readable description of the vehicle's aspiration type.
func (v *Vehicle) ExpandedAspiration() string {
	switch v.Aspiration {
	case "EV":
		return "Electric Vehicle"
	case "NA":
		return "Naturally Aspirated"
	case "TC":
		return "Turbocharged"
	case "SC":
		return "Supercharged"
	case "TC+SC":
		return "Compound Charged"
	default:
		return v.Aspiration
	}
}
