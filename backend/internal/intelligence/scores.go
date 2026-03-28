package intelligence

import (
	"math"
	"time"
)

// SafetyScore holds the total and per-component safety scores (0–100).
type SafetyScore struct {
	Total    float64
	GPS      float64 // GPS accuracy quality
	Silence  float64 // time-since-last-update decay
	CheckIn  float64 // check-in compliance
	Geofence float64 // geofence compliance
	Attest   float64 // signal attestation recency
}

// motionDecay returns the silence decay constant in seconds for a given motion class.
func motionDecay(motionClass string) float64 {
	switch motionClass {
	case "still", "stationary":
		return 300
	case "walk", "walking":
		return 180
	case "vehicle", "driving", "transit":
		return 600
	default:
		return 300
	}
}

// ComputeSafetyScore is a pure function — no locks, no DB, no side effects.
// All inputs are value copies; callers must snapshot ActiveUser fields before calling.
//
// Parameters:
//   - accuracy:         GPS accuracy in metres (nil = unknown)
//   - lastUpdateMs:     unix milliseconds of last position update (0 = never)
//   - lastAttestAtMs:   unix milliseconds of last attestation (0 = never)
//   - checkInEnabled:   whether check-in feature is active
//   - checkInOverdueAt: unix ms at which check-in becomes overdue (0 = not applicable)
//   - geofenceEnabled:  whether geofence is active
//   - geofenceBreached: whether the geofence was breached on last position check
//   - motionClass:      "still" | "walk" | "vehicle" (or empty)
func ComputeSafetyScore(
	accuracy *float64,
	lastUpdateMs int64,
	lastAttestAtMs int64,
	checkInEnabled bool,
	checkInOverdueAt int64,
	geofenceEnabled bool,
	geofenceBreached bool,
	motionClass string,
) SafetyScore {
	now := time.Now().UnixMilli()

	// ── GPS accuracy score ────────────────────────────────────────────────────
	var gpsScore float64
	if accuracy == nil {
		gpsScore = 50
	} else {
		acc := *accuracy
		switch {
		case acc <= 20:
			gpsScore = 100
		case acc >= 200:
			gpsScore = 0
		default:
			// linear interpolation between 20 m (100) and 200 m (0)
			gpsScore = 100 * (1 - (acc-20)/(200-20))
		}
	}

	// ── Silence score (freshness decay) ──────────────────────────────────────
	var silenceScore float64
	if lastUpdateMs <= 0 {
		silenceScore = 0
	} else {
		elapsedSec := float64(now-lastUpdateMs) / 1000.0
		decay := motionDecay(motionClass)
		silenceScore = 100 * math.Exp(-elapsedSec/decay)
	}

	// ── Check-in score ────────────────────────────────────────────────────────
	var checkInScore float64 = 100
	if checkInEnabled && checkInOverdueAt > 0 && now >= checkInOverdueAt {
		checkInScore = 0
	}

	// ── Geofence score ────────────────────────────────────────────────────────
	var geofenceScore float64 = 100
	if geofenceEnabled && geofenceBreached {
		geofenceScore = 0
	}

	// ── Attestation score ─────────────────────────────────────────────────────
	var attestScore float64
	if lastAttestAtMs <= 0 {
		attestScore = 50 // never attested
	} else {
		elapsedHours := float64(now-lastAttestAtMs) / float64(time.Hour.Milliseconds())
		attestScore = 100 * math.Exp(-elapsedHours/12.0)
	}

	// ── Weighted total ────────────────────────────────────────────────────────
	total := 0.30*silenceScore +
		0.25*gpsScore +
		0.20*checkInScore +
		0.15*geofenceScore +
		0.10*attestScore

	return SafetyScore{
		Total:    total,
		GPS:      gpsScore,
		Silence:  silenceScore,
		CheckIn:  checkInScore,
		Geofence: geofenceScore,
		Attest:   attestScore,
	}
}
