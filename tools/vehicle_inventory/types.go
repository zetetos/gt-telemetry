package main

import "errors"

// Errors used throughout the application.
var (
	ErrUnsupportedFormat          = errors.New("unsupported format")
	ErrCarIDRequired              = errors.New("CarID is required")
	ErrCarIDAlreadyExists         = errors.New("a vehicle with this CarID already exists")
	ErrVehicleNotFound            = errors.New("vehicle not found in inventory")
	ErrMainJSBundleNotFound       = errors.New("could not find main JS bundle in HTML")
	ErrCarsJSNotFound             = errors.New("could not find cars JS file in main bundle")
	ErrTunersJSNotFound           = errors.New("could not find tuners JS file in main bundle")
	ErrVariableNameNotFound       = errors.New("could not find variable name in JavaScript")
	ErrTunersVariableNameNotFound = errors.New("could not find variable name in tuners JavaScript")
	ErrCarsObjectNotFound         = errors.New("cars object not found in JavaScript")
	ErrTunersObjectNotFound       = errors.New("tuners object not found in JavaScript")
)

const pdNullValue = "---"

// PDVehicle represents the structure of a vehicle entry in the PD inventory JSON.
type PDVehicle struct {
	ID              string `json:"id"`              //nolint:tagliatelle // third party JSON schema
	NameShort       string `json:"nameShort"`       //nolint:tagliatelle // third party JSON schema
	Manufacturer    string `json:"manufacturer"`    //nolint:tagliatelle // third party JSON schema
	Year            int    `json:"year"`            //nolint:tagliatelle // third party JSON schema
	DriveTrain      string `json:"driveTrain"`      //nolint:tagliatelle // third party JSON schema
	AspirationShort string `json:"aspirationShort"` //nolint:tagliatelle // third party JSON schema
	CarClass        string `json:"carClass"`        //nolint:tagliatelle // third party JSON schema
	LengthV         int    `json:"length_v"`        //nolint:tagliatelle // third party JSON schema
	WidthV          int    `json:"width_v"`         //nolint:tagliatelle // third party JSON schema
	HeightV         int    `json:"height_v"`        //nolint:tagliatelle // third party JSON schema
}

// GTCar represents a car entry from the Gran Turismo website cars.js file.
type GTCar struct {
	ID              string `json:"id"`              //nolint:tagliatelle // third party JSON schema
	NameShort       string `json:"nameShort"`       //nolint:tagliatelle // third party JSON schema
	NameLong        string `json:"nameLong"`        //nolint:tagliatelle // third party JSON schema
	ManufacturerID  string `json:"manufacturerId"`  //nolint:tagliatelle // third party JSON schema
	CarClass        string `json:"carClass"`        //nolint:tagliatelle // third party JSON schema
	DriveTrain      string `json:"driveTrain"`      //nolint:tagliatelle // third party JSON schema
	AspirationShort string `json:"aspirationShort"` //nolint:tagliatelle // third party JSON schema
	LengthV         int    `json:"length_v"`        //nolint:tagliatelle // third party JSON schema
	WidthV          int    `json:"width_v"`         //nolint:tagliatelle // third party JSON schema
	HeightV         int    `json:"height_v"`        //nolint:tagliatelle // third party JSON schema
}

// GTTuner represents a manufacturer/tuner entry from the Gran Turismo website tuners.js file.
type GTTuner struct {
	ID        string `json:"id"`        //nolint:tagliatelle // third party JSON schema
	Name      string `json:"name"`      //nolint:tagliatelle // third party JSON schema
	NameShort string `json:"nameShort"` //nolint:tagliatelle // third party JSON schema
}

// changeRecord tracks changes made to a vehicle during merge operations.
type changeRecord struct {
	carID   int
	changes []string
	isNew   bool
}
