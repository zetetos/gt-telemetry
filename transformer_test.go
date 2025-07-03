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
	inventoryJSON := []byte(`{
		"1234": {
			"Model": "Dummy Model",
			"Manufacturer": "Dummy Manufacturer",
			"Category": "Gr.1",
			"Drivetrain": "FR",
			"Aspiration": "NA",
			"Year": 2025,
			"CarID": 1234,
			"OpenCockpit": false,
			"CarType": "race"
		}
	}`)
	inventory, _ := vehicles.NewInventory(inventoryJSON)
	transformer := NewTransformer(inventory)
	transformer.RawTelemetry = gttelemetry.GranTurismoTelemetry{}

	suite.transformer = transformer
}

func (suite *TransformerTestSuite) TestAngularVelocityVectorReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := Vector{}
	suite.transformer.RawTelemetry.AngularVelocityVector = nil

	// Act
	gotValue := suite.transformer.AngularVelocityVector()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestAngularVelocityVectorReturnsCorrectVectorWhenTelemetryPopulated() {
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

func (suite *TransformerTestSuite) TestBestLaptimeReturnsCorrectDuration() {
	// Arrange
	laptime := 1234567
	wantValue := time.Duration(laptime) * time.Millisecond
	suite.transformer.RawTelemetry.BestLaptime = int32(laptime)

	// Act
	gotValue := suite.transformer.BestLaptime()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestBrakePedalPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(84.31373)
	suite.transformer.RawTelemetry.BrakeRaw = uint8(215)

	// Act
	gotValue := suite.transformer.BrakePedalPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestBrakePercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(56.078434)
	suite.transformer.RawTelemetry.Brake = uint8(143)

	// Act
	gotValue := suite.transformer.BrakePercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestCalculatedVmaxReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestClutchActuationPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(62)
	suite.transformer.RawTelemetry.ClutchActuation = float32(0.62)

	// Act
	gotValue := suite.transformer.ClutchActuationPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestClutcEngagementPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(87)
	suite.transformer.RawTelemetry.ClutchEngagement = float32(0.87)

	// Act
	gotValue := suite.transformer.ClutchEngagementPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestClutchOutputRPMReturnsCorrectValue() {
	// Arrange
	wantValue := float32(2305)
	suite.transformer.RawTelemetry.CluchOutputRpm = wantValue

	// Act
	gotValue := suite.transformer.ClutchOutputRPM()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestCurrentGearReturnssNeutralWhenTelemetryIsNil() {
	// Arrange
	wantValue := 15
	suite.transformer.RawTelemetry.TransmissionGear = nil

	// Act
	gotValue := suite.transformer.CurrentGear()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestCurrentGearReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestCurrentGearRatioReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestCurrentGearRatioReturnsDefaultValueWhenTelemetryIsNil() {
	// Arrange
	suite.transformer.RawTelemetry.TransmissionGearRatio = &gttelemetry.GranTurismoTelemetry_GearRatio{
		Gear: []float32{},
	}

	// Act
	gotValue := suite.transformer.CurrentGearRatio()

	// Assert
	suite.Equal(float32(-1), gotValue)
}

func (suite *TransformerTestSuite) TestCurrentLapReturnsCorrectValue() {
	// Arrange
	wantValue := int16(3)
	suite.transformer.RawTelemetry.CurrentLap = uint16(wantValue)

	// Act
	gotValue := suite.transformer.CurrentLap()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestDifferentialRatioReturnsCorrectValueForDrivetrain() {
	tests := []struct {
		drivetrain string
		gears      []float32
		result     float32
	}{
		{
			drivetrain: "FR",
			gears:      []float32{2.99, 1.31, 0.56},
			result:     4.8210554,
		},
		{
			drivetrain: "FF",
			gears:      []float32{1.8, 1.1, 0.7},
			result:     3.8568444,
		},
		{
			drivetrain: "INVALID",
			gears:      []float32{},
			result:     -1,
		},
	}

	for _, test := range tests {
		suite.Run(test.drivetrain, func() {
			// Arrange
			wantValue := test.result
			suite.transformer.vehicle.Drivetrain = test.drivetrain
			suite.transformer.RawTelemetry.TransmissionTopSpeedRatio = 2.7
			suite.transformer.RawTelemetry.TransmissionGearRatio = &gttelemetry.GranTurismoTelemetry_GearRatio{
				Gear: test.gears,
			}
			suite.transformer.RawTelemetry.CalculatedMaxSpeed = 330
			suite.transformer.RawTelemetry.TyreRadius = &gttelemetry.GranTurismoTelemetry_CornerSet{
				FrontLeft:  0.348,
				FrontRight: 0.348,
				RearLeft:   0.348,
				RearRight:  0.348,
			}

			// Act
			gotValue := suite.transformer.DifferentialRatio()

			// Assert
			suite.Equal(wantValue, gotValue)
		})
	}
}

func (suite *TransformerTestSuite) TestEnergyRecoveryReturnsCorrectValue() {
	// Arrange
	wantValue := float32(-0.48776)
	suite.transformer.RawTelemetry.EnergyRecovery = wantValue

	// Act
	gotValue := suite.transformer.EnergyRecovery()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestEngineRPMReturnsCorrectValue() {
	// Arrange
	wantValue := float32(9876)
	suite.transformer.RawTelemetry.EngineRpm = wantValue

	// Act
	gotValue := suite.transformer.EngineRPM()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestEngineRPMLightReturnsInactiveWhenBelowMinimumRPM() {
	// Arrange
	suite.transformer.RawTelemetry.EngineRpm = float32(1000)
	suite.transformer.RawTelemetry.RevLightRpmMin = uint16(2000)

	// Act
	gotValue := suite.transformer.EngineRPMLight().Active

	// Assert
	suite.False(gotValue)
}

func (suite *TransformerTestSuite) TestEngineRPMLightReturnsActiveWhenAboveMinimumRPM() {
	// Arrange
	suite.transformer.RawTelemetry.EngineRpm = float32(2000)
	suite.transformer.RawTelemetry.RevLightRpmMin = uint16(1000)

	// Act
	gotValue := suite.transformer.EngineRPMLight().Active

	// Assert
	suite.True(gotValue)
}

func (suite *TransformerTestSuite) TestFlagsAreDisabledWhenTelemetryIsNil() {
	// Arrange
	suite.transformer.RawTelemetry.Flags = nil

	// Act
	gotValue := suite.transformer.Flags()

	// Assert
	suite.Equal(Flags{}, gotValue)
}

func (suite *TransformerTestSuite) TestFlagsReturnCorrectValues() {
	states := []bool{true, false}

	for _, state := range states {
		suite.Run(strconv.FormatBool(state), func() {
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
		})
	}
}

func (suite *TransformerTestSuite) TestFuelCapacityPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(98)
	suite.transformer.RawTelemetry.FuelCapacity = wantValue

	// Act
	gotValue := suite.transformer.FuelCapacityPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestFuelLevelPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(50)
	suite.transformer.RawTelemetry.FuelLevel = wantValue

	// Act
	gotValue := suite.transformer.FuelLevelPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGameVersionFIXME() {
	// Arrange
	// wantValue := "gt6"

	// Act
	//suite.transformer.GameVersion()

	// Assert
	// suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTransmissionReturnsCorrectValue() {
	// Arrange
	ratios := []float32{2.0, 1.0, 0.5}
	wantValue := Transmission{
		Gears:      3,
		GearRatios: ratios,
	}
	suite.transformer.RawTelemetry.TransmissionGearRatio = &gttelemetry.GranTurismoTelemetry_GearRatio{
		Gear: ratios,
	}

	// Act
	gotValue := suite.transformer.Transmission()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGroundSpeedMetersPerSecondReturnsCorrectValue() {
	// Arrange
	wantValue := float32(39.33952)
	suite.transformer.RawTelemetry.GroundSpeed = wantValue

	// Act
	gotValue := suite.transformer.GroundSpeedMetersPerSecond()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestHeadingReturnsCorrectValue() {
	// Arrange
	wantValue := float32(0.9477)
	suite.transformer.RawTelemetry.Heading = wantValue

	// Act
	gotValue := suite.transformer.Heading()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestLastLaptimeReturnsCorrectValue() {
	// Arrange
	laptime := 123456
	wantValue := time.Duration(laptime) * time.Millisecond
	suite.transformer.RawTelemetry.LastLaptime = int32(laptime)

	// Act
	gotValue := suite.transformer.LastLaptime()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestOilPressureKPAReturnsCorrectValue() {
	// Arrange
	wantValue := float32(0.12345)
	suite.transformer.RawTelemetry.OilPressure = wantValue

	// Act
	gotValue := suite.transformer.OilPressureKPA()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestOilTemperatureCelsiusReturnsCorrectValue() {
	// Arrange
	wantValue := float32(104.2945)
	suite.transformer.RawTelemetry.OilTemperature = wantValue

	// Act
	gotValue := suite.transformer.OilTemperatureCelsius()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestPositionalMapCoordinatesReturnsEmptyObjectWhenTelemetryIsNil() {
	// Assert
	wantValue := Vector{}
	suite.transformer.RawTelemetry.MapPositionCoordinates = nil

	// Act
	gotValue := suite.transformer.PositionalMapCoordinates()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestPositionalMapCoordinatesReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestRaceEntrantsReturnsCorrectValue() {
	// Arrange
	wantValue := int16(16)
	suite.transformer.RawTelemetry.RaceEntrants = wantValue

	// Act
	gotValue := suite.transformer.RaceEntrants()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestRaceLapsReturnsCorrectValue() {
	// Arrange
	wantValue := uint16(30)
	suite.transformer.RawTelemetry.RaceLaps = wantValue

	// Act
	gotValue := suite.transformer.RaceLaps()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestRideHeightMetersReturnsCorrectValue() {
	// Arrange
	wantValue := float32(0.12345)
	suite.transformer.RawTelemetry.RideHeight = wantValue

	// Act
	gotValue := suite.transformer.RideHeightMeters()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestRotationEnvelopeReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := RotationalEnvelope{}
	suite.transformer.RawTelemetry.RotationalEnvelope = nil

	// Act
	gotValue := suite.transformer.RotationEnvelope()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestRotationEnvelopeReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.RotationalEnvelope = &gttelemetry.GranTurismoTelemetry_RotationalEnvelope{
		Yaw:   0.1,
		Pitch: 0.2,
		Roll:  0.3,
	}

	// Act
	gotValue := suite.transformer.RotationEnvelope()

	// Assert
	suite.Equal(float32(0.1), gotValue.Yaw)
	suite.Equal(float32(0.2), gotValue.Pitch)
	suite.Equal(float32(0.3), gotValue.Roll)
}

func (suite *TransformerTestSuite) TestRotationVectorReturnssEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := SymmetryAxes{}
	suite.transformer.RawTelemetry.RotationalEnvelope = nil

	// Act
	gotValue := suite.transformer.RotationVector()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestRotationVectorReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.RotationalEnvelope = &gttelemetry.GranTurismoTelemetry_RotationalEnvelope{
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

func (suite *TransformerTestSuite) TestSequenceIDReturnsCorrectValue() {
	// Arrange
	wantValue := uint32(123456789)
	suite.transformer.RawTelemetry.SequenceId = wantValue

	// Act
	gotValue := suite.transformer.SequenceID()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestSteeringWheelAngleRadiansReturnsCorrectValue() {
	// Arrange
	wantValue := float32(2.18334)
	suite.transformer.RawTelemetry.SteeringWheelAngleRadians = wantValue

	// Act
	gotValue := suite.transformer.SteeringWheelAngleRadians()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestSteeringWheelAngleDegreesReturnsCorrectValue() {
	// Arrange
	wantValue := float32(125.096176)
	suite.transformer.RawTelemetry.SteeringWheelAngleRadians = float32(2.18334)

	// Act
	gotValue := suite.transformer.SteeringWheelAngleDegrees()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestStartingPositionReturnsCorrectValue() {
	// Arrange
	wantValue := int16(5)
	suite.transformer.RawTelemetry.StartingPosition = wantValue

	// Act
	gotValue := suite.transformer.StartingPosition()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestSuggestedGearReturnsNeutralWhenTelemetryIsNil() {
	// Arrange
	wantValue := uint64(15)
	suite.transformer.RawTelemetry.TransmissionGear = nil

	// Act
	gotValue := suite.transformer.SuggestedGear()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestSuggestedGearReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestSuspensionHeightMetersReturnsEmptyCornerSetWhenTelemetryNil() {
	// Arrange
	suite.transformer.RawTelemetry.SuspensionHeight = nil

	// Act
	gotValue := suite.transformer.SuspensionHeightMeters()

	// Assert
	suite.Equal(CornerSet{}, gotValue)
}

func (suite *TransformerTestSuite) TestSuspensionHeightMetersRetursCorrectValue() {
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

func (suite *TransformerTestSuite) TestTelemetryFormatIndicatorsFIXME() {
	tests := []struct {
		name    string
		format  string
		results []bool
	}{
		{
			name:    "format_a",
			format:  "A",
			results: []bool{true, false, false},
		},
		{
			name:    "format_b",
			format:  "B",
			results: []bool{true, true, false},
		},
		{
			name:    "format_tilde",
			format:  "~",
			results: []bool{true, true, true},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// Arrange
			// TODO: setup telemetry with format `test.format`

			// Act
			// gotValueA, err := suite.transformer.RawTelemetry.HasSectionA()
			// suite.Require().NoError(err)

			// gotValueB, err := suite.transformer.RawTelemetry.HasSectionA()
			// suite.Require().NoError(err)

			// gotValueTilde, err := suite.transformer.RawTelemetry.HasSectionA()
			// suite.Require().NoError(err)

			// Assert
			// suite.Equal(test.results[0], gotValueA)
			// suite.Equal(test.results[1], gotValueB)
			// suite.Equal(test.results[2], gotValueTilde)
			suite.Equal(true, test.results[0])
		})
	}
}

func (suite *TransformerTestSuite) TestThrottlePercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(79.60784)
	suite.transformer.RawTelemetry.Throttle = 203

	// Act
	gotValue := suite.transformer.ThrottlePercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestThrottlePedalPercentReturnsCorrectValue() {
	// Arrange
	wantValue := float32(37.64706)
	suite.transformer.RawTelemetry.ThrottleRaw = uint8(96)

	// Act
	gotValue := suite.transformer.ThrottlePedalPercent()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTimeOfDayReturnsCorrectDuration() {
	// Arrange
	timeMS := 34567890
	wantValue := time.Duration(timeMS) * time.Millisecond
	suite.transformer.RawTelemetry.TimeOfDay = uint32(timeMS)

	// Act
	gotValue := suite.transformer.TimeOfDay()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTranslationEnvelopeReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := TranslationalEnvelope{}
	suite.transformer.RawTelemetry.TranslationalEnvelope = nil

	// Act
	gotValue := suite.transformer.TranslationEnvelope()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTranslationEnvelopeReturnsCorrectValue() {
	// Arrange
	suite.transformer.RawTelemetry.TranslationalEnvelope = &gttelemetry.GranTurismoTelemetry_TranslationalEnvelope{
		Sway:  0.1,
		Heave: 0.2,
		Surge: 0.3,
	}

	// Act
	gotValue := suite.transformer.TranslationEnvelope()

	// Assert
	suite.Equal(float32(0.1), gotValue.Sway)
	suite.Equal(float32(0.2), gotValue.Heave)
	suite.Equal(float32(0.3), gotValue.Surge)
}

func (suite *TransformerTestSuite) TestTransmissionTopSpeedRatioReturnsCorrectValue() {
	// Arrange
	wantValue := float32(0.7890)
	suite.transformer.RawTelemetry.TransmissionTopSpeedRatio = wantValue

	// Act
	gotValue := suite.transformer.TransmissionTopSpeedRatio()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTurboBoostBarReturnsCorrectValue() {
	// Arrange
	wantValue := float32(0.821)
	suite.transformer.RawTelemetry.ManifoldPressure = float32(1.821)

	// Act
	gotValue := suite.transformer.TurboBoostBar()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTyreDiameterMetersReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := CornerSet{}
	suite.transformer.RawTelemetry.TyreRadius = nil

	// Act
	gotValue := suite.transformer.TyreDiameterMeters()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTyreDiameterMetersReturnsCorrectValue() {
	// Arrange
	tyreRadius := float32(0.323)
	wantValue := CornerSet{
		FrontLeft:  tyreRadius * 2,
		FrontRight: tyreRadius * 2,
		RearLeft:   tyreRadius * 2,
		RearRight:  tyreRadius * 2,
	}
	suite.transformer.RawTelemetry.TyreRadius = &gttelemetry.GranTurismoTelemetry_CornerSet{
		FrontLeft:  tyreRadius,
		FrontRight: tyreRadius,
		RearLeft:   tyreRadius,
		RearRight:  tyreRadius,
	}

	// Act
	gotValue := suite.transformer.TyreDiameterMeters()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTyreRadiusMetersReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := CornerSet{}
	suite.transformer.RawTelemetry.TyreRadius = nil

	// Act
	gotValue := suite.transformer.TyreRadiusMeters()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTyreRadiusMetersReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestTyreSlipRatioReturnsCorrectValueWhenGroundSpeedIsZero() {
	// Arrange
	suite.transformer.RawTelemetry.GroundSpeed = 0

	// Act
	gotValue := suite.transformer.TyreSlipRatio()

	// Assert
	suite.Equal(CornerSet{FrontLeft: 1, FrontRight: 1, RearLeft: 1, RearRight: 1}, gotValue)
}

func (suite *TransformerTestSuite) TestTyreSlipRatioReturnsCorrectValueWhenGroundSpeedIsNotZero() {
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

func (suite *TransformerTestSuite) TestTyreTemperatureCelsiusReturnsEmptyOjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := CornerSet{}
	suite.transformer.RawTelemetry.TyreTemperature = nil

	// Act
	gotValue := suite.transformer.TyreTemperatureCelsius()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestTyreTemperatureCelsiusReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestGetVehicleAspirationReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "NA"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleAspiration()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleAspirationExpandedReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "Naturally Aspirated"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleAspirationExpanded()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleIDReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := uint32(1234)
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = wantValue

	// Act
	gotValue := suite.transformer.VehicleID()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleCategoryReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "Gr.1"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleCategory()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleDrivetrainReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "FR"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleDrivetrain()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleManufacturerReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "Dummy Manufacturer"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleManufacturer()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleModelReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "Dummy Model"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleModel()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleHasOpenCockpitReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := false
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleHasOpenCockpit()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleTypeReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := "race"
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleType()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestGetVehicleYearReturnsCorrectValueWhenTelemetryHasKnownID() {
	// Arrange
	wantValue := 2025
	suite.transformer.vehicle = vehicles.Vehicle{}
	suite.transformer.RawTelemetry.VehicleId = 1234

	// Act
	gotValue := suite.transformer.VehicleYear()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestWheelSpeedRadiansPerSecondReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := CornerSet{}
	suite.transformer.RawTelemetry.WheelRadiansPerSecond = nil

	// Act
	gotValue := suite.transformer.WheelSpeedRadiansPerSecond()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestWheelSpeedRadiansPerSecondReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestVehicleObjectIsEmptyWhenTelemetryHasUnknownID() {
	// Arrange
	wantValue := vehicles.Vehicle{}

	initialVehicleID := 1234
	suite.transformer.vehicle = vehicles.Vehicle{
		CarID: initialVehicleID,
	}

	invalidVehicleID := uint32(0)
	suite.transformer.RawTelemetry.VehicleId = invalidVehicleID

	// Act
	suite.transformer.VehicleID() // trigger vehicle object update
	gotValue := suite.transformer.vehicle

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestVelocityVectorReturnsEmptyObjectWhenTelemetryIsNil() {
	// Arrange
	wantValue := Vector{}
	suite.transformer.RawTelemetry.VelocityVector = nil

	// Act
	gotValue := suite.transformer.VelocityVector()

	// Assert
	suite.Equal(wantValue, gotValue)
}

func (suite *TransformerTestSuite) TestVelocityVectorReturnsCorrectValue() {
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

func (suite *TransformerTestSuite) TestWaterTemperatureCelsiusReturnsCorrectValue() {
	// Arrange
	wantValue := float32(94.56)
	suite.transformer.RawTelemetry.WaterTemperature = wantValue

	// Act
	gotValue := suite.transformer.WaterTemperatureCelsius()

	// Assert
	suite.Equal(wantValue, gotValue)
}
