package circuits_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zetetos/gt-telemetry/v2/pkg/circuits"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"
)

type CircuitsTestSuite struct {
	suite.Suite
}

func TestCircuitsTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CircuitsTestSuite))
}

func (suite *CircuitsTestSuite) TestEmptyJSONParameterFallsBackToBaseInventory() {
	// Arrange
	var inventoryJSON []byte

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	// Assert - should be able to get a circuit from the base inventory
	got := db.GetAllCircuitIDs()
	suite.NotEmpty(got, "Should have circuits from base inventory")
}

func (suite *CircuitsTestSuite) TestValidJSONParameterCanConstructInventory() {
	// Arrange
	inventoryJSON := []byte(`{
		"circuits": {
			"test_circuit": {
				"id": "test_circuit",
				"name": "Test Circuit",
				"region": "test",
				"length": 1000,
				"startline": { "x": 100, "y": 10, "z": 200 }
			}
		},
		"coordinates": {}
	}`)

	// Act
	_, err := circuits.NewDB(inventoryJSON)

	// Assert
	suite.Require().NoError(err)
}

func (suite *CircuitsTestSuite) TestInvalidJSONParameterReturnsError() {
	// Arrange
	inventoryJSON := []byte(`{
		not_valid_json: }
	}`)

	// Act
	_, err := circuits.NewDB(inventoryJSON)

	// Assert
	suite.ErrorContains(err, "unmarshall circuit inventory JSON")
}

func (suite *CircuitsTestSuite) TestGetCircuitByIDWithInvalidIDReturnsNotFound() {
	// Arrange
	inventoryJSON := []byte(`{
		"circuits": {},
		"coordinates": {}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	_, found := db.GetCircuitByID("nonexistent_circuit")

	// Assert
	suite.False(found, "Should not find non-existent circuit")
}

func (suite *CircuitsTestSuite) TestGetCircuitByIDWithValidIDReturnsCircuit() {
	// Arrange
	want := circuits.CircuitInfo{
		ID:        "test_circuit",
		Name:      "Test Circuit",
		Length:    1500,
		StartLine: models.CoordinateNorm{X: 100, Y: 20, Z: 300},
	}

	inventoryJSON := []byte(`{
		"circuits": {
			"test_circuit": {
				"id": "test_circuit",
				"name": "Test Circuit", 
				"region": "test",
				"length": 1500,
				"startline": { "x": 100, "y": 20, "z": 300 }
			}
		},
		"coordinates": {}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	got, found := db.GetCircuitByID("test_circuit")

	// Assert
	suite.True(found, "Should find the test circuit")
	suite.Equal(want, got)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtCoordinateWithValidCoordinateReturnsCircuit() {
	// Arrange
	want := "circuit1"
	inventoryJSON := []byte(`{
		"circuits": {},
		"coordinates": {
			"x:80,y:8,z:64": "circuit1"
		}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	// Test coordinate that should normalize to x:80,y:8,z:64
	got, found := db.GetCircuitAtCoordinate(models.Coordinate{X: 80, Y: 8, Z: 70}, models.CoordinateTypeCircuit)

	// Assert
	suite.True(found, "Should find circuit at coordinate")
	suite.Equal(want, got)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtCoordinateWithInvalidCoordinateReturnsNotFound() {
	// Arrange
	inventoryJSON := []byte(`{
		"circuits": {},
		"coordinates": {}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	got, found := db.GetCircuitAtCoordinate(models.Coordinate{X: 100, Y: 100, Z: 100}, models.CoordinateTypeCircuit)

	// Assert
	suite.False(found, "Should not find circuit at non-existent coordinate")
	suite.Empty(got)
}

func (suite *CircuitsTestSuite) TestGetCircuitAtStartLineWithValidCoordinateReturnsCircuit() {
	// Arrange
	want := "test_circuit"
	coordinate := models.Coordinate{X: 40, Y: 4, Z: 40}
	inventoryJSON := []byte(`{
		"circuits": {
			"test_circuit": {
				"id": "test_circuit",
				"name": "Test Circuit",
				"region": "test", 
				"length": 1000,
				"startline": { "x": 32, "y": 4, "z": 32 }
			}
		},
		"coordinates": {}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	got, found := db.GetCircuitAtCoordinate(coordinate, models.CoordinateTypeStartLine)

	// Assert
	suite.True(found, "Should find circuit at start line")
	suite.Equal(want, got)
}

func (suite *CircuitsTestSuite) TestGetAllCircuitIDsReturnsAllIDs() {
	// Arrange
	want := []string{"circuit1", "circuit2"}
	inventoryJSON := []byte(`{
		"circuits": {
			"circuit1": {
				"id": "circuit1",
				"name": "Circuit 1",
				"region": "test",
				"length": 1000,
				"startline": { "x": 0, "y": 0, "z": 0 }
			},
			"circuit2": {
				"id": "circuit2", 
				"name": "Circuit 2",
				"region": "test",
				"length": 2000,
				"startline": { "x": 100, "y": 10, "z": 200 }
			}
		},
		"coordinates": {}
	}`)

	// Act
	db, err := circuits.NewDB(inventoryJSON)
	suite.Require().NoError(err)

	got := db.GetAllCircuitIDs()

	// Assert
	suite.Len(got, len(want))
	suite.ElementsMatch(want, got)
}

func (suite *CircuitsTestSuite) TestNormaliseStartLineCoordinate() { //nolint:dupl // Intentional similarity
	// Arrange
	tests := []struct {
		name  string
		input models.Coordinate
		want  models.CoordinateNorm
	}{
		{
			name:  "coordinates divisible by normalisation factors",
			input: models.Coordinate{X: 64, Y: 8, Z: 96},
			want:  models.CoordinateNorm{X: 64, Y: 8, Z: 96},
		},
		{
			name:  "coordinates not divisible by normalisation factors",
			input: models.Coordinate{X: 50, Y: 6, Z: 70},
			want:  models.CoordinateNorm{X: 48, Y: 6, Z: 64},
		},
		{
			name:  "negative coordinates",
			input: models.Coordinate{X: -50, Y: -6, Z: -70},
			want:  models.CoordinateNorm{X: -48, Y: -6, Z: -64},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// Act
			got := circuits.NormaliseStartLineCoordinate(test.input)

			// Assert
			suite.Equal(test.want, got)
		})
	}
}

func (suite *CircuitsTestSuite) TestNormaliseCircuitCoordinate() { //nolint:dupl // Intentional similarity
	// Arrange
	tests := []struct {
		name  string
		input models.Coordinate
		want  models.CoordinateNorm
	}{
		{
			name:  "coordinates divisible by normalisation factors",
			input: models.Coordinate{X: 128, Y: 16, Z: 192},
			want:  models.CoordinateNorm{X: 128, Y: 16, Z: 192},
		},
		{
			name:  "coordinates not divisible by normalisation factors",
			input: models.Coordinate{X: 100, Y: 12, Z: 150},
			want:  models.CoordinateNorm{X: 96, Y: 12, Z: 144},
		},
		{
			name:  "negative coordinates",
			input: models.Coordinate{X: -100, Y: -12, Z: -150},
			want:  models.CoordinateNorm{X: -96, Y: -12, Z: -144},
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// Act
			got := circuits.NormaliseCircuitCoordinate(test.input)

			// Assert
			suite.Equal(test.want, got)
		})
	}
}

func (suite *CircuitsTestSuite) TestCoordinateNormToString() {
	// Arrange
	want := "x:100,y:200,z:300"
	coordinate := models.CoordinateNorm{X: 100, Y: 200, Z: 300}

	// Act
	got := coordinate.String()

	// Assert
	suite.Equal(want, got)
}
