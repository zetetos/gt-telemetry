package vehicles

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type VehiclesTestSuite struct {
	suite.Suite
}

func TestVehiclesTestSuite(t *testing.T) {
	suite.Run(t, new(VehiclesTestSuite))
}

func (suite *VehiclesTestSuite) TestEmptyJSONParameterFallsBackToBaseInventory() {
	// Arrange
	wantValue := Vehicle{
		CarID:                 -10000,
		Model:                 "Dummy Model",
		Manufacturer:          "Dummy Manufacturer",
		Category:              "",
		Drivetrain:            "-",
		Aspiration:            "-",
		EngineLayout:          "-",
		EngineCylinderAngle:   0,
		EngineCrankPlaneAngle: 0,
		Year:                  0,
		OpenCockpit:           false,
		CarType:               "street",
	}

	var inventoryJSON []byte

	// Act
	inventory, err := NewInventory(inventoryJSON)
	suite.Require().NoError(err)

	gotValue, err := inventory.GetVehicleByID(wantValue.CarID)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *VehiclesTestSuite) TestValidJSONParameterCanConstructInventory() {
	// Arrange
	inventoryJSON := []byte(`{
		"0": {
			"Model": "",
			"Manufacturer": "",
			"Category": "",
			"Drivetrain": "",
			"Aspiration": "",
			"EngineLayout": "",
			"EngineAngle": "",
			"Year": 0,
			"CarID": 0,
			"OpenCockpit": false,
			"CarType": ""
		}
	}`)

	// Act
	_, err := NewInventory(inventoryJSON)

	// Assert
	suite.Require().NoError(err)
}

func (suite *VehiclesTestSuite) TestInvalidJSONParameterReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{
		not_an_int: {}
	}`)

	// Act
	_, err := NewInventory(inventoryJSON)

	// Assert
	suite.Assert().ErrorContains(err, "unmarshall inventory JSON")
}

func (suite *VehiclesTestSuite) TestGetVehicleIDWithInvalidIDReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{}`)

	// Act
	inventory, err := NewInventory(inventoryJSON)
	suite.Require().NoError(err)

	_, err = inventory.GetVehicleByID(1)

	// Assert
	suite.Assert().ErrorContains(err, "no vehicle found with id")
}

func (suite *VehiclesTestSuite) TestGetVehicleWIthValidIDReturnsVehicle() {
	// Arrange
	wantValue := Vehicle{
		CarID:                 1234,
		Model:                 "Dummy Model",
		Manufacturer:          "Dummy Manufacturer",
		Category:              "",
		Drivetrain:            "-",
		Aspiration:            "-",
		EngineLayout:          "-",
		EngineCylinderAngle:   0,
		EngineCrankPlaneAngle: 0,
		Year:                  0,
		OpenCockpit:           false,
		CarType:               "street",
	}

	inventoryJSON := []byte(`{
		"1234": {
			"Model": "Dummy Model",
			"Manufacturer": "Dummy Manufacturer",
			"Category": "",
			"Drivetrain": "-",
			"Aspiration": "-",
			"EngineLayout": "-",
			"EngineAngle": "-",
			"Year": 0,
			"CarID": 1234,
			"OpenCockpit": false,
			"CarType": "street"
		}
	}`)

	// Act
	inventory, err := NewInventory(inventoryJSON)
	suite.Require().NoError(err)

	gotValue, err := inventory.GetVehicleByID(wantValue.CarID)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *VehiclesTestSuite) TestShortAspirationExpandsToLongName() {
	tests := map[string]string{
		"NA":      "Naturally Aspirated",
		"TC":      "Turbocharged",
		"SC":      "Supercharged",
		"TC+SC":   "Compound Charged",
		"EV":      "Electric Vehicle",
		"-":       "-",
		"INVALID": "INVALID",
	}

	for shortName, wantValue := range tests {
		suite.Run(shortName, func() {
			// Arrange
			vehicle := Vehicle{
				Aspiration: shortName,
			}

			// Act
			gotValue := vehicle.ExpandedAspiration()

			// Assert
			suite.Equal(wantValue, gotValue)
		})
	}
}
