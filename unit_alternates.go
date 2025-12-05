package gttelemetry

import (
	"strconv"

	"github.com/zetetos/gt-telemetry/internal/units"
	"github.com/zetetos/gt-telemetry/pkg/models"
)

func (t *Transformer) CurrentGearString() string {
	gear := strconv.Itoa(t.CurrentGear())
	switch gear {
	case "0":
		gear = "R"
	case "15":
		gear = "N"
	}

	return gear
}

func (t *Transformer) GroundSpeedKPH() float32 {
	return units.MetersPerSecondToKilometersPerHour(t.GroundSpeedMetersPerSecond())
}

func (t *Transformer) OilTemperatureFahrenheit() float32 {
	return units.CelsiusToFahrenheit(t.RawTelemetry.OilTemperature)
}

func (t *Transformer) RideHeightMillimeters() float32 {
	return units.MetersToMillimeters(t.RideHeightMeters())
}

func (t *Transformer) SuspensionHeightFeet() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToFeet(set.FrontLeft),
		FrontRight: units.MetersToFeet(set.FrontRight),
		RearLeft:   units.MetersToFeet(set.RearLeft),
		RearRight:  units.MetersToFeet(set.RearRight),
	}
}

func (t *Transformer) SuspensionHeightInches() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToInches(set.FrontLeft),
		FrontRight: units.MetersToInches(set.FrontRight),
		RearLeft:   units.MetersToInches(set.RearLeft),
		RearRight:  units.MetersToInches(set.RearRight),
	}
}

func (t *Transformer) SuspensionHeightMillimeters() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToMillimeters(set.FrontLeft),
		FrontRight: units.MetersToMillimeters(set.FrontRight),
		RearLeft:   units.MetersToMillimeters(set.RearLeft),
		RearRight:  units.MetersToMillimeters(set.RearRight),
	}
}

func (t *Transformer) TurboBoostPSI() float32 {
	return units.BarToPSI(t.TurboBoostBar())
}

func (t *Transformer) TurboBoostInHg() float32 {
	return units.BarToInHg(t.TurboBoostBar())
}

func (t *Transformer) TurboBoostKPA() float32 {
	return units.BarToKPA(t.TurboBoostBar())
}

func (t *Transformer) TyreDiameterFeet() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToFeet(set.FrontLeft),
		FrontRight: units.MetersToFeet(set.FrontRight),
		RearLeft:   units.MetersToFeet(set.RearLeft),
		RearRight:  units.MetersToFeet(set.RearRight),
	}
}

func (t *Transformer) TyreDiameterInches() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToInches(set.FrontLeft),
		FrontRight: units.MetersToInches(set.FrontRight),
		RearLeft:   units.MetersToInches(set.RearLeft),
		RearRight:  units.MetersToInches(set.RearRight),
	}
}

func (t *Transformer) TyreDiameterMillimeters() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToMillimeters(set.FrontLeft),
		FrontRight: units.MetersToMillimeters(set.FrontRight),
		RearLeft:   units.MetersToMillimeters(set.RearLeft),
		RearRight:  units.MetersToMillimeters(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusFeet() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToFeet(set.FrontLeft),
		FrontRight: units.MetersToFeet(set.FrontRight),
		RearLeft:   units.MetersToFeet(set.RearLeft),
		RearRight:  units.MetersToFeet(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusInches() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToInches(set.FrontLeft),
		FrontRight: units.MetersToInches(set.FrontRight),
		RearLeft:   units.MetersToInches(set.RearLeft),
		RearRight:  units.MetersToInches(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusMillimeters() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  units.MetersToMillimeters(set.FrontLeft),
		FrontRight: units.MetersToMillimeters(set.FrontRight),
		RearLeft:   units.MetersToMillimeters(set.RearLeft),
		RearRight:  units.MetersToMillimeters(set.RearRight),
	}
}

func (t *Transformer) TyreTemperatureFahrenheit() models.CornerSet {
	set := t.TyreTemperatureCelsius()

	return models.CornerSet{
		FrontLeft:  units.MetersPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: units.MetersPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   units.MetersPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  units.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *Transformer) WheelSpeedKPH() models.CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return models.CornerSet{
		FrontLeft:  units.MetersPerSecondToKilometersPerHour(set.FrontLeft),
		FrontRight: units.MetersPerSecondToKilometersPerHour(set.FrontRight),
		RearLeft:   units.MetersPerSecondToKilometersPerHour(set.RearLeft),
		RearRight:  units.MetersPerSecondToKilometersPerHour(set.RearRight),
	}
}

func (t *Transformer) WheelSpeedMPH() models.CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return models.CornerSet{
		FrontLeft:  units.MetersPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: units.MetersPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   units.MetersPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  units.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *Transformer) WheelSpeedRPM() models.CornerSet {
	rps := t.WheelSpeedRadiansPerSecond()

	return models.CornerSet{
		FrontLeft:  units.RadiansPerSecondToRevolutionsPerMinute(rps.FrontLeft),
		FrontRight: units.RadiansPerSecondToRevolutionsPerMinute(rps.FrontRight),
		RearLeft:   units.RadiansPerSecondToRevolutionsPerMinute(rps.RearLeft),
		RearRight:  units.RadiansPerSecondToRevolutionsPerMinute(rps.RearRight),
	}
}

func (t *Transformer) WaterTemperatureFahrenheit() float32 {
	return units.CelsiusToFahrenheit(t.RawTelemetry.WaterTemperature)
}

func (t *Transformer) VehicleLengthInches() float32 {
	return units.MillimetersToInches(t.VehicleLengthMillimeters())
}

func (t *Transformer) VehicleWidthInches() float32 {
	return units.MillimetersToInches(t.VehicleWidthMillimeters())
}

func (t *Transformer) VehicleHeightInches() float32 {
	return units.MillimetersToInches(t.VehicleHeightMillimeters())
}

func (t *Transformer) VehicleWheelbaseInches() float32 {
	return units.MillimetersToInches(t.VehicleWheelbaseMillimeters())
}

func (t *Transformer) VehicleTrackFrontInches() float32 {
	return units.MillimetersToInches(t.VehicleTrackFrontMillimeters())
}

func (t *Transformer) VehicleTrackRearInches() float32 {
	return units.MillimetersToInches(t.VehicleTrackRearMillimeters())
}
