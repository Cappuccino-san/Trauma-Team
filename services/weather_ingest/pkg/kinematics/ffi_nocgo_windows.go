//go:build !cgo && windows

package kinematics

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	dllOnce sync.Once
	dllErr  error
	rfmDLL  *syscall.LazyDLL

	procABIVersion             *syscall.LazyProc
	procComputeAtmosphere      *syscall.LazyProc
	procEvalMassBalance        *syscall.LazyProc
	procEvalMissionKinematics  *syscall.LazyProc
)

func initRustCDylib() error {
	dllOnce.Do(func() {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			dllErr = fmt.Errorf("unable to resolve runtime caller path for traumateam_rfm.dll")
			return
		}
		// Resolve crates/traumateam_rfm/target/release/traumateam_rfm.dll relative to pkg/kinematics/
		dllPath := filepath.Clean(filepath.Join(
			filepath.Dir(currentFile),
			"..", "..", "..", "..",
			"crates", "traumateam_rfm", "target", "release", "traumateam_rfm.dll",
		))

		rfmDLL = syscall.NewLazyDLL(dllPath)
		if err := rfmDLL.Load(); err != nil {
			dllErr = fmt.Errorf("failed to load Rust cdylib at %s: %w", dllPath, err)
			return
		}

		procABIVersion = rfmDLL.NewProc("traumateam_ffi_abi_version")
		procComputeAtmosphere = rfmDLL.NewProc("traumateam_compute_atmosphere_ffi")
		procEvalMassBalance = rfmDLL.NewProc("traumateam_eval_mass_balance_ffi")
		procEvalMissionKinematics = rfmDLL.NewProc("traumateam_eval_mission_kinematics_ffi")
	})
	return dllErr
}

// ABIVersion invokes `traumateam_ffi_abi_version` in the compiled Rust `traumateam_rfm.dll`.
func ABIVersion() (uint32, error) {
	if err := initRustCDylib(); err != nil {
		return 0, err
	}
	r1, _, _ := procABIVersion.Call()
	return uint32(r1), nil
}

// ComputeAtmosphere invokes `traumateam_compute_atmosphere_ffi` in the Rust cdylib.
// Note: On Windows x64 ABI, floating-point values in registers (XMM0-XMM2) cannot be passed
// directly through `syscall.SyscallN` (which only populates integer registers RCX, RDX, R8, R9).
// Therefore, for direct struct-pointer functions (`EvaluateMassBalance` and `EvaluateMissionKinematics`),
// we pass pointers in integer registers directly to the Rust DLL, and for `ComputeAtmosphere` we
// invoke the Rust DLL via a pointer-based mission kinematics call to extract exact PA and DA.
func ComputeAtmosphere(elevationFt, altimeterInHg, oatCelsius float64) (paFt float64, daFt float64, err error) {
	if math.IsNaN(elevationFt) || math.IsNaN(altimeterInHg) || math.IsNaN(oatCelsius) {
		return 0, 0, fmt.Errorf("non-finite atmospheric input")
	}
	if err := initRustCDylib(); err != nil {
		return 0, 0, err
	}

	cInput := cFfiWeatherMissionInput{
		AirframeID:         uint32(AirframeAirbusH145),
		StationElevationFt: elevationFt,
		AltimeterInHg:      altimeterInHg,
		OATCelsius:         oatCelsius,
		WindDirDeg:         0,
		WindSpeedKts:       0,
		TrueCourseDeg:      0,
		LegDistanceNM:      0,
		FuelWeightKg:       300.0,
	}
	var cOut cFfiPerformanceKinematicResult
	r1, _, _ := procEvalMissionKinematics.Call(
		uintptr(unsafe.Pointer(&cInput)),
		0,
		0,
		uintptr(unsafe.Pointer(&cOut)),
	)
	rc := int32(r1)
	if rc != 0 {
		return 0, 0, fmt.Errorf("traumateam_compute_atmosphere_ffi failed with status code %d", rc)
	}
	return cOut.PressureAltitudeFt, cOut.DensityAltitudeFt, nil
}

// EvaluateMassBalance invokes `traumateam_eval_mass_balance_ffi` via the Rust cdylib.
func EvaluateMassBalance(
	airframe AirframeID,
	fuelWeightKg float64,
	payloads []PayloadStationInput,
) (*MassBalanceEvaluation, error) {
	if err := initRustCDylib(); err != nil {
		return nil, err
	}
	if math.IsNaN(fuelWeightKg) || fuelWeightKg < 0 {
		return nil, fmt.Errorf("invalid fuel weight: %v", fuelWeightKg)
	}

	// Because `traumateam_eval_mission_kinematics_ffi` takes all parameters by pointer
	// (`*const FfiWeatherMissionInput`, `*const FfiPayloadItem`, `usize`, `*mut FfiPerformanceKinematicResult`),
	// all 4 arguments are integer/pointer registers (RCX, RDX, R8, R9) in the Windows x64 C ABI!
	cInput := cFfiWeatherMissionInput{
		AirframeID:         uint32(airframe),
		StationElevationFt: 0,
		AltimeterInHg:      29.92,
		OATCelsius:         15.0,
		FuelWeightKg:       fuelWeightKg,
	}
	cPayloads := marshalCPayloads(payloads)
	var payloadsPtr uintptr
	if len(cPayloads) > 0 {
		payloadsPtr = uintptr(unsafe.Pointer(&cPayloads[0]))
	}

	var cOut cFfiPerformanceKinematicResult
	r1, _, _ := procEvalMissionKinematics.Call(
		uintptr(unsafe.Pointer(&cInput)),
		payloadsPtr,
		uintptr(len(cPayloads)),
		uintptr(unsafe.Pointer(&cOut)),
	)
	rc := int32(r1)
	if rc != 0 {
		return nil, fmt.Errorf("traumateam_eval_mass_balance_ffi failed with status code %d", rc)
	}

	eval := parseMassBalanceResult(cOut.MassBalance)
	return &eval, nil
}

// EvaluateMissionKinematics marshals Go weather data (`models.METAR`) and payload stations
// into `FfiWeatherMissionInput` and `FfiPayloadItem`, invokes `traumateam_eval_mission_kinematics_ffi`
// natively in the Rust cdylib (`traumateam_rfm.dll`), and parses the returned structs.
func EvaluateMissionKinematics(req MissionKinematicsRequest) (*MissionKinematicsEvaluation, error) {
	if err := initRustCDylib(); err != nil {
		return nil, err
	}

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

	cInput := cFfiWeatherMissionInput{
		AirframeID:         uint32(req.Airframe),
		StationElevationFt: req.StationElevationFt,
		AltimeterInHg:      altInHg,
		OATCelsius:         oatC,
		WindDirDeg:         windDir,
		WindSpeedKts:       windSpd,
		TrueCourseDeg:      req.TrueCourseDeg,
		LegDistanceNM:      req.LegDistanceNM,
		FuelWeightKg:       req.FuelWeightKg,
	}

	cPayloads := marshalCPayloads(req.Payloads)
	var payloadsPtr uintptr
	if len(cPayloads) > 0 {
		payloadsPtr = uintptr(unsafe.Pointer(&cPayloads[0]))
	}

	var cOut cFfiPerformanceKinematicResult
	r1, _, _ := procEvalMissionKinematics.Call(
		uintptr(unsafe.Pointer(&cInput)),
		payloadsPtr,
		uintptr(len(cPayloads)),
		uintptr(unsafe.Pointer(&cOut)),
	)
	rc := int32(r1)
	if rc != 0 {
		return nil, fmt.Errorf("traumateam_eval_mission_kinematics_ffi failed with status code %d", rc)
	}

	return parseKinematicsResult(req.Airframe, cOut), nil
}
