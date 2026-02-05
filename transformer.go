package gttelemetry

import (
	"math"
	"time"

	"github.com/zetetos/gt-telemetry/internal/telemetry"
	"github.com/zetetos/gt-telemetry/internal/units"
	"github.com/zetetos/gt-telemetry/pkg/models"
	"github.com/zetetos/gt-telemetry/pkg/vehicles"
)

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

type Vmax struct {
	Speed uint16
	RPM   uint16
}

type Transformer struct {
	RawTelemetry telemetry.GranTurismoTelemetry
	inventory    *vehicles.VehicleDB
	Vehicle      vehicles.Vehicle
}

func NewTransformer(inventory *vehicles.VehicleDB) *Transformer {
	return &Transformer{
		RawTelemetry: telemetry.GranTurismoTelemetry{},
		inventory:    inventory,
		Vehicle:      vehicles.Vehicle{},
	}
}

func (t *Transformer) AngularVelocityVector() models.Vector {
	velocity := t.RawTelemetry.AngularVelocityVector
	if velocity == nil {
		return models.Vector{}
	}

	return models.Vector{
		X: velocity.VectorX,
		Y: velocity.VectorY,
		Z: velocity.VectorZ,
	}
}

func (t *Transformer) BestLaptime() time.Duration {
	return time.Duration(t.RawTelemetry.BestLaptime) * time.Millisecond
}

func (t *Transformer) BrakeInputPercent() float32 {
	return float32(t.RawTelemetry.BrakeInput) / 2.55
}

func (t *Transformer) BrakeOutputPercent() float32 {
	return float32(t.RawTelemetry.BrakeOutput) / 2.55
}

func (t *Transformer) CalculatedVmax() Vmax {
	vMaxSpeed := t.RawTelemetry.CalculatedMaxSpeed
	vMaxMetresPerMinute := float32(vMaxSpeed) * 1000 / 60
	tyreCircumference := t.TyreDiameterMetres().RearLeft * math.Pi

	return Vmax{
		Speed: vMaxSpeed,
		RPM:   uint16((vMaxMetresPerMinute / tyreCircumference) * t.TransmissionTopSpeedRatio()),
	}
}

func (t *Transformer) ClutchActuationPercent() float32 {
	return t.RawTelemetry.ClutchActuation * 100
}

func (t *Transformer) ClutchEngagementPercent() float32 {
	return t.RawTelemetry.ClutchEngagement * 100
}

func (t *Transformer) ClutchOutputRPM() float32 {
	return t.RawTelemetry.CluchOutputRpm
}

// CurrentGear returns the currently selected transmission gear, 15 is neutral.
func (t *Transformer) CurrentGear() int {
	gear := t.RawTelemetry.TransmissionGear
	if gear == nil {
		return 15
	}

	return int(gear.Current) //nolint:gosec // Value will always be a small positive integer
}

func (t *Transformer) CurrentGearRatio() float32 {
	gear := t.CurrentGear()
	if gear > len(t.Transmission().GearRatios) {
		return -1
	}

	return t.Transmission().GearRatios[gear-1]
}

func (t *Transformer) CurrentLap() int16 {
	return t.RawTelemetry.CurrentLap
}

func (t *Transformer) DifferentialRatio() float32 {
	t.UpdateVehicle()

	transmission := t.Transmission()
	if transmission.Gears == 0 {
		return -1
	}

	highestRatio := transmission.GearRatios[transmission.Gears-1]
	vMax := t.CalculatedVmax()

	var rollingDiameter float32

	switch t.Vehicle.Drivetrain {
	case "FF":
		rollingDiameter = t.TyreDiameterMetres().FrontLeft
	default:
		rollingDiameter = t.TyreDiameterMetres().RearLeft
	}

	vMaxMetresPerMinute := float32(vMax.Speed) * 1000 / 60
	wheelRpm := vMaxMetresPerMinute / (rollingDiameter * math.Pi)
	diffRatio := (float32(vMax.RPM) / highestRatio) / wheelRpm

	return diffRatio
}

func (t *Transformer) EnergyRecovery() float32 {
	return t.RawTelemetry.EnergyRecovery
}

func (t *Transformer) EngineRPM() float32 {
	val := t.RawTelemetry.EngineRpm

	return val
}

func (t *Transformer) EngineRPMLight() RevLight {
	rpm := uint16(t.EngineRPM())
	lightMin := t.RawTelemetry.RevLightRpmMin
	lightMax := t.RawTelemetry.RevLightRpmMax

	active := rpm > lightMin

	return RevLight{
		Min:    lightMin,
		Max:    lightMax,
		Active: active,
	}
}

func (t *Transformer) Flags() Flags {
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

func (t *Transformer) FuelCapacity() float32 {
	val := t.RawTelemetry.FuelCapacity

	return val
}

func (t *Transformer) FuelLevel() float32 {
	val := t.RawTelemetry.FuelLevel

	return val
}

func (t *Transformer) FuelLevelPercent() float32 {
	val := t.RawTelemetry.FuelLevel / t.RawTelemetry.FuelCapacity

	return val * 100
}

func (t *Transformer) GameState() models.GameState {
	if t.IsInMainMenu() {
		return models.GameStateMainMenu
	}

	if t.IsInRaceMenu() {
		return models.GameStateRaceMenu
	}

	if t.IsOnCircuit() {
		if t.Flags().Live {
			return models.GameStateLive
		}

		return models.GameStateReplay
	}

	return models.GameStateUnknown
}

func (t *Transformer) GameVersion() string {
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

func (t *Transformer) GridPosition() int16 {
	return t.RawTelemetry.GridPosition
}

func (t *Transformer) GroundSpeedMetresPerSecond() float32 {
	return t.RawTelemetry.GroundSpeed
}

func (t *Transformer) Heading() float32 {
	return t.RawTelemetry.Heading
}

func (t *Transformer) IsInMainMenu() bool {
	if t.RawTelemetry.RaceLaps < 0 && t.RawTelemetry.RaceEntrants < 0 {
		return true
	}

	return false
}

func (t *Transformer) IsInRaceMenu() bool {
	if t.RawTelemetry.RaceLaps >= 0 && t.RawTelemetry.RaceEntrants < 0 {
		return true
	}

	return false
}

func (t *Transformer) IsOnCircuit() bool {
	if t.RawTelemetry.RaceLaps >= 0 && t.RawTelemetry.RaceEntrants >= 0 {
		return true
	}

	return false
}

func (t *Transformer) LastLaptime() time.Duration {
	return time.Duration(t.RawTelemetry.LastLaptime) * time.Millisecond
}

func (t *Transformer) RaceComplete() bool {
	if t.RawTelemetry.RaceLaps < 1 {
		return false
	}

	return t.RawTelemetry.CurrentLap > t.RawTelemetry.RaceLaps
}

func (t *Transformer) OilPressureKPA() float32 {
	return t.RawTelemetry.OilPressure
}

func (t *Transformer) OilTemperatureCelsius() float32 {
	return t.RawTelemetry.OilTemperature
}

func (t *Transformer) PositionalMapCoordinates() models.Coordinate {
	position := t.RawTelemetry.MapPositionCoordinates
	if position == nil {
		return models.Coordinate{}
	}

	return models.Coordinate{
		X: position.CoordinateX,
		Y: position.CoordinateY,
		Z: position.CoordinateZ,
	}
}

func (t *Transformer) RaceEntrants() int16 {
	return t.RawTelemetry.RaceEntrants
}

func (t *Transformer) RaceLaps() int16 {
	return t.RawTelemetry.RaceLaps
}

func (t *Transformer) RaceType() models.RaceType {
	if !t.IsOnCircuit() {
		return models.RaceTypeUnknown
	}

	if t.RawTelemetry.RaceEntrants <= 3 && t.RawTelemetry.RaceLaps == 0 {
		return models.RaceTypeTimeTrial
	}

	if t.RawTelemetry.RaceEntrants > 3 && t.RawTelemetry.RaceLaps == 0 {
		return models.RaceTypeEndurance
	}

	if t.RawTelemetry.RaceEntrants > 3 && t.RawTelemetry.RaceLaps > 0 {
		return models.RaceTypeSprint
	}

	return models.RaceTypeUnknown
}

func (t *Transformer) RideHeightMetres() float32 {
	return t.RawTelemetry.RideHeight
}

func (t *Transformer) RotationEnvelope() models.RotationalEnvelope {
	rotation := t.RawTelemetry.RotationalEnvelope
	if rotation == nil {
		return models.RotationalEnvelope{}
	}

	return models.RotationalEnvelope{
		Pitch: rotation.Pitch,
		Yaw:   rotation.Yaw,
		Roll:  rotation.Roll,
	}
}

func (t *Transformer) SequenceID() uint32 {
	return t.RawTelemetry.SequenceId
}

func (t *Transformer) SteeringWheelAngleDegrees() float32 {
	return units.RadiansToDegrees(t.RawTelemetry.SteeringWheelAngleRadians)
}

func (t *Transformer) SteeringWheelAngleRadians() float32 {
	return t.RawTelemetry.SteeringWheelAngleRadians
}

func (t *Transformer) SteeringWheelForceFeedback() float32 {
	return t.RawTelemetry.SteeringWheelForceFeedback
}

func (t *Transformer) SuggestedGear() uint64 {
	gear := t.RawTelemetry.TransmissionGear
	if gear == nil {
		return 15
	}

	return gear.Suggested
}

func (t *Transformer) SuspensionHeightMetres() models.CornerSet {
	height := t.RawTelemetry.SuspensionHeight
	if height == nil {
		return models.CornerSet{}
	}

	return models.CornerSet{
		FrontLeft:  height.FrontLeft,
		FrontRight: height.FrontRight,
		RearLeft:   height.RearLeft,
		RearRight:  height.RearRight,
	}
}

func (t *Transformer) TelemetryFormat() models.Name {
	isAddendum2Format, err := t.RawTelemetry.Addendum2Format()
	if err != nil && isAddendum2Format {
		return models.Addendum2
	}

	isAddendum1Format, err := t.RawTelemetry.Addendum1Format()
	if err != nil && isAddendum1Format {
		return models.Addendum1
	}

	isStandardFormat, err := t.RawTelemetry.StandardFormat()
	if err != nil && isStandardFormat {
		return models.Standard
	}

	return "unknown"
}

func (t *Transformer) TelemetryStarted() bool {
	return t.RawTelemetry.SequenceId > 0
}

func (t *Transformer) ThrottleInputPercent() float32 {
	return float32(t.RawTelemetry.ThrottleInput) / 2.55
}

func (t *Transformer) ThrottleOutputPercent() float32 {
	return float32(t.RawTelemetry.ThrottleOutput) / 2.55
}

func (t *Transformer) TimeOfDay() time.Duration {
	return time.Duration(t.RawTelemetry.TimeOfDay) * time.Millisecond
}

func (t *Transformer) TranslationEnvelope() models.TranslationalEnvelope {
	translation := t.RawTelemetry.TranslationalEnvelope
	if translation == nil {
		return models.TranslationalEnvelope{}
	}

	return models.TranslationalEnvelope{
		Sway:  translation.Sway,
		Heave: translation.Heave,
		Surge: translation.Surge,
	}
}

func (t *Transformer) Transmission() Transmission {
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

func (t *Transformer) TransmissionTopSpeedRatio() float32 {
	return t.RawTelemetry.TransmissionTopSpeedRatio
}

func (t *Transformer) TurboBoostBar() float32 {
	return (t.RawTelemetry.ManifoldPressure - 1)
}

func (t *Transformer) TyreDiameterMetres() models.CornerSet {
	radius := t.RawTelemetry.TyreRadius
	if radius == nil {
		return models.CornerSet{}
	}

	return models.CornerSet{
		FrontLeft:  radius.FrontLeft * 2,
		FrontRight: radius.FrontRight * 2,
		RearLeft:   radius.RearLeft * 2,
		RearRight:  radius.RearRight * 2,
	}
}

func (t *Transformer) TyreRadiusMetres() models.CornerSet {
	radius := t.RawTelemetry.TyreRadius
	if radius == nil {
		return models.CornerSet{}
	}

	return models.CornerSet{
		FrontLeft:  radius.FrontLeft,
		FrontRight: radius.FrontRight,
		RearLeft:   radius.RearLeft,
		RearRight:  radius.RearRight,
	}
}

func (t *Transformer) TyreSlipRatio() models.CornerSet {
	groundSpeed := units.MetresPerSecondToKilometresPerHour(t.GroundSpeedMetresPerSecond())
	wheelSpeed := t.WheelSpeedMetresPerSecond()

	if groundSpeed < 0.0001 {
		return models.CornerSet{
			FrontLeft:  1,
			FrontRight: 1,
			RearLeft:   1,
			RearRight:  1,
		}
	}

	return models.CornerSet{
		FrontLeft:  units.MetresPerSecondToKilometresPerHour(wheelSpeed.FrontLeft) / groundSpeed,
		FrontRight: units.MetresPerSecondToKilometresPerHour(wheelSpeed.FrontRight) / groundSpeed,
		RearLeft:   units.MetresPerSecondToKilometresPerHour(wheelSpeed.RearLeft) / groundSpeed,
		RearRight:  units.MetresPerSecondToKilometresPerHour(wheelSpeed.RearRight) / groundSpeed,
	}
}

func (t *Transformer) TyreTemperatureCelsius() models.CornerSet {
	temperature := t.RawTelemetry.TyreTemperature
	if temperature == nil {
		return models.CornerSet{}
	}

	return models.CornerSet{
		FrontLeft:  temperature.FrontLeft,
		FrontRight: temperature.FrontRight,
		RearLeft:   temperature.RearLeft,
		RearRight:  temperature.RearRight,
	}
}

func (t *Transformer) Unknown0x13E() uint8 {
	return t.RawTelemetry.Unknown0x13e
}

func (t *Transformer) Unknown0x13F() uint8 {
	return t.RawTelemetry.Unknown0x13f
}

func (t *Transformer) Unknown0x140() float32 {
	return t.RawTelemetry.Unknown0x140
}

func (t *Transformer) Unknown0x144() float32 {
	return t.RawTelemetry.Unknown0x144
}

func (t *Transformer) Unknown0x148() float32 {
	return t.RawTelemetry.Unknown0x148
}

func (t *Transformer) Unknown0x14C() float32 {
	return t.RawTelemetry.Unknown0x14c
}

func (t *Transformer) Unknown0x154() float32 {
	return t.RawTelemetry.Unknown0x154
}

func (t *Transformer) VehicleAspiration() string {
	t.UpdateVehicle()

	return t.Vehicle.Aspiration
}

func (t *Transformer) VehicleAspirationExpanded() string {
	t.UpdateVehicle()

	return t.Vehicle.ExpandedAspiration()
}

func (t *Transformer) VehicleEngineLayout() string {
	t.UpdateVehicle()

	return t.Vehicle.EngineLayout
}

func (t *Transformer) VehicleEngineBankAngle() float32 {
	t.UpdateVehicle()

	return t.Vehicle.EngineBankAngle
}

func (t *Transformer) VehicleEngineCrankPlaneAngle() float32 {
	t.UpdateVehicle()

	return t.Vehicle.EngineCrankPlaneAngle
}

func (t *Transformer) VehicleCategory() string {
	t.UpdateVehicle()

	return t.Vehicle.Category
}

func (t *Transformer) VehicleDrivetrain() string {
	t.UpdateVehicle()

	return t.Vehicle.Drivetrain
}

func (t *Transformer) VehicleHasOpenCockpit() bool {
	t.UpdateVehicle()

	return t.Vehicle.OpenCockpit
}

func (t *Transformer) VehicleID() uint32 {
	t.UpdateVehicle()

	return uint32(t.Vehicle.CarID) //nolint:gosec // TODO: might be an issue with the -10000 validation ID
}

func (t *Transformer) VehicleManufacturer() string {
	t.UpdateVehicle()

	return t.Vehicle.Manufacturer
}

func (t *Transformer) VehicleModel() string {
	t.UpdateVehicle()

	return t.Vehicle.Model
}

func (t *Transformer) VehicleType() string {
	t.UpdateVehicle()

	return t.Vehicle.CarType
}

func (t *Transformer) VehicleYear() int {
	t.UpdateVehicle()

	return t.Vehicle.Year
}

func (t *Transformer) VehicleLengthMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.Length
}

func (t *Transformer) VehicleWidthMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.Width
}

func (t *Transformer) VehicleHeightMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.Height
}

func (t *Transformer) VehicleWheelbaseMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.Wheelbase
}

func (t *Transformer) VehicleTrackFrontMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.TrackFront
}

func (t *Transformer) VehicleTrackRearMillimetres() int {
	t.UpdateVehicle()

	return t.Vehicle.TrackRear
}

func (t *Transformer) VelocityVector() models.Vector {
	velocity := t.RawTelemetry.VelocityVector
	if velocity == nil {
		return models.Vector{}
	}

	return models.Vector{
		X: velocity.VectorX,
		Y: velocity.VectorY,
		Z: velocity.VectorZ,
	}
}

func (t *Transformer) WheelSpeedMetresPerSecond() models.CornerSet {
	radius := t.TyreRadiusMetres()
	rps := t.WheelSpeedRadiansPerSecond()

	return models.CornerSet{
		FrontLeft:  rps.FrontLeft * radius.FrontLeft,
		FrontRight: rps.FrontRight * radius.FrontRight,
		RearLeft:   rps.RearLeft * radius.RearLeft,
		RearRight:  rps.RearRight * radius.RearLeft,
	}
}

func (t *Transformer) WheelSpeedRadiansPerSecond() models.CornerSet {
	rps := t.RawTelemetry.WheelRadiansPerSecond
	if rps == nil {
		return models.CornerSet{}
	}

	return models.CornerSet{
		FrontLeft:  float32(math.Abs(float64(rps.FrontLeft))),
		FrontRight: float32(math.Abs(float64(rps.FrontRight))),
		RearLeft:   float32(math.Abs(float64(rps.RearLeft))),
		RearRight:  float32(math.Abs(float64(rps.RearRight))),
	}
}

func (t *Transformer) WaterTemperatureCelsius() float32 {
	return t.RawTelemetry.WaterTemperature
}

func (t *Transformer) UpdateVehicle() {
	if t.IsInMainMenu() {
		t.Vehicle = vehicles.Vehicle{}

		return
	}

	vehicleID := int(t.RawTelemetry.VehicleId)

	if t.Vehicle.CarID != vehicleID {
		vehicle, err := t.inventory.GetVehicleByID(vehicleID)
		if err != nil {
			vehicle = vehicles.Vehicle{
				CarID: vehicleID,
			}
		}

		t.Vehicle = vehicle
	}
}
