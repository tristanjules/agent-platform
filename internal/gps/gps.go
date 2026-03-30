// Package gps provides a location interface for DUSTY.
// The initial implementation is a stub returning a configurable fixed location.
// Future implementations can replace the stub with real GPS hardware (e.g. GPSD, serial NMEA).
package gps

// GPSProvider returns the current location.
type GPSProvider interface {
	// Location returns latitude, longitude, and whether a fix is available.
	Location() (lat, lon float64, hasFix bool)
}

// StubGPS is a configurable fixed-location provider with no hardware dependency.
type StubGPS struct {
	lat    float64
	lon    float64
	hasFix bool
}

// NewStub creates a StubGPS with no fix.
func NewStub() *StubGPS {
	return &StubGPS{}
}

// NewStubWithLocation creates a StubGPS with a fixed location and a valid fix.
func NewStubWithLocation(lat, lon float64) *StubGPS {
	return &StubGPS{lat: lat, lon: lon, hasFix: true}
}

// Location returns the configured fixed coordinates.
func (g *StubGPS) Location() (lat, lon float64, hasFix bool) {
	return g.lat, g.lon, g.hasFix
}
