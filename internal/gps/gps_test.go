package gps_test

import (
	"testing"

	"github.com/tristanj/dusty/internal/gps"
)

func TestStubGPSNoFix(t *testing.T) {
	g := gps.NewStub()
	_, _, hasFix := g.Location()
	if hasFix {
		t.Error("unconfigured stub should report no fix")
	}
}

func TestStubGPSWithLocation(t *testing.T) {
	g := gps.NewStubWithLocation(-31.416668, 19.233334)
	lat, lon, hasFix := g.Location()
	if !hasFix {
		t.Error("configured stub should report a fix")
	}
	if lat != -31.416668 {
		t.Errorf("lat: want -31.416668, got %f", lat)
	}
	if lon != 19.233334 {
		t.Errorf("lon: want 19.233334, got %f", lon)
	}
}

func TestStubGPSImplementsInterface(t *testing.T) {
	var _ gps.GPSProvider = gps.NewStub()
}
