package vehicles

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
)

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
	EngineLayout          string  `json:"EngineLayout"`
	EngineBankAngle       float32 `json:"EngineBankAngle"`
	EngineCrankPlaneAngle float32 `json:"EngineCrankPlaneAngle"`
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
