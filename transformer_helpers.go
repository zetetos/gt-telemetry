package gttelemetry

// SetFormatStandard sets the telemetry format to Standard (format "A") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (t *Transformer) SetFormatStandard() {
	t.RawTelemetry.SetFormatStandard()
}

// SetFormatAddendum1 sets the telemetry format to Addendum1 (format "B") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (t *Transformer) SetFormatAddendum1() {
	t.RawTelemetry.SetFormatAddendum1()
}

// SetFormatAddendum2 sets the telemetry format to Addendum2 (format "~") for testing purposes.
// This allows tests to set the format without parsing a binary packet.
func (t *Transformer) SetFormatAddendum2() {
	t.RawTelemetry.SetFormatAddendum2()
}

// SetAngularVelocityVector sets the angular velocity vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetAngularVelocityVector(x, y, z float32) {
	t.RawTelemetry.SetAngularVelocityVector(x, y, z)
}

// SetVelocityVector sets the velocity vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetVelocityVector(x, y, z float32) {
	t.RawTelemetry.SetVelocityVector(x, y, z)
}

// SetTranslationalEnvelope sets the translational envelope for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetTranslationalEnvelope(sway, heave, surge float32) {
	t.RawTelemetry.SetTranslationalEnvelope(sway, heave, surge)
}

// SetTransmissionGear sets the transmission gear for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetTransmissionGear(current, suggested uint64) {
	t.RawTelemetry.SetTransmissionGear(current, suggested)
}

// SetHeader sets the header magic value for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetHeader(magic uint32) {
	t.RawTelemetry.SetHeader(magic)
}

// SetMapPositionCoordinates sets the map position coordinates for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetMapPositionCoordinates(x, y, z float32) {
	t.RawTelemetry.SetMapPositionCoordinates(x, y, z)
}

// SetRotationalEnvelope sets the rotational envelope for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetRotationalEnvelope(pitch, yaw, roll float32) {
	t.RawTelemetry.SetRotationalEnvelope(pitch, yaw, roll)
}

// SetTyreTemperature sets the tyre temperature corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetTyreTemperature(fl, fr, rl, rr float32) {
	t.RawTelemetry.SetTyreTemperature(fl, fr, rl, rr)
}

// SetFlags sets the telemetry flags for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetFlags(
	live, gamePaused, loading, inGear, hasTurbo, revLimiterAlert,
	handBrakeActive, headlightsActive, highBeamActive, lowBeamActive, asmActive, tcsActive bool,
) {
	t.RawTelemetry.SetFlags(
		live, gamePaused, loading, inGear, hasTurbo, revLimiterAlert,
		handBrakeActive, headlightsActive, highBeamActive, lowBeamActive, asmActive, tcsActive,
	)
}

// SetRoadPlaneVector sets the road plane vector for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetRoadPlaneVector(x, y, z float32) {
	t.RawTelemetry.SetRoadPlaneVector(x, y, z)
}

// SetWheelRadiansPerSecond sets the wheel radians per second corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetWheelRadiansPerSecond(fl, fr, rl, rr float32) {
	t.RawTelemetry.SetWheelRadiansPerSecond(fl, fr, rl, rr)
}

// SetTyreRadius sets the tyre radius corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetTyreRadius(fl, fr, rl, rr float32) {
	t.RawTelemetry.SetTyreRadius(fl, fr, rl, rr)
}

// SetSuspensionHeight sets the suspension height corner set for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
func (t *Transformer) SetSuspensionHeight(fl, fr, rl, rr float32) {
	t.RawTelemetry.SetSuspensionHeight(fl, fr, rl, rr)
}

// SetTransmissionGearRatio sets the transmission gear ratios for testing purposes.
// This allows tests to set telemetry values without parsing a binary packet.
// The gears slice should contain up to 8 gear ratios.
func (t *Transformer) SetTransmissionGearRatio(gears []float32) {
	t.RawTelemetry.SetTransmissionGearRatio(gears)
}
