## ADDED Requirements

### Requirement: GPS service exposes a location interface
The GPS service SHALL expose a `Location()` method returning the current latitude, longitude, and a boolean indicating whether a fix is available. The initial implementation SHALL be a stub returning a configurable fixed location.

#### Scenario: Stub returns configured location
- **WHEN** `Location()` is called on the stub GPS service configured with lat -31.416668, lon 19.233334
- **THEN** it returns `(-31.416668, 19.233334, true)`

#### Scenario: Unconfigured stub returns no fix
- **WHEN** `Location()` is called on the stub GPS service with no configured location
- **THEN** it returns `(0, 0, false)`

### Requirement: GPS interface supports future hardware implementations
The GPS interface SHALL be defined as a Go interface (`GPSProvider`) with `Location() (lat, lon float64, hasFix bool)` so that future implementations (e.g., GPSD, serial NMEA) can be swapped in without changing consumers.

#### Scenario: Interface is implementable
- **WHEN** a new GPS implementation satisfies the `GPSProvider` interface
- **THEN** it can be injected into the mesh subsystem without modifying mesh code
