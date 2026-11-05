//go:build cgo

package kinematics

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo windows LDFLAGS: -L${SRCDIR}/../../../../crates/traumateam_rfm/target/release -ltraumateam_rfm.dll
#cgo linux LDFLAGS: -L${SRCDIR}/../../../../crates/traumateam_rfm/target/release -ltraumateam_rfm -lm -ldl -lpthread
#cgo darwin LDFLAGS: -L${SRCDIR}/../../../../crates/traumateam_rfm/target/release -ltraumateam_rfm -lm -ldl -lpthread
#include "traumateam_rfm.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// ABIVersion returns the ABI version reported by the linked Rust `traumateam_rfm` cdylib.
func ABIVersion() (uint32, error) {
	v := C.traumateam_ffi_abi_version()
	return uint32(v), nil
}

// ComputeAtmosphere invokes `traumateam_compute_atmosphere_ffi` via cgo to calculate
// exact ICAO barometric pressure altitude (ft) and thermodynamic density altitude (ft).
func ComputeAtmosphere(elevationFt, altimeterInHg, oatCelsius float64) (paFt float64, daFt float64, err error) {
	var cPA C.double
	var cDA C.double
	rc := C.traumateam_compute_atmosphere_ffi(
		C.double(elevationFt),
		C.double(altimeterInHg),
		C.double(oatCelsius),
		&cPA,
		&cDA,
	)
	if rc != 0 {
		return 0, 0, fmt.Errorf("traumateam_compute_atmosphere_ffi failed with status code %d", int(rc))
	}
	return float64(cPA), float64(cDA), nil
}

// EvaluateMassBalance invokes `traumateam_eval_mass_balance_ffi` via cgo to evaluate
// total gross weight, longitudinal/lateral CG, and certified RFM envelope compliance.
func EvaluateMassBalance(
	airframe AirframeID,
	fuelWeightKg float64,
	payloads []PayloadStationInput,
) (*MassBalanceEvaluation, error) {
	cPayloads := marshalCPayloads(payloads)
	var payloadsPtr *C.FfiPayloadItem
	if len(cPayloads) > 0 {
		payloadsPtr = (*C.FfiPayloadItem)(unsafe.Pointer(&cPayloads[0]))
	}

	var cOut C.FfiMassBalanceResult
	rc := C.traumateam_eval_mass_balance_ffi(
		C.uint32_t(airframe),
		C.double(fuelWeightKg),
		payloadsPtr,
		C.size_t(len(cPayloads)),
		&cOut,
	)
	if rc != 0 {
		return nil, fmt.Errorf("traumateam_eval_mass_balance_ffi failed with status code %d", int(rc))
	}

	raw := *(*cFfiMassBalanceResult)(unsafe.Pointer(&cOut))
	eval := parseMassBalanceResult(raw)
	return &eval, nil
}

// EvaluateMissionKinematics marshals Go weather data (`models.METAR`) and mission payload stations
// into `C.FfiWeatherMissionInput` and `C.FfiPayloadItem`, invokes `traumateam_eval_mission_kinematics_ffi`
// natively in the Rust physics core, and parses the returned `FfiPerformanceKinematicResult`.
func EvaluateMissionKinematics(req MissionKinematicsRequest) (*MissionKinematicsEvaluation, error) {
	altInHg := 29.92
	oatC := 15.0
	windDir := 0.0
	windSpd := 0.0

	if req.Weather != nil {
		if req.Weather.AltimeterInHg > 10.0 {
			altInHg = req.Weather.AltimeterInHg
		}
		oatC = req.Weather.TemperatureC
		windDir = req.Weather.WindDirDeg
		windSpd = req.Weather.WindSpeedKts
	}

	cInput := C.FfiWeatherMissionInput{
		airframe_id:          C.uint32_t(req.Airframe),
		_padding:             0,
		station_elevation_ft: C.double(req.StationElevationFt),
		altimeter_inhg:       C.double(altInHg),
		oat_celsius:          C.double(oatC),
		wind_dir_deg:         C.double(windDir),
		wind_speed_kts:       C.double(windSpd),
		true_course_deg:      C.double(req.TrueCourseDeg),
		leg_distance_nm:      C.double(req.LegDistanceNM),
		fuel_weight_kg:       C.double(req.FuelWeightKg),
	}

	cPayloads := marshalCPayloads(req.Payloads)
	var payloadsPtr *C.FfiPayloadItem
	if len(cPayloads) > 0 {
		payloadsPtr = (*C.FfiPayloadItem)(unsafe.Pointer(&cPayloads[0]))
	}

	var cOut C.FfiPerformanceKinematicResult
	rc := C.traumateam_eval_mission_kinematics_ffi(
		&cInput,
		payloadsPtr,
		C.size_t(len(cPayloads)),
		&cOut,
	)
	if rc != 0 {
		return nil, fmt.Errorf("traumateam_eval_mission_kinematics_ffi failed with status code %d", int(rc))
	}

	raw := *(*cFfiPerformanceKinematicResult)(unsafe.Pointer(&cOut))
	return parseKinematicsResult(req.Airframe, raw), nil
}
