package gttelemetry_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	gttelemetry "github.com/zetetos/gt-telemetry/v2"
	"github.com/zetetos/gt-telemetry/v2/internal/telemetry"
	"github.com/zetetos/gt-telemetry/v2/pkg/vehicles"
)

type UnitAlternatesTestSuite struct {
	suite.Suite

	transformer *gttelemetry.Transformer
}

func TestUnitAlternatesTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnitAlternatesTestSuite))
}

func (suite *UnitAlternatesTestSuite) SetupTest() {
	inventoryJSON := []byte(`{
		"1234": {
			"Model": "Dummy Model",
			"Manufacturer": "Dummy Manufacturer",
			"Category": "Gr.1",
			"Drivetrain": "FR",
			"Aspiration": "NA",
			"EngineLayout": "V6",
			"EngineBankAngle": 60,
			"EngineCrankPlaneAngle": 120,
			"Year": 2025,
			"CarID": 1234,
			"OpenCockpit": false,
			"CarType": "race",
			"Length": 4500,
			"Width": 1800,
			"Height": 1300,
			"Wheelbase": 2700,
			"TrackFront": 1550,
			"TrackRear": 1600
		}
	}`)
	inventory, _ := vehicles.NewDB(inventoryJSON)
	transformer := gttelemetry.NewTransformer(inventory)
	transformer.RawTelemetry = telemetry.GranTurismoTelemetry{}

	suite.transformer = transformer
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesCurrentGearStringReturnsCorrectValue() {
	wantValues := map[int]string{
		0:  "R",
		1:  "1",
		2:  "2",
		3:  "3",
		4:  "4",
		5:  "5",
		6:  "6",
		7:  "7",
		8:  "8",
		9:  "9",
		10: "10",
		11: "11",
		12: "12",
		13: "13",
		14: "14",
		15: "N",
	}

	for testValue, wantValue := range wantValues {
		suite.Run("Gear"+strconv.Itoa(testValue), func() {
			// Arrange
			suite.transformer.RawTelemetry.TransmissionGear = &telemetry.GranTurismoTelemetry_TransmissionGear{
				Current: uint64(testValue), //nolint:gosec // not an issue as test is between 0 and 15
			}

			// Act
			gotValue := suite.transformer.CurrentGearString()

			// Assert
			suite.Equal(wantValue, gotValue)
		})
	}
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesGroundSpeedKPHReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.GroundSpeed = 82

	// Act
	gotValue := suite.transformer.GroundSpeedKPH()

	// Assert
	suite.InEpsilon(float32(295.19998), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesOilTemperatureFahrenheitReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.OilTemperature = 104.2945

	// Act
	gotValue := suite.transformer.OilTemperatureFahrenheit()

	// Assert
	suite.InEpsilon(float32(219.7301), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesRideHeightMillimetresReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.RideHeight = 0.10267

	// Act
	gotValue := suite.transformer.RideHeightMillimetres()

	// Assert
	suite.InEpsilon(float32(102.67), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesSuspensionHeightFeetReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.0267,
		FrontRight: 0.0213,
		RearLeft:   0.0312,
		RearRight:  0.0298,
	}

	// Act
	gotValue := suite.transformer.SuspensionHeightFeet()

	// Assert
	suite.InEpsilon(float32(0.08759842), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(0.069881886), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(0.1023622), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(0.09776903), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesSuspensionHeightInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.0267,
		FrontRight: 0.0213,
		RearLeft:   0.0312,
		RearRight:  0.0298,
	}

	// Act
	gotValue := suite.transformer.SuspensionHeightInches()

	// Assert
	suite.InEpsilon(float32(1.0511816), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(0.83858305), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(1.2283471), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(1.1732289), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesSuspensionHeightMillimetresReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.0267,
		FrontRight: 0.0213,
		RearLeft:   0.0312,
		RearRight:  0.0298,
	}

	// Act
	gotValue := suite.transformer.SuspensionHeightMillimetres()

	// Assert
	suite.InEpsilon(float32(26.699999), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(21.3), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(31.199999), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(29.8), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTurboBoostPSIReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.ManifoldPressure = 2.13

	// Act
	gotValue := suite.transformer.TurboBoostPSI()

	// Assert
	suite.InEpsilon(float32(16.389261), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTurboBoostInHgReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.ManifoldPressure = 2.13

	// Act
	gotValue := suite.transformer.TurboBoostInHg()

	// Assert
	suite.InEpsilon(float32(33.36888), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTurboBoostKPAReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.ManifoldPressure = 2.13

	// Act
	gotValue := suite.transformer.TurboBoostKPA()

	// Assert
	suite.InEpsilon(float32(113.000015), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreDiameterFeetReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreDiameterFeet()

	// Assert
	suite.InEpsilon(float32(2.0603676), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(2.0603676), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(2.2506561), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(2.2506561), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreDiameterInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreDiameterInches()
	// Assert
	suite.InEpsilon(float32(24.724422), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(24.724422), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(27.007887), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(27.007887), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreDiameterMillimetresReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreDiameterMillimetres()

	// Assert
	suite.InEpsilon(float32(628), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(628), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(686), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(686), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreRadiusFeetReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreRadiusFeet()

	// Assert
	suite.InEpsilon(float32(1.0301838), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(1.0301838), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(1.1253281), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(1.1253281), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreRadiusInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreRadiusInches()

	// Assert
	suite.InEpsilon(float32(12.362211), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(12.362211), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(13.503943), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(13.503943), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreRadiusMillimetresReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.314,
		FrontRight: 0.314,
		RearLeft:   0.343,
		RearRight:  0.343,
	}

	// Act
	gotValue := suite.transformer.TyreRadiusMillimetres()

	// Assert
	suite.InEpsilon(float32(314), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(314), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(343), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(343), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesTyreTemperatureFahrenheitReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreTemperature = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  64.3,
		FrontRight: 64.1,
		RearLeft:   68.2,
		RearRight:  67.8,
	}

	// Act
	gotValue := suite.transformer.TyreTemperatureFahrenheit()

	// Assert
	suite.InEpsilon(float32(143.835), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(143.38762), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(152.55905), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(151.66429), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesWheelSpeedKPHReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.317,
		FrontRight: 0.317,
		RearLeft:   0.317,
		RearRight:  0.317,
	}
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  132.50,
		FrontRight: 132.51,
		RearLeft:   132.45,
		RearRight:  132.40,
	}

	// Act
	gotValue := suite.transformer.WheelSpeedKPH()

	// Assert
	suite.InEpsilon(float32(151.20898), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(151.2204), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(151.15193), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(151.09486), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesWheelSpeedMPHReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.317,
		FrontRight: 0.317,
		RearLeft:   0.317,
		RearRight:  0.317,
	}
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  132.50,
		FrontRight: 132.51,
		RearLeft:   132.45,
		RearRight:  132.40,
	}

	// Act
	gotValue := suite.transformer.WheelSpeedMPH()

	// Assert
	suite.InEpsilon(float32(93.95692), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(93.964005), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(93.92146), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(93.886), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesWheelSpeedRPMReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = &telemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  132.50,
		FrontRight: 132.51,
		RearLeft:   132.45,
		RearRight:  132.40,
	}

	// Act
	gotValue := suite.transformer.WheelSpeedRPM()

	// Assert
	suite.InEpsilon(float32(1265.2817), gotValue.FrontLeft, 1e-5)
	suite.InEpsilon(float32(1265.3772), gotValue.FrontRight, 1e-5)
	suite.InEpsilon(float32(1264.8043), gotValue.RearLeft, 1e-5)
	suite.InEpsilon(float32(1264.3268), gotValue.RearRight, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesWaterTemperatureFahrenheitReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.WaterTemperature = 94.56

	// Act
	gotValue := suite.transformer.WaterTemperatureFahrenheit()

	// Assert
	suite.InEpsilon(float32(202.208), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleLengthInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleLengthInches()

	// Assert
	suite.InEpsilon(float32(177.165), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleWidthInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleWidthInches()

	// Assert
	suite.InEpsilon(float32(70.866), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleHeightInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleHeightInches()

	// Assert
	suite.InEpsilon(float32(51.181), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleWheelbaseInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleWheelbaseInches()

	// Assert
	suite.InEpsilon(float32(106.299), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleTrackFrontInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleTrackFrontInches()

	// Assert
	suite.InEpsilon(float32(61.024), gotValue, 1e-5)
}

func (suite *UnitAlternatesTestSuite) TestUnitAlternatesVehicleTrackRearInchesReturnsCorrectValue() {
	// Arrange
	suite.transformer.Vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleTrackRearInches()

	// Assert
	suite.InEpsilon(float32(62.992), gotValue, 1e-5)
}
