package units_test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/internal/units"
)

type UnitConversionTestSuite struct {
	suite.Suite
}

func TestUnitConversionTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnitConversionTestSuite))
}

func (suite *UnitConversionTestSuite) TestUnitConversionFunctionsReturnCorrectValues() {
	type testCase struct {
		function  func(float32) float32
		withValue float32
		wantValue float32
	}

	// Arrange
	testCases := []testCase{
		{units.BarToPSI, 1, 14.50377},
		{units.BarToInHg, 1, 29.52998},
		{units.BarToKPA, 1, 100},
		{units.CelsiusToFahrenheit, 0, 32},
		{units.CelsiusToFahrenheit, 100, 212},
		{units.MetersToFeet, 1, 3.28084},
		{units.MetersToInches, 1, 39.3701},
		{units.MetersToMillimeters, 1, 1000},
		{units.MetersPerSecondToKilometersPerHour, 1, 3.6},
		{units.MetersPerSecondToMilesPerHour, 1, 2.2369363},
		{units.RadiansPerSecondToRevolutionsPerMinute, 1, 9.549296},
		{units.RadiansToDegrees, 1, 57.29578},
		{units.RadiansToDegrees, -3.14159265, -180},
	}

	for _, testCase := range testCases {
		fnNameSegments := strings.Split(runtime.FuncForPC(reflect.ValueOf(testCase.function).Pointer()).Name(), ".")
		fnName := fnNameSegments[len(fnNameSegments)-1]

		suite.Run(fnName, func() {
			// Act
			gotValue := testCase.function(testCase.withValue)

			// Assert
			suite.InEpsilon(testCase.wantValue, gotValue, 1e-5)
		})
	}
}

func (suite *UnitConversionTestSuite) TestMillimetersToInchesReturnsCorrectValue() {
	// Arrange
	testCases := []struct {
		withValue int
		wantValue float32
	}{
		{1, 0.03937008},
		{25, 0.9842520},
		{100, 3.937008},
		{1000, 39.37008},
		{4500, 177.16535},
		{1800, 70.866142},
		{1300, 51.181103},
		{2700, 106.29921},
	}

	for _, tc := range testCases {
		// Act
		gotValue := units.MillimetersToInches(tc.withValue)

		// Assert
		suite.InEpsilon(tc.wantValue, gotValue, 1e-5)
	}
}
