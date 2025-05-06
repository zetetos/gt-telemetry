package telemetry

import (
	"fmt"

	"github.com/zetetos/gt-telemetry/internal/utils"
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

func (t *transformer) SuspensionHeightFeet() CornerSet {
	set := t.SuspensionHeightMeters()

	return CornerSet{
		utils.MetersToFeet(set.FrontLeft),
		utils.MetersToFeet(set.FrontRight),
		utils.MetersToFeet(set.RearLeft),
		utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) SuspensionHeightInches() CornerSet {
	set := t.SuspensionHeightMeters()

	return CornerSet{
		utils.MetersToInches(set.FrontLeft),
		utils.MetersToInches(set.FrontRight),
		utils.MetersToInches(set.RearLeft),
		utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) SuspensionHeightMillimeters() CornerSet {
	set := t.SuspensionHeightMeters()

	return CornerSet{
		utils.MetersToMillimeters(set.FrontLeft),
		utils.MetersToMillimeters(set.FrontRight),
		utils.MetersToMillimeters(set.RearLeft),
		utils.MetersToMillimeters(set.RearRight),
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

func (t *transformer) TyreDiameterFeet() CornerSet {
	set := t.TyreDiameterMeters()

	return CornerSet{
		utils.MetersToFeet(set.FrontLeft),
		utils.MetersToFeet(set.FrontRight),
		utils.MetersToFeet(set.RearLeft),
		utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) TyreDiameterInches() CornerSet {
	set := t.TyreDiameterMeters()

	return CornerSet{
		utils.MetersToInches(set.FrontLeft),
		utils.MetersToInches(set.FrontRight),
		utils.MetersToInches(set.RearLeft),
		utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) TyreDiameterMillimeters() CornerSet {
	set := t.TyreDiameterMeters()

	return CornerSet{
		utils.MetersToMillimeters(set.FrontLeft),
		utils.MetersToMillimeters(set.FrontRight),
		utils.MetersToMillimeters(set.RearLeft),
		utils.MetersToMillimeters(set.RearRight),
	}
}

func (t *transformer) TyreRadiusFeet() CornerSet {
	set := t.TyreRadiusMeters()

	return CornerSet{
		utils.MetersToFeet(set.FrontLeft),
		utils.MetersToFeet(set.FrontRight),
		utils.MetersToFeet(set.RearLeft),
		utils.MetersToFeet(set.RearRight),
	}
}

func (t *transformer) TyreRadiusInches() CornerSet {
	set := t.TyreRadiusMeters()

	return CornerSet{
		utils.MetersToInches(set.FrontLeft),
		utils.MetersToInches(set.FrontRight),
		utils.MetersToInches(set.RearLeft),
		utils.MetersToInches(set.RearRight),
	}
}

func (t *transformer) TyreRadiusMillimeters() CornerSet {
	set := t.TyreRadiusMeters()

	return CornerSet{
		utils.MetersToMillimeters(set.FrontLeft),
		utils.MetersToMillimeters(set.FrontRight),
		utils.MetersToMillimeters(set.RearLeft),
		utils.MetersToMillimeters(set.RearRight),
	}
}

func (t *transformer) TyreTemperatureFahrenheit() CornerSet {
	set := t.TyreTemperatureCelsius()

	return CornerSet{
		utils.MetersPerSecondToMilesPerHour(set.FrontLeft),
		utils.MetersPerSecondToMilesPerHour(set.FrontRight),
		utils.MetersPerSecondToMilesPerHour(set.RearLeft),
		utils.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedKPH() CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return CornerSet{
		utils.MetersPerSecondToKilometersPerHour(set.FrontLeft),
		utils.MetersPerSecondToKilometersPerHour(set.FrontRight),
		utils.MetersPerSecondToKilometersPerHour(set.RearLeft),
		utils.MetersPerSecondToKilometersPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedMPH() CornerSet {
	set := t.WheelSpeedMetersPerSecond()

	return CornerSet{
		utils.MetersPerSecondToMilesPerHour(set.FrontLeft),
		utils.MetersPerSecondToMilesPerHour(set.FrontRight),
		utils.MetersPerSecondToMilesPerHour(set.RearLeft),
		utils.MetersPerSecondToMilesPerHour(set.RearRight),
	}
}

func (t *transformer) WheelSpeedRPM() CornerSet {
	rps := t.WheelSpeedRadiansPerSecond()

	return CornerSet{
		FrontLeft:  utils.RadiansPerSecondToRevolutionsPerMinute(rps.FrontLeft),
		FrontRight: utils.RadiansPerSecondToRevolutionsPerMinute(rps.FrontRight),
		RearLeft:   utils.RadiansPerSecondToRevolutionsPerMinute(rps.RearLeft),
		RearRight:  utils.RadiansPerSecondToRevolutionsPerMinute(rps.RearRight),
	}
}

func (t *transformer) WaterTemperatureFahrenheit() float32 {
	return utils.CelsiusToFahrenheit(t.RawTelemetry.WaterTemperature)
}
