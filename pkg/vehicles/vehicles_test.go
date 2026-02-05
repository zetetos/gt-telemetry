package vehicles_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

type VehiclesTestSuite struct {
	suite.Suite
}

func TestVehiclesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VehiclesTestSuite))
}

func (suite *VehiclesTestSuite) TestEmptyJSONParameterFallsBackToBaseInventory() {
	// Arrange
	wantValue := vehicles.Vehicle{
		CarID:                 -10000,
		Model:                 "Dummy Model",
		Manufacturer:          "Dummy Manufacturer",
		Category:              "",
		Drivetrain:            "-",
		Aspiration:            "-",
		EngineLayout:          "-",
		EngineBankAngle:       0,
		EngineCrankPlaneAngle: 0,
		Year:                  0,
		OpenCockpit:           false,
		CarType:               "street",
		Length:                0,
		Width:                 0,
		Height:                0,
		Wheelbase:             0,
		TrackFront:            0,
		TrackRear:             0,
	}

	var inventoryJSON []byte

	// Act
	inventory, err := vehicles.NewDB(inventoryJSON)
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
			"CarType": "",
			"Length": 0,
			"Width": 0,
			"Height": 0,
			"Wheelbase": 0,
			"TrackFront": 0,
			"TrackRear": 0
		}
	}`)

	// Act
	_, err := vehicles.NewDB(inventoryJSON)

	// Assert
	suite.Require().NoError(err)
}

func (suite *VehiclesTestSuite) TestInvalidJSONParameterReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{
		not_an_int: {}
	}`)

	// Act
	_, err := vehicles.NewDB(inventoryJSON)

	// Assert
	suite.ErrorContains(err, "unmarshall vehicle inventory JSON")
}

func (suite *VehiclesTestSuite) TestGetVehicleIDWithInvalidIDReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{}`)

	// Act
	inventory, err := vehicles.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	_, err = inventory.GetVehicleByID(1)

	// Assert
	suite.ErrorContains(err, "no vehicle found with id")
}

func (suite *VehiclesTestSuite) TestGetVehicleWIthValidIDReturnsVehicle() {
	// Arrange
	wantValue := vehicles.Vehicle{
		CarID:                 1234,
		Model:                 "Dummy Model",
		Manufacturer:          "Dummy Manufacturer",
		Category:              "",
		Drivetrain:            "-",
		Aspiration:            "-",
		EngineLayout:          "-",
		EngineBankAngle:       0,
		EngineCrankPlaneAngle: 0,
		Year:                  0,
		OpenCockpit:           false,
		CarType:               "street",
		Length:                4500,
		Width:                 1800,
		Height:                1300,
		Wheelbase:             2700,
		TrackFront:            1550,
		TrackRear:             1600,
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
			"CarType": "street",
			"Length": 4500,
			"Width": 1800,
			"Height": 1300,
			"Wheelbase": 2700,
			"TrackFront": 1550,
			"TrackRear": 1600
		}
	}`)

	// Act
	inventory, err := vehicles.NewDB(inventoryJSON)
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
			vehicle := vehicles.Vehicle{
				Aspiration: shortName,
			}

			// Act
			gotValue := vehicle.ExpandedAspiration()

			// Assert
			suite.Equal(wantValue, gotValue)
		})
	}
}
