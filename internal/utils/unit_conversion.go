package utils

import "math"

func BarToPSI(bar float32) float32 {
	return bar * 14.50377
}

func BarToInHg(bar float32) float32 {
	return bar * 29.52998
}

func BarToKPA(bar float32) float32 {
	return bar * 100
}

func CelsiusToFahrenheit(c float32) float32 {
	return c*1.8 + 32
}

func MetersToFeet(m float32) float32 {
	return m * 3.28084
}

func MetersToInches(m float32) float32 {
	return m * 39.3701
}

func MetersToMillimeters(m float32) float32 {
	return m * 1000
}

func MillimetersToInches(mm int) float32 {
	return float32(mm) / 25.4
}

func MetersPerSecondToKilometersPerHour(mps float32) float32 {
	return mps * 3.6
}

func MetersPerSecondToMilesPerHour(mps float32) float32 {
	return mps / 0.44704
}

func RadiansPerSecondToRevolutionsPerMinute(rps float32) float32 {
	return rps * (60 / (2 * math.Pi))
}

func RadiansToDegrees(r float32) float32 {
	return r * (180 / math.Pi)
}
