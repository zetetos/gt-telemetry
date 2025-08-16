package telemetry

import (
	"math"
	"time"

	"github.com/zetetos/gt-telemetry/internal/gttelemetry"
	"github.com/zetetos/gt-telemetry/internal/telemetrysrc"
	"github.com/zetetos/gt-telemetry/internal/utils"
	"github.com/zetetos/gt-telemetry/internal/vehicles"
)

type CornerSet struct {
	FrontLeft  float32
	FrontRight float32
	RearLeft   float32
	RearRight  float32
}

type Flags struct {
	ASMActive        bool
	GamePaused       bool
	HandbrakeActive  bool
	HasTurbo         bool
	HeadlightsActive bool
	HighBeamActive   bool
	InGear           bool
	Live             bool
	Loading          bool
	LowBeamActive    bool
	RevLimiterAlert  bool
	TCSActive        bool
	Flag13           bool
	Flag14           bool
	Flag15           bool
	Flag16           bool
}

type Transmission struct {
	Gears      int
	GearRatios []float32
}

type RevLight struct {
	Min    uint16
	Max    uint16
	Active bool
}

type RotationalEnvelope struct {
	Pitch float32
	Yaw   float32
	Roll  float32
}

type SymmetryAxes struct {
	Pitch float32
	Yaw   float32
	Roll  float32
}

type TranslationalEnvelope struct {
	Sway  float32
	Heave float32
	Surge float32
}

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Vmax struct {
	Speed uint16
	RPM   uint16
}

type transformer struct {
	RawTelemetry gttelemetry.GranTurismoTelemetry
	inventory    *vehicles.Inventory
	vehicle      vehicles.Vehicle
}

func NewTransformer(inventory *vehicles.Inventory) *transformer {
	return &transformer{
		RawTelemetry: gttelemetry.GranTurismoTelemetry{},
		inventory:    inventory,
		vehicle:      vehicles.Vehicle{},
	}
}

func (t *transformer) AngularVelocityVector() Vector {
	velocity := t.RawTelemetry.AngularVelocityVector
	if velocity == nil {
		return Vector{}
	}

	return Vector{
		X: velocity.VectorX,
		Y: velocity.VectorY,
		Z: velocity.VectorZ,
	}
}

func (t *transformer) BestLaptime() time.Duration {
	return time.Duration(t.RawTelemetry.BestLaptime) * time.Millisecond
}

func (t *transformer) BrakeInputPercent() float32 {
	return float32(t.RawTelemetry.BrakeInput) / 2.55
}

func (t *transformer) BrakeOutputPercent() float32 {
	return float32(t.RawTelemetry.BrakeOutput) / 2.55
}

// To be deprecated in 2.0 release
func (t *transformer) BrakePercent() float32 {
	return float32(t.RawTelemetry.BrakeInput) / 2.55
}

func (t *transformer) CalculatedVmax() Vmax {
	vMaxSpeed := t.RawTelemetry.CalculatedMaxSpeed
	vMaxMetersPerMinute := float32(vMaxSpeed) * 1000 / 60
	tyreCircumference := t.TyreDiameterMeters().RearLeft * math.Pi

	return Vmax{
		Speed: vMaxSpeed,
		RPM:   uint16((vMaxMetersPerMinute / tyreCircumference) * t.TransmissionTopSpeedRatio()),
	}
}

func (t *transformer) ClutchActuationPercent() float32 {
	return t.RawTelemetry.ClutchActuation * 100
}

func (t *transformer) ClutchEngagementPercent() float32 {
	return t.RawTelemetry.ClutchEngagement * 100
}

func (t *transformer) ClutchOutputRPM() float32 {
	return t.RawTelemetry.CluchOutputRpm
}

// Currently selected transmission gear, 15 is neutral
func (t *transformer) CurrentGear() int {
	gear := t.RawTelemetry.TransmissionGear
	if gear == nil {
		return 15
	}

	return int(gear.Current)
}

func (t *transformer) CurrentGearRatio() float32 {
	gear := t.CurrentGear()
	if gear > len(t.Transmission().GearRatios) {
		return -1
	}

	return t.Transmission().GearRatios[gear-1]
}

func (t *transformer) CurrentLap() int16 {
	return int16(t.RawTelemetry.CurrentLap)
}

func (t *transformer) DifferentialRatio() float32 {
	t.updateVehicle()

	transmission := t.Transmission()
	if transmission.Gears == 0 {
		return -1
	}
	highestRatio := transmission.GearRatios[transmission.Gears-1]
	vMax := t.CalculatedVmax()

	rollingDiameter := float32(0)
	switch t.vehicle.Drivetrain {
	case "FF":
		rollingDiameter = t.TyreDiameterMeters().FrontLeft
	default:
		rollingDiameter = t.TyreDiameterMeters().RearLeft
	}

	vMaxMetersPerMinute := float32(vMax.Speed) * 1000 / 60
	wheelRpm := vMaxMetersPerMinute / (rollingDiameter * math.Pi)
	diffRatio := (float32(vMax.RPM) / highestRatio) / wheelRpm

	return diffRatio
}

func (t *transformer) EnergyRecovery() float32 {
	return t.RawTelemetry.EnergyRecovery
}

func (t *transformer) EngineRPM() float32 {
	val := t.RawTelemetry.EngineRpm

	return val
}

func (t *transformer) EngineRPMLight() RevLight {
	rpm := uint16(t.EngineRPM())
	lightMin := t.RawTelemetry.RevLightRpmMin
	lightMax := t.RawTelemetry.RevLightRpmMax

	active := false
	if rpm > lightMin {
		active = true
	}

	return RevLight{
		Min:    lightMin,
		Max:    lightMax,
		Active: active,
	}
}

func (t *transformer) Flags() Flags {
	flags := t.RawTelemetry.Flags
	if flags == nil {
		return Flags{}
	}

	return Flags{
		ASMActive:        flags.AsmActive,
		GamePaused:       flags.GamePaused,
		HandbrakeActive:  flags.HandBrakeActive,
		HasTurbo:         flags.HasTurbo,
		HeadlightsActive: flags.HeadlightsActive,
		HighBeamActive:   flags.HighBeamActive,
		InGear:           flags.InGear,
		Live:             flags.Live,
		Loading:          flags.Loading,
		LowBeamActive:    flags.LowBeamActive,
		RevLimiterAlert:  flags.RevLimiterAlert,
		TCSActive:        flags.TcsActive,
		Flag13:           flags.Flag13,
		Flag14:           flags.Flag14,
		Flag15:           flags.Flag15,
		Flag16:           flags.Flag16,
	}
}

func (t *transformer) FuelCapacityPercent() float32 {
	val := t.RawTelemetry.FuelCapacity

	return val
}

func (t *transformer) FuelLevelPercent() float32 {
	val := t.RawTelemetry.FuelLevel

	return val
}

func (t *transformer) GameVersion() string {
	isGT7, err := t.RawTelemetry.HeaderIsGt7()
	if err != nil && isGT7 {
		return "gt7"
	}

	isGT6, err := t.RawTelemetry.HeaderIsGt6()
	if err != nil && isGT6 {
		return "gt6"
	}

	return "unknown"
}

func (t *transformer) GroundSpeedMetersPerSecond() float32 {
	return t.RawTelemetry.GroundSpeed
}

func (t *transformer) Heading() float32 {
	return t.RawTelemetry.Heading
}

func (t *transformer) LastLaptime() time.Duration {
	return time.Duration(t.RawTelemetry.LastLaptime) * time.Millisecond
}

func (t *transformer) OilPressureKPA() float32 {
	return t.RawTelemetry.OilPressure
}

func (t *transformer) OilTemperatureCelsius() float32 {
	return t.RawTelemetry.OilTemperature
}

func (t *transformer) PositionalMapCoordinates() Vector {
	position := t.RawTelemetry.MapPositionCoordinates
	if position == nil {
		return Vector{}
	}

	return Vector{
		X: position.CoordinateX,
		Y: position.CoordinateY,
		Z: position.CoordinateZ,
	}
}

func (t *transformer) RaceEntrants() int16 {
	return t.RawTelemetry.RaceEntrants
}

func (t *transformer) RaceLaps() uint16 {
	return t.RawTelemetry.RaceLaps
}

func (t *transformer) RideHeightMeters() float32 {
	return t.RawTelemetry.RideHeight
}

func (t *transformer) RotationEnvelope() RotationalEnvelope {
	rotation := t.RawTelemetry.RotationalEnvelope
	if rotation == nil {
		return RotationalEnvelope{}
	}

	return RotationalEnvelope{
		Pitch: rotation.Pitch,
		Yaw:   rotation.Yaw,
		Roll:  rotation.Roll,
	}
}

// To be deprecated in 2.0 release in favor of RotationalEnvelope()
func (t *transformer) RotationVector() SymmetryAxes {
	rotation := t.RawTelemetry.RotationalEnvelope
	if rotation == nil {
		return SymmetryAxes{}
	}

	return SymmetryAxes{
		Pitch: rotation.Pitch,
		Yaw:   rotation.Yaw,
		Roll:  rotation.Roll,
	}
}

func (t *transformer) SequenceID() uint32 {
	return t.RawTelemetry.SequenceId
}

func (t *transformer) StartingPosition() int16 {
	return t.RawTelemetry.StartingPosition
}

func (t *transformer) SteeringWheelAngleDegrees() float32 {
	return utils.RadiansToDegrees(t.RawTelemetry.SteeringWheelAngleRadians)
}

func (t *transformer) SteeringWheelAngleRadians() float32 {
	return t.RawTelemetry.SteeringWheelAngleRadians
}

func (t *transformer) SteeringWheelForceFeedback() float32 {
	return t.RawTelemetry.SteeringWheelForceFeedback
}

func (t *transformer) SuggestedGear() uint64 {
	gear := t.RawTelemetry.TransmissionGear
	if gear == nil {
		return 15
	}

	return gear.Suggested
}

func (t *transformer) SuspensionHeightMeters() CornerSet {
	height := t.RawTelemetry.SuspensionHeight
	if height == nil {
		return CornerSet{}
	}

	return CornerSet{
		FrontLeft:  height.FrontLeft,
		FrontRight: height.FrontRight,
		RearLeft:   height.RearLeft,
		RearRight:  height.RearRight,
	}
}

func (t *transformer) TelemetryFormat() telemetrysrc.TelemetryFormat {
	isFormatTilde, err := t.RawTelemetry.HasSectionTilde()
	if err != nil && isFormatTilde {
		return telemetrysrc.TelemetryFormatTilde
	}

	isFormatB, err := t.RawTelemetry.HasSectionB()
	if err != nil && isFormatB {
		return telemetrysrc.TelemetryFormatB
	}

	isFormatA, err := t.RawTelemetry.HasSectionA()
	if err != nil && isFormatA {
		return telemetrysrc.TelemetryFormatA
	}

	return "unknown"
}

func (t *transformer) ThrottleInputPercent() float32 {
	return float32(t.RawTelemetry.ThrottleInput) / 2.55
}

func (t *transformer) ThrottleOutputPercent() float32 {
	return float32(t.RawTelemetry.ThrottleOutput) / 2.55
}

// To be deprecated in 2.0 release
func (t *transformer) ThrottlePercent() float32 {
	return float32(t.RawTelemetry.ThrottleOutput) / 2.55
}

func (t *transformer) TimeOfDay() time.Duration {
	return time.Duration(t.RawTelemetry.TimeOfDay) * time.Millisecond
}

func (t *transformer) TranslationEnvelope() TranslationalEnvelope {
	translation := t.RawTelemetry.TranslationalEnvelope
	if translation == nil {
		return TranslationalEnvelope{}
	}

	return TranslationalEnvelope{
		Sway:  translation.Sway,
		Heave: translation.Heave,
		Surge: translation.Surge,
	}
}

func (t *transformer) Transmission() Transmission {
	ratios := t.RawTelemetry.TransmissionGearRatio
	if ratios == nil {
		return Transmission{
			Gears:      0,
			GearRatios: make([]float32, 8),
		}
	}

	// TODO: figure out how to support vehicles with more than 8 gears (Lexus LC500)
	gearCount := 0
	for _, ratio := range ratios.Gear {
		if ratio > 0 {
			gearCount++
		}
	}

	return Transmission{
		Gears:      gearCount,
		GearRatios: ratios.Gear,
	}
}

func (t *transformer) TransmissionTopSpeedRatio() float32 {
	return t.RawTelemetry.TransmissionTopSpeedRatio
}

func (t *transformer) TurboBoostBar() float32 {
	return (t.RawTelemetry.ManifoldPressure - 1)
}

func (t *transformer) TyreDiameterMeters() CornerSet {
	radius := t.RawTelemetry.TyreRadius
	if radius == nil {
		return CornerSet{}
	}

	return CornerSet{
		FrontLeft:  radius.FrontLeft * 2,
		FrontRight: radius.FrontRight * 2,
		RearLeft:   radius.RearLeft * 2,
		RearRight:  radius.RearRight * 2,
	}
}

func (t *transformer) TyreRadiusMeters() CornerSet {
	radius := t.RawTelemetry.TyreRadius
	if radius == nil {
		return CornerSet{}
	}

	return CornerSet{
		FrontLeft:  radius.FrontLeft,
		FrontRight: radius.FrontRight,
		RearLeft:   radius.RearLeft,
		RearRight:  radius.RearRight,
	}
}

func (t *transformer) TyreSlipRatio() CornerSet {
	groundSpeed := utils.MetersPerSecondToKilometersPerHour(t.GroundSpeedMetersPerSecond())
	wheelSpeed := t.WheelSpeedMetersPerSecond()
	if groundSpeed == 0 {
		return CornerSet{
			FrontLeft:  1,
			FrontRight: 1,
			RearLeft:   1,
			RearRight:  1,
		}
	}

	return CornerSet{
		FrontLeft:  utils.MetersPerSecondToKilometersPerHour(wheelSpeed.FrontLeft) / groundSpeed,
		FrontRight: utils.MetersPerSecondToKilometersPerHour(wheelSpeed.FrontRight) / groundSpeed,
		RearLeft:   utils.MetersPerSecondToKilometersPerHour(wheelSpeed.RearLeft) / groundSpeed,
		RearRight:  utils.MetersPerSecondToKilometersPerHour(wheelSpeed.RearRight) / groundSpeed,
	}
}

func (t *transformer) TyreTemperatureCelsius() CornerSet {
	temperature := t.RawTelemetry.TyreTemperature
	if temperature == nil {
		return CornerSet{}
	}

	return CornerSet{
		FrontLeft:  temperature.FrontLeft,
		FrontRight: temperature.FrontRight,
		RearLeft:   temperature.RearLeft,
		RearRight:  temperature.RearRight,
	}
}

func (t *transformer) Unknown0x13E() uint8 {
	return t.RawTelemetry.Unknown0x13e
}

func (t *transformer) Unknown0x13F() uint8 {
	return t.RawTelemetry.Unknown0x13f
}

func (t *transformer) Unknown0x140() float32 {
	return t.RawTelemetry.Unknown0x140
}

func (t *transformer) Unknown0x144() float32 {
	return t.RawTelemetry.Unknown0x144
}

func (t *transformer) Unknown0x148() float32 {
	return t.RawTelemetry.Unknown0x148
}

func (t *transformer) Unknown0x14C() float32 {
	return t.RawTelemetry.Unknown0x14c
}

func (t *transformer) Unknown0x154() float32 {
	return t.RawTelemetry.Unknown0x154
}

func (t *transformer) VehicleAspiration() string {
	t.updateVehicle()

	return t.vehicle.Aspiration
}

func (t *transformer) VehicleAspirationExpanded() string {
	t.updateVehicle()

	return t.vehicle.ExpandedAspiration()
}

func (t *transformer) VehicleEngineLayout() string {
	t.updateVehicle()

	return t.vehicle.EngineLayout
}

func (t *transformer) VehicleEngineBankAngle() float32 {
	t.updateVehicle()

	return t.vehicle.EngineBankAngle
}

func (t *transformer) VehicleEngineCrankPlaneAngle() float32 {
	t.updateVehicle()

	return t.vehicle.EngineCrankPlaneAngle
}

func (t *transformer) VehicleCategory() string {
	t.updateVehicle()

	return t.vehicle.Category
}

func (t *transformer) VehicleDrivetrain() string {
	t.updateVehicle()

	return t.vehicle.Drivetrain
}

func (t *transformer) VehicleHasOpenCockpit() bool {
	t.updateVehicle()

	return t.vehicle.OpenCockpit
}

func (t *transformer) VehicleID() uint32 {
	t.updateVehicle()

	return uint32(t.vehicle.CarID)
}

func (t *transformer) VehicleManufacturer() string {
	t.updateVehicle()

	return t.vehicle.Manufacturer
}

func (t *transformer) VehicleModel() string {
	t.updateVehicle()

	return t.vehicle.Model
}

func (t *transformer) VehicleType() string {
	t.updateVehicle()

	return t.vehicle.CarType
}

func (t *transformer) VehicleYear() int {
	t.updateVehicle()

	return t.vehicle.Year
}

func (t *transformer) VelocityVector() Vector {
	velocity := t.RawTelemetry.VelocityVector
	if velocity == nil {
		return Vector{}
	}

	return Vector{
		X: velocity.VectorX,
		Y: velocity.VectorY,
		Z: velocity.VectorZ,
	}
}

func (t *transformer) WheelSpeedMetersPerSecond() CornerSet {
	radius := t.TyreRadiusMeters()
	rps := t.WheelSpeedRadiansPerSecond()

	return CornerSet{
		FrontLeft:  rps.FrontLeft * radius.FrontLeft,
		FrontRight: rps.FrontRight * radius.FrontRight,
		RearLeft:   rps.RearLeft * radius.RearLeft,
		RearRight:  rps.RearRight * radius.RearLeft,
	}
}

func (t *transformer) WheelSpeedRadiansPerSecond() CornerSet {
	rps := t.RawTelemetry.WheelRadiansPerSecond
	if rps == nil {
		return CornerSet{}
	}

	return CornerSet{
		FrontLeft:  float32(math.Abs(float64(rps.FrontLeft))),
		FrontRight: float32(math.Abs(float64(rps.FrontRight))),
		RearLeft:   float32(math.Abs(float64(rps.RearLeft))),
		RearRight:  float32(math.Abs(float64(rps.RearRight))),
	}
}

func (t *transformer) WaterTemperatureCelsius() float32 {
	return t.RawTelemetry.WaterTemperature
}

func (t *transformer) updateVehicle() {
	vehicleID := int(t.RawTelemetry.VehicleId)

	if t.vehicle.CarID != vehicleID {
		vehicle, err := t.inventory.GetVehicleByID(vehicleID)
		if err != nil {
			vehicle = vehicles.Vehicle{
				CarID: vehicleID,
			}
		}

		t.vehicle = vehicle
	}
}
