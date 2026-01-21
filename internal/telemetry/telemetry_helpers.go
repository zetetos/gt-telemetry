package telemetry

// SetFormatStandard sets the telemetry format to Standard (format "A") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (this *GranTurismoTelemetry) SetFormatStandard() { //nolint:revive,staticcheck // meatches kaitai generated code style
	this._f_standardFormat = true
	this.standardFormat = true
	this._f_addendum1Format = true
	this.addendum1Format = false
	this._f_addendum2Format = true
	this.addendum2Format = false
}

// SetFormatAddendum1 sets the telemetry format to Addendum1 (format "B") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (this *GranTurismoTelemetry) SetFormatAddendum1() { //nolint:revive,staticcheck // meatches kaitai generated code style
	this._f_standardFormat = true
	this.standardFormat = false
	this._f_addendum1Format = true
	this.addendum1Format = true
	this._f_addendum2Format = true
	this.addendum2Format = false
}

// SetFormatAddendum2 sets the telemetry format to Addendum2 (format "~") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (this *GranTurismoTelemetry) SetFormatAddendum2() { //nolint:revive,staticcheck // meatches kaitai generated code style
	this._f_standardFormat = true
	this.standardFormat = false
	this._f_addendum1Format = true
	this.addendum1Format = true
	this._f_addendum2Format = true
	this.addendum2Format = true
}

// SetAngularVelocityVector sets the angular velocity vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetAngularVelocityVector(x, y, z float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.AngularVelocityVector = &GranTurismoTelemetry_Vector{
		VectorX: x,
		VectorY: y,
		VectorZ: z,
	}
}

// SetVelocityVector sets the velocity vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetVelocityVector(x, y, z float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.VelocityVector = &GranTurismoTelemetry_Vector{
		VectorX: x,
		VectorY: y,
		VectorZ: z,
	}
}

// SetTranslationalEnvelope sets the translational envelope for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetTranslationalEnvelope(sway, heave, surge float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.TranslationalEnvelope = &GranTurismoTelemetry_TranslationalEnvelope{
		Sway:  sway,
		Heave: heave,
		Surge: surge,
	}
}

// SetTransmissionGear sets the transmission gear for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetTransmissionGear(current, suggested uint64) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.TransmissionGear = &GranTurismoTelemetry_TransmissionGear{
		Current:   current,
		Suggested: suggested,
	}
}

// SetHeader sets the header magic value for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetHeader(magic uint32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.Header = &GranTurismoTelemetry_Header{
		Magic: magic,
	}
}

// SetMapPositionCoordinates sets the map position coordinates for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetMapPositionCoordinates(x, y, z float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.MapPositionCoordinates = &GranTurismoTelemetry_Coordinate{
		CoordinateX: x,
		CoordinateY: y,
		CoordinateZ: z,
	}
}

// SetRotationalEnvelope sets the rotational envelope for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetRotationalEnvelope(pitch, yaw, roll float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.RotationalEnvelope = &GranTurismoTelemetry_RotationalEnvelope{
		Pitch: pitch,
		Yaw:   yaw,
		Roll:  roll,
	}
}

// SetTyreTemperature sets the tyre temperature corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetTyreTemperature(fl, fr, rl, rr float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.TyreTemperature = &GranTurismoTelemetry_CornerSet{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
	}
}

// SetFlags sets the telemetry flags for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
//
//nolint:revive,staticcheck // matches kaitai generated code style
func (this *GranTurismoTelemetry) SetFlags(
	live, gamePaused, loading, inGear, hasTurbo, revLimiterAlert,
	handBrakeActive, headlightsActive, highBeamActive, lowBeamActive, asmActive, tcsActive bool,
) {
	this.Flags = &GranTurismoTelemetry_Flags{
		Live:             live,
		GamePaused:       gamePaused,
		Loading:          loading,
		InGear:           inGear,
		HasTurbo:         hasTurbo,
		RevLimiterAlert:  revLimiterAlert,
		HandBrakeActive:  handBrakeActive,
		HeadlightsActive: headlightsActive,
		HighBeamActive:   highBeamActive,
		LowBeamActive:    lowBeamActive,
		AsmActive:        asmActive,
		TcsActive:        tcsActive,
	}
}

// SetRoadPlaneVector sets the road plane vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetRoadPlaneVector(x, y, z float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.RoadPlaneVector = &GranTurismoTelemetry_Vector{
		VectorX: x,
		VectorY: y,
		VectorZ: z,
	}
}

// SetWheelRadiansPerSecond sets the wheel radians per second corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetWheelRadiansPerSecond(fl, fr, rl, rr float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.WheelRadiansPerSecond = &GranTurismoTelemetry_CornerSet{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
	}
}

// SetTyreRadius sets the tyre radius corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetTyreRadius(fl, fr, rl, rr float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.TyreRadius = &GranTurismoTelemetry_CornerSet{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
	}
}

// SetSuspensionHeight sets the suspension height corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (this *GranTurismoTelemetry) SetSuspensionHeight(fl, fr, rl, rr float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.SuspensionHeight = &GranTurismoTelemetry_CornerSet{
		FrontLeft:  fl,
		FrontRight: fr,
		RearLeft:   rl,
		RearRight:  rr,
	}
}

// SetTransmissionGearRatio sets the transmission gear ratios for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
// The gears slice should contain up to 8 gear ratios.
func (this *GranTurismoTelemetry) SetTransmissionGearRatio(gears []float32) { //nolint:revive,staticcheck // matches kaitai generated code style
	this.TransmissionGearRatio = &GranTurismoTelemetry_GearRatio{
		Gear: gears,
	}
}
