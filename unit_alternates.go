package gttelemetry

import (
	"fmt"

	"github.com/zetetos/gt-telemetry/internal/utils"
	"github.com/zetetos/gt-telemetry/pkg/models"
)

func (t *transformer) CurrentGearString() string {
	gear := fmt.Sprint(t.CurrentGear())
	switch gear {
	case "0":
		gear = "R"
	case "15":
		gear = "N"
	}
	return gear
}

func (t *transformer) GroundSpeedKPH() float32 {
	return utils.MetersPerSecondToKilometersPerHour(t.GroundSpeedMetersPerSecond())
}

func (t *transformer) OilTemperatureFahrenheit() float32 {
	return utils.CelsiusToFahrenheit(t.RawTelemetry.OilTemperature)
}

func (t *transformer) RideHeightMillimeters() float32 {
	return utils.MetersToMillimeters(t.RideHeightMeters())
}

func (t *transformer) SuspensionHeightFeet() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToFeet(set.FrontLeft),
		FrontRight: utils.MetersToFeet(set.FrontRight),
		RearLeft:   utils.MetersToFeet(set.RearLeft),
		RearRight:  utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) SuspensionHeightInches() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToInches(set.FrontLeft),
		FrontRight: utils.MetersToInches(set.FrontRight),
		RearLeft:   utils.MetersToInches(set.RearLeft),
		RearRight:  utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) SuspensionHeightMillimeters() models.CornerSet {
	set := t.SuspensionHeightMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToMillimeters(set.FrontLeft),
		FrontRight: utils.MetersToMillimeters(set.FrontRight),
		RearLeft:   utils.MetersToMillimeters(set.RearLeft),
		RearRight:  utils.MetersToMillimeters(set.RearRight),
	}
}

func (t *transformer) TurboBoostPSI() float32 {
	return utils.BarToPSI(t.TurboBoostBar())
}

func (t *transformer) TurboBoostInHg() float32 {
	return utils.BarToInHg(t.TurboBoostBar())
}

func (t *transformer) TurboBoostKPA() float32 {
	return utils.BarToKPA(t.TurboBoostBar())
}

func (t *transformer) TyreDiameterFeet() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToFeet(set.FrontLeft),
		FrontRight: utils.MetersToFeet(set.FrontRight),
		RearLeft:   utils.MetersToFeet(set.RearLeft),
		RearRight:  utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) TyreDiameterInches() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToInches(set.FrontLeft),
		FrontRight: utils.MetersToInches(set.FrontRight),
		RearLeft:   utils.MetersToInches(set.RearLeft),
		RearRight:  utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) TyreDiameterMillimeters() models.CornerSet {
	set := t.TyreDiameterMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToMillimeters(set.FrontLeft),
		FrontRight: utils.MetersToMillimeters(set.FrontRight),
		RearLeft:   utils.MetersToMillimeters(set.RearLeft),
		RearRight:  utils.MetersToMillimeters(set.RearRight),
	}
}

func (t *transformer) TyreRadiusFeet() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToFeet(set.FrontLeft),
		FrontRight: utils.MetersToFeet(set.FrontRight),
		RearLeft:   utils.MetersToFeet(set.RearLeft),
		RearRight:  utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) TyreRadiusInches() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToInches(set.FrontLeft),
		FrontRight: utils.MetersToInches(set.FrontRight),
		RearLeft:   utils.MetersToInches(set.RearLeft),
		RearRight:  utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) TyreRadiusMillimeters() models.CornerSet {
	set := t.TyreRadiusMeters()

	return models.CornerSet{
		FrontLeft:  utils.MetersToMillimeters(set.FrontLeft),
		FrontRight: utils.MetersToMillimeters(set.FrontRight),
		RearLeft:   utils.MetersToMillimeters(set.RearLeft),
		RearRight:  utils.MetersToMillimeters(set.RearRight),
	}
}

func (t *transformer) TyreTemperatureFahrenheit() models.CornerSet {
	set := t.TyreTemperatureCelsius()

	return models.CornerSet{
		FrontLeft:  utils.MetersPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: utils.MetersPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   utils.MetersPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  utils.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedKPH() models.CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return models.CornerSet{
		FrontLeft:  utils.MetersPerSecondToKilometersPerHour(set.FrontLeft),
		FrontRight: utils.MetersPerSecondToKilometersPerHour(set.FrontRight),
		RearLeft:   utils.MetersPerSecondToKilometersPerHour(set.RearLeft),
		RearRight:  utils.MetersPerSecondToKilometersPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedMPH() models.CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return models.CornerSet{
		FrontLeft:  utils.MetersPerSecondToMilesPerHour(set.FrontLeft),
		FrontRight: utils.MetersPerSecondToMilesPerHour(set.FrontRight),
		RearLeft:   utils.MetersPerSecondToMilesPerHour(set.RearLeft),
		RearRight:  utils.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedRPM() models.CornerSet {
	rps := t.WheelSpeedRadiansPerSecond()

	return models.CornerSet{
		FrontLeft:  utils.RadiansPerSecondToRevolutionsPerMinute(rps.FrontLeft),
		FrontRight: utils.RadiansPerSecondToRevolutionsPerMinute(rps.FrontRight),
		RearLeft:   utils.RadiansPerSecondToRevolutionsPerMinute(rps.RearLeft),
		RearRight:  utils.RadiansPerSecondToRevolutionsPerMinute(rps.RearRight),
	}
}

func (t *transformer) WaterTemperatureFahrenheit() float32 {
	return utils.CelsiusToFahrenheit(t.RawTelemetry.WaterTemperature)
}
