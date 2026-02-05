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
	return units.MetresPerSecondToKilometresPerHour(t.GroundSpeedMetresPerSecond())
}

func (t *Transformer) OilTemperatureFahrenheit() float32 {
	return units.CelsiusToFahrenheit(t.RawTelemetry.OilTemperature)
}

func (t *Transformer) RideHeightMillimetres() float32 {
	return units.MetresToMillimetres(t.RideHeightMetres())
}

func (t *Transformer) SuspensionHeightFeet() models.CornerSet {
	set := t.SuspensionHeightMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToFeet(set.FrontLeft),
		FrontRight: units.MetresToFeet(set.FrontRight),
		RearLeft:   units.MetresToFeet(set.RearLeft),
		RearRight:  units.MetresToFeet(set.RearRight),
	}
}

func (t *Transformer) SuspensionHeightInches() models.CornerSet {
	set := t.SuspensionHeightMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToInches(set.FrontLeft),
		FrontRight: units.MetresToInches(set.FrontRight),
		RearLeft:   units.MetresToInches(set.RearLeft),
		RearRight:  units.MetresToInches(set.RearRight),
	}
}

func (t *Transformer) SuspensionHeightMillimetres() models.CornerSet {
	set := t.SuspensionHeightMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToMillimetres(set.FrontLeft),
		FrontRight: units.MetresToMillimetres(set.FrontRight),
		RearLeft:   units.MetresToMillimetres(set.RearLeft),
		RearRight:  units.MetresToMillimetres(set.RearRight),
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
	set := t.TyreDiameterMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToFeet(set.FrontLeft),
		FrontRight: units.MetresToFeet(set.FrontRight),
		RearLeft:   units.MetresToFeet(set.RearLeft),
		RearRight:  units.MetresToFeet(set.RearRight),
	}
}

func (t *Transformer) TyreDiameterInches() models.CornerSet {
	set := t.TyreDiameterMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToInches(set.FrontLeft),
		FrontRight: units.MetresToInches(set.FrontRight),
		RearLeft:   units.MetresToInches(set.RearLeft),
		RearRight:  units.MetresToInches(set.RearRight),
	}
}

func (t *Transformer) TyreDiameterMillimetres() models.CornerSet {
	set := t.TyreDiameterMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToMillimetres(set.FrontLeft),
		FrontRight: units.MetresToMillimetres(set.FrontRight),
		RearLeft:   units.MetresToMillimetres(set.RearLeft),
		RearRight:  units.MetresToMillimetres(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusFeet() models.CornerSet {
	set := t.TyreRadiusMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToFeet(set.FrontLeft),
		FrontRight: units.MetresToFeet(set.FrontRight),
		RearLeft:   units.MetresToFeet(set.RearLeft),
		RearRight:  units.MetresToFeet(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusInches() models.CornerSet {
	set := t.TyreRadiusMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToInches(set.FrontLeft),
		FrontRight: units.MetresToInches(set.FrontRight),
		RearLeft:   units.MetresToInches(set.RearLeft),
		RearRight:  units.MetresToInches(set.RearRight),
	}
}

func (t *Transformer) TyreRadiusMillimetres() models.CornerSet {
	set := t.TyreRadiusMetres()

	return models.CornerSet{
		FrontLeft:  units.MetresToMillimetres(set.FrontLeft),
		FrontRight: units.MetresToMillimetres(set.FrontRight),
		RearLeft:   units.MetresToMillimetres(set.RearLeft),
		RearRight:  units.MetresToMillimetres(set.RearRight),
	}
}

func (t *Transformer) TyreTemperatureFahrenheit() models.CornerSet {
	set := t.TyreTemperatureCelsius()

	return models.CornerSet{
		FrontLeft:  units.MetresPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: units.MetresPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   units.MetresPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  units.MetresPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *Transformer) WheelSpeedKPH() models.CornerSet {
	set := t.WheelSpeedMetresPerSecond()

	return models.CornerSet{
		FrontLeft:  units.MetresPerSecondToKilometresPerHour(set.FrontLeft),
		FrontRight: units.MetresPerSecondToKilometresPerHour(set.FrontRight),
		RearLeft:   units.MetresPerSecondToKilometresPerHour(set.RearLeft),
		RearRight:  units.MetresPerSecondToKilometresPerHour(set.RearRight),
	}
}

func (t *Transformer) WheelSpeedMPH() models.CornerSet {
	set := t.WheelSpeedMetresPerSecond()

	return models.CornerSet{
		FrontLeft:  units.MetresPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: units.MetresPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   units.MetresPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  units.MetresPerSecondToMilesPerHour(set.RearRight),
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
	return units.MillimetresToInches(t.VehicleLengthMillimetres())
}

func (t *Transformer) VehicleWidthInches() float32 {
	return units.MillimetresToInches(t.VehicleWidthMillimetres())
}

func (t *Transformer) VehicleHeightInches() float32 {
	return units.MillimetresToInches(t.VehicleHeightMillimetres())
}

func (t *Transformer) VehicleWheelbaseInches() float32 {
	return units.MillimetresToInches(t.VehicleWheelbaseMillimetres())
}

func (t *Transformer) VehicleTrackFrontInches() float32 {
	return units.MillimetresToInches(t.VehicleTrackFrontMillimetres())
}

func (t *Transformer) VehicleTrackRearInches() float32 {
	return units.MillimetresToInches(t.VehicleTrackRearMillimetres())
}
