package telemetry

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/internal/gttelemetry"
	"github.com/zetetos/gt-telemetry/internal/vehicles"
)

type TransformerTestSuite struct {
	suite.Suite
	transformer *transformer
}

func TestTransformerTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerTestSuite))
}

func (suite *TransformerTestSuite) SetupTest() {
	transformer := NewTransformer(&vehicles.Inventory{})
	// 	db: map[uint32]vehicles.Vehicle{
	// 		1: {
	// 			Id:           1,
	// 			Model:        "Test Model",
	// 			Manufacturer: "Test Manufacturer",
	// 			Year:         2021,
	// 			Category:     "Test Category",
	// 			CarType:      "street",
	// 			Drivetrain:   "fr",
	// 			Aspiration:   "na",
	// 			OpenCockpit:  false,
	// 		},
	// 	},
	// })
	transformer.RawTelemetry = gttelemetry.GranTurismoTelemetry{}

	suite.transformer = transformer
}

func (suite *TransformerTestSuite) TestTransformerWithMissingAngularVelocityVectorReturnsZeroVector() {
	// Arrange
	suite.transformer.RawTelemetry.AngularVelocityVector = nil

	// Act
	gotValue := suite.transformer.AngularVelocityVector()

	// Assert
	suite.Equal(float32(0), gotValue.X)
	suite.Equal(float32(0), gotValue.Y)
	suite.Equal(float32(0), gotValue.Z)
}

func (suite *TransformerTestSuite) TestTransformerWithPopulatedAngularVelocityVectorHasValidVector() {
	// Arrange
	wantValue := Vector{X: 0.1, Y: 0.2, Z: 0.3}
	suite.transformer.RawTelemetry.AngularVelocityVector = &gttelemetry.GranTurismoTelemetry_Vector{
		VectorX: wantValue.X,
		VectorY: wantValue.Y,
		VectorZ: wantValue.Z,
	}

	// Act
	gotValue := suite.transformer.AngularVelocityVector()

	// Assert
	suite.Equal(wantValue.X, gotValue.X)
	suite.Equal(wantValue.Y, gotValue.Y)
	suite.Equal(wantValue.Z, gotValue.Z)
}

func (suite *TransformerTestSuite) TestTrasnformerBestLaptimeReportsCorrectDuration() {
	// Arrange
	laptime := 1234567
	wantValue := time.Duration(laptime) * time.Millisecond
	suite.transformer.RawTelemetry.BestLaptime = int32(laptime)

	// Act
	gotValue := suite.transformer.BestLaptime()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerBrakePercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(56.078434)
	suite.transformer.RawTelemetry.Brake = uint8(143)

	// Act
	gotValue := suite.transformer.BrakePercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerCalculatedVmaxReportsCorrectValue() {
	// Arrange
	wantSpeed := uint16(322)
	wantRPM := uint16(6709)
	suite.transformer.RawTelemetry.CalculatedMaxSpeed = wantSpeed
	suite.transformer.RawTelemetry.TyreRadius = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.317,
		FrontRight: 0.317,
		RearLeft:   0.317,
		RearRight:  0.317,
	}
	suite.transformer.RawTelemetry.TransmissionTopSpeedRatio = 2.49

	// Act
	gotValue := suite.transformer.CalculatedVmax()

	// Assert
	suite.Equal(wantSpeed, gotValue.Speed)
	suite.Equal(wantRPM, gotValue.RPM)
}

func (suite *TransformerTestSuite) TestTransformerClutchActuationPercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(62)
	suite.transformer.RawTelemetry.ClutchActuation = float32(0.62)

	// Act
	gotValue := suite.transformer.ClutchActuationPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerClutcEngagementPercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(87)
	suite.transformer.RawTelemetry.ClutchEngagement = float32(0.87)

	// Act
	gotValue := suite.transformer.ClutchEngagementPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerClutchOutputRPMReportsCorrectValue() {
	// Arrange
	wantValue := float32(2305)
	suite.transformer.RawTelemetry.CluchOutputRpm = wantValue

	// Act
	gotValue := suite.transformer.ClutchOutputRPM()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerNilCurrentGearReportsNeutral() {
	// Arrange
	suite.transformer.RawTelemetry.TransmissionGear = nil

	// Act
	gotValue := suite.transformer.CurrentGear()

	// Assert
	suite.Equal(15, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerCurrentGearReportsCorrectValue() {
	minGear := 0
	maxGear := 15

	for tc := minGear; tc <= maxGear; tc++ {
		suite.Run("Gear"+strconv.Itoa(tc), func() {
			// Arrange
			suite.transformer.RawTelemetry.TransmissionGear = &gttelemetry.GranTurismoTelemetry_TransmissionGear{
				Current: uint64(tc),
			}

			// Act
			gotValue := suite.transformer.CurrentGear()

			// Assert
			suite.Equal(tc, gotValue)
		})
	}
}

func (suite *TransformerTestSuite) TestTransformerCurrentGearRatioReportsCorrectValue() {
	// Arrange
	wantValues := []float32{4.32, 3.21, 2.10, 1.09, 0.87}
	suite.transformer.RawTelemetry.TransmissionGearRatio = &gttelemetry.GranTurismoTelemetry_GearRatio{
		Gear: wantValues,
	}

	for tc := 0; tc < len(wantValues); tc++ {
		suite.Run("Gear"+strconv.Itoa(tc), func() {
			// Arrange
			suite.transformer.RawTelemetry.TransmissionGear = &gttelemetry.GranTurismoTelemetry_TransmissionGear{
				Current: uint64(tc + 1),
			}

			// Act
			gotValue := suite.transformer.CurrentGearRatio()

			// Assert
			suite.Equal(wantValues[tc], gotValue)
		})
	}
}

func (suite *TransformerTestSuite) TestTransformerCurrentGearRatioOutOfBoundsReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TransmissionGearRatio = &gttelemetry.GranTurismoTelemetry_GearRatio{
		Gear: []float32{},
	}

	// Act
	gotValue := suite.transformer.CurrentGearRatio()

	// Assert
	suite.Equal(float32(-1), gotValue)
}

func (suite *TransformerTestSuite) TestTransformerCurrentLapReportsCorrectValue() {
	// Arrange
	wantValue := int16(3)
	suite.transformer.RawTelemetry.CurrentLap = uint16(wantValue)

	// Act
	gotValue := suite.transformer.CurrentLap()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerDifferentialRatioReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.DifferentialRatio()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerEngineRPMReportsCorrectValue() {
	// Arrange
	wantValue := float32(9876)
	suite.transformer.RawTelemetry.EngineRpm = wantValue

	// Act
	gotValue := suite.transformer.EngineRPM()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerEngineRPMLightInactiveBelowRPMMin() {
	// Arrange
	suite.transformer.RawTelemetry.EngineRpm = float32(1000)
	suite.transformer.RawTelemetry.RevLightRpmMin = uint16(2000)

	// Act
	gotValue := suite.transformer.EngineRPMLight().Active

	// Assert
	suite.False(gotValue)

}

func (suite *TransformerTestSuite) TestTransformerEngineRPMLightActiveAboveRPMMin() {
	// Arrange
	suite.transformer.RawTelemetry.EngineRpm = float32(2000)
	suite.transformer.RawTelemetry.RevLightRpmMin = uint16(1000)

	// Act
	gotValue := suite.transformer.EngineRPMLight().Active

	// Assert
	suite.True(gotValue)

}

func (suite *TransformerTestSuite) TestTransformerAllFlagsDisabledWhenFlagsNil() {
	// Arrange
	suite.transformer.RawTelemetry.Flags = nil

	// Act
	gotValue := suite.transformer.Flags()

	// Assert
	suite.Equal(Flags{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerFlagsReportCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.Flags = &gttelemetry.GranTurismoTelemetry_Flags{
		Live:             true,
		GamePaused:       true,
		Loading:          true,
		InGear:           true,
		HasTurbo:         true,
		RevLimiterAlert:  true,
		HandBrakeActive:  true,
		HeadlightsActive: true,
		HighBeamActive:   true,
		LowBeamActive:    true,
		AsmActive:        true,
		TcsActive:        true,
		Flag13:           true,
		Flag14:           true,
		Flag15:           true,
		Flag16:           true,
	}

	// Act
	suite.transformer.Flags()

	// Assert
	suite.Equal(true, suite.transformer.Flags().Live)
	suite.Equal(true, suite.transformer.Flags().GamePaused)
	suite.Equal(true, suite.transformer.Flags().Loading)
	suite.Equal(true, suite.transformer.Flags().InGear)
	suite.Equal(true, suite.transformer.Flags().HasTurbo)
	suite.Equal(true, suite.transformer.Flags().RevLimiterAlert)
	suite.Equal(true, suite.transformer.Flags().HandbrakeActive)
	suite.Equal(true, suite.transformer.Flags().HeadlightsActive)
	suite.Equal(true, suite.transformer.Flags().HighBeamActive)
	suite.Equal(true, suite.transformer.Flags().LowBeamActive)
	suite.Equal(true, suite.transformer.Flags().ASMActive)
	suite.Equal(true, suite.transformer.Flags().TCSActive)
	suite.Equal(true, suite.transformer.Flags().Flag13)
	suite.Equal(true, suite.transformer.Flags().Flag14)
	suite.Equal(true, suite.transformer.Flags().Flag15)
	suite.Equal(true, suite.transformer.Flags().Flag16)
}

func (suite *TransformerTestSuite) TestTransformerFuelCapacityPercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(98)
	suite.transformer.RawTelemetry.FuelCapacity = wantValue

	// Act
	gotValue := suite.transformer.FuelCapacityPercent()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerFuelLevelPercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(50)
	suite.transformer.RawTelemetry.FuelLevel = wantValue

	// Act
	gotValue := suite.transformer.FuelLevelPercent()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerTransmissionFIXME() {
	// Arrange
	// FIXME

	// Act
	suite.transformer.Transmission()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerGroundSpeedMetersPerSecondReportsCorrectValue() {
	// Arrange
	wantValue := float32(39.33952)
	suite.transformer.RawTelemetry.GroundSpeed = wantValue

	// Act
	gotValue := suite.transformer.GroundSpeedMetersPerSecond()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerHeadingReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.9477)
	suite.transformer.RawTelemetry.Heading = wantValue

	// Act
	gotValue := suite.transformer.Heading()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerLastLaptimeReportsCorrectValue() {
	// Arrange
	laptime := 123456
	wantValue := time.Duration(laptime) * time.Millisecond
	suite.transformer.RawTelemetry.LastLaptime = int32(laptime)

	// Act
	gotValue := suite.transformer.LastLaptime()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerOilPressureKPAReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.12345)
	suite.transformer.RawTelemetry.OilPressure = wantValue

	// Act
	gotValue := suite.transformer.OilPressureKPA()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerOilTemperatureCelsiusReportsCorrectValue() {
	// Arrange
	wantValue := float32(104.2945)
	suite.transformer.RawTelemetry.OilTemperature = wantValue

	// Act
	gotValue := suite.transformer.OilTemperatureCelsius()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerNilPositionalMapCoordinatesReportsEmptyVector() {
	suite.transformer.RawTelemetry.RotationAxes = nil

	// Act
	gotValue := suite.transformer.PositionalMapCoordinates()

	// Assert
	suite.Equal(Vector{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerPositionalMapCoordinatesReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.MapPositionCoordinates = &gttelemetry.GranTurismoTelemetry_Coordinate{
		CoordinateX: 10.1,
		CoordinateY: 20.2,
		CoordinateZ: 30.3,
	}

	// Act
	suite.transformer.PositionalMapCoordinates()

	// Assert
	suite.Equal(float32(10.1), suite.transformer.PositionalMapCoordinates().X)
	suite.Equal(float32(20.2), suite.transformer.PositionalMapCoordinates().Y)
	suite.Equal(float32(30.3), suite.transformer.PositionalMapCoordinates().Z)
}

func (suite *TransformerTestSuite) TestTransformerRaceEntrantsReportsCorrectValue() {
	// Arrange
	wantValue := int16(16)
	suite.transformer.RawTelemetry.RaceEntrants = wantValue

	// Act
	gotValue := suite.transformer.RaceEntrants()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerRaceLapsReportsCorrectValue() {
	// Arrange
	wantValue := uint16(30)
	suite.transformer.RawTelemetry.RaceLaps = wantValue

	// Act
	gotValue := suite.transformer.RaceLaps()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerRideHeightMetersReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.12345)
	suite.transformer.RawTelemetry.RideHeight = wantValue

	// Act
	gotValue := suite.transformer.RideHeightMeters()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerNilRotationVectorReportsEmptyVector() {
	// Arrange
	suite.transformer.RawTelemetry.RotationAxes = nil

	// Act
	gotValue := suite.transformer.RotationVector()

	// Assert
	suite.Equal(SymmetryAxes{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerRotationVectorReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.RotationAxes = &gttelemetry.GranTurismoTelemetry_SymmetryAxes{
		Yaw:   0.1,
		Pitch: 0.2,
		Roll:  0.3,
	}

	// Act
	gotValue := suite.transformer.RotationVector()

	// Assert
	suite.Equal(float32(0.1), gotValue.Yaw)
	suite.Equal(float32(0.2), gotValue.Pitch)
	suite.Equal(float32(0.3), gotValue.Roll)
}

func (suite *TransformerTestSuite) TestTransformerSequenceIDReportsCorrectValue() {
	// Arrange
	wantValue := uint32(123456789)
	suite.transformer.RawTelemetry.SequenceId = wantValue

	// Act
	gotValue := suite.transformer.SequenceID()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerStartingPositionReportsCorrectValue() {
	// Arrange
	wantValue := int16(5)
	suite.transformer.RawTelemetry.StartingPosition = wantValue

	// Act
	gotValue := suite.transformer.StartingPosition()

	// Assert
	suite.Equal(wantValue, gotValue)

}

func (suite *TransformerTestSuite) TestTransformerNilSuggestedGearReportsNeutral() {
	// Arrange
	suite.transformer.RawTelemetry.TransmissionGear = nil

	// Act
	gotValue := suite.transformer.SuggestedGear()

	// Assert
	suite.Equal(uint64(15), gotValue)
}

func (suite *TransformerTestSuite) TestTransformerSuggestedGearReportsCorrectValue() {
	minGear := 0
	maxGear := 15

	for tc := minGear; tc <= maxGear; tc++ {
		suite.Run("Gear"+strconv.Itoa(tc), func() {
			// Arrange
			suite.transformer.RawTelemetry.TransmissionGear = &gttelemetry.GranTurismoTelemetry_TransmissionGear{
				Suggested: uint64(tc),
			}

			// Act
			gotValue := suite.transformer.SuggestedGear()

			// Assert
			suite.Equal(uint64(tc), gotValue)
		})
	}
}

func (suite *TransformerTestSuite) TestTransformerNilSuspensionHeightMetersReportsEmptyCornerSet() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = nil

	// Act
	gotValue := suite.transformer.SuspensionHeightMeters()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerSuspensionHeightMetersReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.101,
		FrontRight: 0.102,
		RearLeft:   0.103,
		RearRight:  0.104,
	}

	// Act
	gotValue := suite.transformer.SuspensionHeightMeters()

	// Assert
	suite.Equal(float32(0.101), gotValue.FrontLeft)
	suite.Equal(float32(0.102), gotValue.FrontRight)
	suite.Equal(float32(0.103), gotValue.RearLeft)
	suite.Equal(float32(0.104), gotValue.RearRight)
}

func (suite *TransformerTestSuite) TestTransformerThrottlePercentReportsCorrectValue() {
	// Arrange
	wantValue := float32(79.60784)
	suite.transformer.RawTelemetry.Throttle = 203

	// Act
	gotValue := suite.transformer.ThrottlePercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTimeOfDayReportsCorrectValue() {
	// Arrange
	timeMS := 34567890
	wantValue := time.Duration(timeMS) * time.Millisecond
	suite.transformer.RawTelemetry.TimeOfDay = uint32(timeMS)

	// Act
	gotValue := suite.transformer.TimeOfDay()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTransmissionTopSpeedRatioReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.7890)
	suite.transformer.RawTelemetry.TransmissionTopSpeedRatio = wantValue

	// Act
	gotValue := suite.transformer.TransmissionTopSpeedRatio()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTurboBoostBarReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.821)
	suite.transformer.RawTelemetry.ManifoldPressure = float32(1.821)

	// Act
	gotValue := suite.transformer.TurboBoostBar()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerNilTyreDiameterMetersReportsEmptyCornerSet() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = nil

	// Act
	gotValue := suite.transformer.TyreDiameterMeters()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTyreDiameterMetersReportsCorrectValue() {
	// Arrange
	// FIXME

	// Act
	suite.transformer.TyreDiameterMeters()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerNilTyreRadiusMetersReportsEmptyCornerSet() {
	// Arrange
	suite.transformer.RawTelemetry.TyreRadius = nil

	// Act
	gotValue := suite.transformer.TyreRadiusMeters()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTyreRadiusMetersReportsCorrectValue() {
	// Arrange
	wantValue := float32(0.317)
	suite.transformer.RawTelemetry.TyreRadius = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  wantValue,
		FrontRight: wantValue,
		RearLeft:   wantValue,
		RearRight:  wantValue,
	}

	// Act
	gotValue := suite.transformer.TyreRadiusMeters()

	// Assert
	suite.Equal(wantValue, gotValue.FrontLeft)
	suite.Equal(wantValue, gotValue.FrontRight)
	suite.Equal(wantValue, gotValue.RearLeft)
	suite.Equal(wantValue, gotValue.RearRight)
}

func (suite *TransformerTestSuite) TestTransformerTyreSlipRatioIsOneWhenGroundSpeedZero() {
	// Arrange
	suite.transformer.RawTelemetry.GroundSpeed = 0

	// Act
	gotValue := suite.transformer.TyreSlipRatio()

	// Assert
	suite.Equal(CornerSet{FrontLeft: 1, FrontRight: 1, RearLeft: 1, RearRight: 1}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTyreSlipRatioReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.GroundSpeed = 42
	suite.transformer.RawTelemetry.TyreRadius = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  0.317,
		FrontRight: 0.317,
		RearLeft:   0.317,
		RearRight:  0.317,
	}
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  132.50,
		FrontRight: 132.51,
		RearLeft:   132.45,
		RearRight:  132.40,
	}

	// Act
	gotValue := suite.transformer.TyreSlipRatio()

	// Assert
	suite.Equal(float32(1.0000595), gotValue.FrontLeft)
	suite.Equal(float32(1.000135), gotValue.FrontRight)
	suite.Equal(float32(0.9996821), gotValue.RearLeft)
	suite.Equal(float32(0.99930465), gotValue.RearRight)
}

func (suite *TransformerTestSuite) TestTransformerNilTyreTemperatureCelsiusReportsEmptyCornerSet() {
	// Arrange
	suite.transformer.RawTelemetry.TyreTemperature = nil

	// Act
	gotValue := suite.transformer.TyreTemperatureCelsius()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerTyreTemperatureCelsiusReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TyreTemperature = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  64.3,
		FrontRight: 64.1,
		RearLeft:   68.2,
		RearRight:  67.8,
	}

	// Act
	gotValue := suite.transformer.TyreTemperatureCelsius()

	// Assert
	suite.Equal(float32(64.3), gotValue.FrontLeft)
	suite.Equal(float32(64.1), gotValue.FrontRight)
	suite.Equal(float32(68.2), gotValue.RearLeft)
	suite.Equal(float32(67.8), gotValue.RearRight)
}

func (suite *TransformerTestSuite) TestTransformerVehicleIDReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleID()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleCategoryReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleCategory()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleDrivetrainReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleDrivetrain()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleManufacturerReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleManufacturer()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleModelReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleModel()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleHasOpenCockpitReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleHasOpenCockpit()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerVehicleYearReportsCorrectValue() {
	// Arrange
	// FIXME needs vehicle inventory to be mod-able

	// Act
	suite.transformer.VehicleYear()

	// Assert
}

func (suite *TransformerTestSuite) TestTransformerNilWheelSpeedRadiansPerSecondReportsEmptyCornerSet() {
	// Arrange
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = nil

	// Act
	gotValue := suite.transformer.WheelSpeedRadiansPerSecond()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerWheelSpeedRadiansPerSecondReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  132.50,
		FrontRight: 132.51,
		RearLeft:   132.45,
		RearRight:  132.40,
	}

	// Act
	gotValue := suite.transformer.WheelSpeedRadiansPerSecond()

	// Assert
	suite.Equal(float32(132.50), gotValue.FrontLeft)
	suite.Equal(float32(132.51), gotValue.FrontRight)
	suite.Equal(float32(132.45), gotValue.RearLeft)
	suite.Equal(float32(132.40), gotValue.RearRight)
}

func (suite *TransformerTestSuite) TestTransformerNilVelocityVectorReportsEmptyVector() {
	// Arrange
	suite.transformer.RawTelemetry.VelocityVector = nil

	// Act
	gotValue := suite.transformer.VelocityVector()

	// Assert
	suite.Equal(Vector{}, gotValue)
}

func (suite *TransformerTestSuite) TestTransformerVelocityVectorReportsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.VelocityVector = &gttelemetry.GranTurismoTelemetry_Vector{
		VectorX: 42.1,
		VectorY: 1.2,
		VectorZ: 0.3,
	}

	// Act
	gotValue := suite.transformer.VelocityVector()

	// Assert
	suite.Equal(float32(42.1), gotValue.X)
	suite.Equal(float32(1.2), gotValue.Y)
	suite.Equal(float32(0.3), gotValue.Z)
}

func (suite *TransformerTestSuite) TestTransformerWaterTemperatureCelsiusReportsCorrectValue() {
	// Arrange
	wantValue := float32(94.56)
	suite.transformer.RawTelemetry.WaterTemperature = wantValue

	// Act
	gotValue := suite.transformer.WaterTemperatureCelsius()

	// Assert
	suite.Equal(wantValue, gotValue)
}
