package vehicles

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var ErrVehicleNotFound = errors.New("no vehicle found with id")

// Vehicle represents information about a specific vehicle.
type Vehicle struct {
	CarID                 int     `csv:"CarId"                 json:"carId"`
	Manufacturer          string  `csv:"Manufacturer"          json:"manufacturer"`
	Model                 string  `csv:"Model"                 json:"model"`
	Year                  int     `csv:"Year"                  json:"year"`
	OpenCockpit           bool    `csv:"OpenCockpit"           json:"openCockpit"`
	CarType               string  `csv:"CarType"               json:"carType"`
	Category              string  `csv:"Category"              json:"category"`
	Drivetrain            string  `csv:"Drivetrain"            json:"drivetrain"`
	Aspiration            string  `csv:"Aspiration"            json:"aspiration"`
	Length                int     `csv:"Length"                json:"length"`
	Width                 int     `csv:"Width"                 json:"width"`
	Height                int     `csv:"Height"                json:"height"`
	Wheelbase             int     `csv:"Wheelbase"             json:"wheelbase"`
	TrackFront            int     `csv:"TrackFront"            json:"trackFront"`
	TrackRear             int     `csv:"TrackRear"             json:"trackRear"`
	EngineLayout          string  `csv:"EngineLayout"          json:"engineLayout"`
	EngineBankAngle       float32 `csv:"EngineBankAngle"       json:"engineBankAngle"`
	EngineCrankPlaneAngle float32 `csv:"EngineCrankPlaneAngle" json:"engineCrankPlaneAngle"`
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
