package vehicles

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
)

type Vehicle struct {
	CarID                 int
	Model                 string
	Manufacturer          string
	Year                  int
	Category              string
	CarType               string
	Drivetrain            string
	Aspiration            string
	EngineLayout          string
	EngineCylinderAngle   float32
	EngineCrankPlaneAngle float32
	OpenCockpit           bool
}

type Inventory struct {
	db map[string]Vehicle
}

//go:embed inventory.json
var baseInventoryJSON []byte

func NewInventory(inventoryJSON []byte) (*Inventory, error) {
	inventory := Inventory{}

	if inventoryJSON == nil {
		inventoryJSON = baseInventoryJSON
	}

	err := json.Unmarshal([]byte(inventoryJSON), &inventory.db)
	if err != nil {
		return &Inventory{}, fmt.Errorf("unmarshall inventory JSON: %w", err)
	}

	return &inventory, nil
}

func (i *Inventory) GetVehicleByID(id int) (Vehicle, error) {
	vehicle, ok := i.db[strconv.Itoa(id)]
	if !ok {
		return Vehicle{}, fmt.Errorf("no vehicle found with id %d", id)
	}

	return vehicle, nil
}

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
