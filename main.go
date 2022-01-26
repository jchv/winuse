package main

import (
	"fmt"
	"log"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	rstrtmgr            = windows.NewLazySystemDLL("rstrtmgr.dll")
	rmStartSession      = rstrtmgr.NewProc("RmStartSession")
	rmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	rmGetList           = rstrtmgr.NewProc("RmGetList")
	rmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

//go:generate stringer -type=RM_APP_TYPE
//go:generate stringer -type=RM_REBOOT_REASON

type RM_APP_TYPE int32

const (
	RmUnknownApp  RM_APP_TYPE = 0
	RmMainWindow  RM_APP_TYPE = 1
	RmOtherWindow RM_APP_TYPE = 2
	RmService     RM_APP_TYPE = 3
	RmExplorer    RM_APP_TYPE = 4
	RmConsole     RM_APP_TYPE = 5
	RmCritical    RM_APP_TYPE = 1000
)

type RM_REBOOT_REASON int32

const (
	RmRebootReasonNone             RM_REBOOT_REASON = 0
	RmRebootReasonPermissionDenied RM_REBOOT_REASON = 1
	RmRebootReasonSessionMismatch  RM_REBOOT_REASON = 2
	RmRebootReasonCriticalProcess  RM_REBOOT_REASON = 4
	RmRebootReasonCriticalService  RM_REBOOT_REASON = 8
	RmRebootReasonDetectedSelf     RM_REBOOT_REASON = 16
)

type RM_UNIQUE_PROCESS struct {
	DwProcessId      uint32
	ProcessStartTime windows.Filetime
}

type RM_PROCESS_INFO struct {
	Process             RM_UNIQUE_PROCESS
	StrAppName          [256]uint16
	StrServiceShortName [64]uint16
	ApplicationType     RM_APP_TYPE
	AppStatus           uint32
	SessionId           uint32
	Restartable         int32
}

func main() {
	var (
		session uint32
		sesskey [32]uint16

		reason      RM_REBOOT_REASON
		nproc       uint32
		nprocneeded uint32
	)

	// Convert arguments into wchar_t strings.
	filenames := make([]*uint16, len(os.Args))
	for _, arg := range os.Args[1:] {
		u16fn, err := windows.UTF16PtrFromString(arg)
		if err != nil {
			log.Fatalf("Invalid command line argument: %v", err)
		}
		filenames = append(filenames, u16fn)
	}

	// Call RmStartSession
	ret, _, _ := rmStartSession.Call(
		uintptr(unsafe.Pointer(&session)),    // pSessionHandle
		uintptr(0),                           // dwSessionFlags
		uintptr(unsafe.Pointer(&sesskey[0])), // strSessionKey
	)
	if ret != uintptr(windows.ERROR_SUCCESS) {
		log.Fatalf("RmStartSession: returned %08x", ret)
	}

	// Call RmEndSession at the end of the program
	defer rmEndSession.Call(uintptr(session))

	// Call RmRegisterResources for all arguments
	ret, _, _ = rmRegisterResources.Call(
		uintptr(session),                       // dwSessionHandle
		uintptr(len(filenames)),                // nFiles
		uintptr(unsafe.Pointer(&filenames[0])), // rgsFileNames
		uintptr(0),                             // nApplications
		uintptr(0),                             // rgApplications
		uintptr(0),                             // nServices
		uintptr(0),                             // rgsServiceNames
	)
	if ret != uintptr(windows.ERROR_SUCCESS) {
		log.Fatalf("RmRegisterResources: returned %08x", ret)
	}

	// Call RmGetList to get the number of proc info structs.
	ret, _, _ = rmGetList.Call(
		uintptr(session),                      // dwSessionHandle
		uintptr(unsafe.Pointer(&nprocneeded)), // pnProcInfoNeeded
		uintptr(unsafe.Pointer(&nproc)),       // pnProcInfo
		uintptr(0),                            // rgAffectedApps
		uintptr(unsafe.Pointer(&reason)),      // lpdwRebootReasons
	)
	if ret == uintptr(windows.ERROR_SUCCESS) {
		log.Println("RmGetList: no data")
		return
	} else if ret != uintptr(windows.ERROR_MORE_DATA) {
		log.Fatalf("RmGetList: returned %08x", ret)
	}

	// Call RmGetList to get the proc info structs.
	procs := make([]RM_PROCESS_INFO, nprocneeded)
	nproc = nprocneeded
	ret, _, _ = rmGetList.Call(
		uintptr(session),                      // dwSessionHandle
		uintptr(unsafe.Pointer(&nprocneeded)), // pnProcInfoNeeded
		uintptr(unsafe.Pointer(&nproc)),       // pnProcInfo
		uintptr(unsafe.Pointer(&procs[0])),    // rgAffectedApps
		uintptr(unsafe.Pointer(&reason)),      // lpdwRebootReasons
	)
	if ret != uintptr(windows.ERROR_SUCCESS) {
		log.Fatalf("RmGetList: returned %08x", ret)
	}

	// Print out user-friendly list of processes.
	fmt.Printf("processes:\n")
	for _, proc := range procs {
		fmt.Printf(
			"- %d (AppName=%s)\n",
			proc.Process.DwProcessId,
			windows.UTF16ToString(proc.StrAppName[:]),
		)
	}
	fmt.Printf("reason: %s\n", reason)
}
