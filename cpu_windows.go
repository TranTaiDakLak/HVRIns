package main

import (
	goruntime "runtime"
	"syscall"
	"time"
	"unsafe"
)

// getProcessCPUTime trả về tổng CPU time (kernel + user) của process hiện tại
func getProcessCPUTime() time.Duration {
	handle, _ := syscall.GetCurrentProcess()
	var creation, exit, kernel, user syscall.Filetime
	err := syscall.GetProcessTimes(handle, &creation, &exit, &kernel, &user)
	if err != nil {
		return 0
	}
	return filetimeToDuration(kernel) + filetimeToDuration(user)
}

// filetimeToDuration chuyển đổi syscall.Filetime (Windows FILETIME) sang time.Duration.
// ft: FILETIME struct từ Windows API (HighDateTime và LowDateTime dạng 100-nanosecond intervals).
// Dùng để tính kernel time + user time của process từ GetProcessTimes().
func filetimeToDuration(ft syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)
	return time.Duration(n * 100)
}

// getNumCPU trả về số CPU cores — dùng để chia CPU% cho chính xác
func getNumCPU() int {
	return goruntime.NumCPU()
}

// PROCESS_MEMORY_COUNTERS_EX — struct Windows API (Win2000+)
type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

// PROCESS_MEMORY_COUNTERS_EX2 — mở rộng EX (Windows 10 version 2004+).
// PrivateWorkingSetSize = physical RAM thuộc riêng process (không shared DLLs)
// — đây chính là con số "Memory" mà Task Manager hiển thị ở cột Processes.
type processMemoryCountersEx2 struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
	PrivateWorkingSetSize      uintptr // KHỚP Task Manager "Memory (active private working set)"
	SharedCommitUsage          uint64
}

var (
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")

	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx  = kernel32.NewProc("GlobalMemoryStatusEx")
)

// memoryStatusEx — Windows MEMORYSTATUSEX struct cho GlobalMemoryStatusEx.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// getSystemMemoryMB trả về tổng RAM vật lý của hệ thống (MB).
// Dùng để auto-detect GOMEMLIMIT phù hợp với từng máy.
// Trả 0 nếu lỗi syscall.
func getSystemMemoryMB() uint64 {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if ret == 0 {
		return 0
	}
	return mem.TotalPhys / 1024 / 1024
}

// getProcessMemoryMB trả về RAM app đang dùng (MB) — khớp Task Manager cột "Memory".
// Ưu tiên PrivateWorkingSetSize (Win10 2004+): bỏ phần shared DLLs (WebView2, Windows)
// → khớp chính xác con số Task Manager thay vì tổng working set (lớn hơn do gồm shared).
// Fallback WorkingSetSize nếu OS cũ hoặc call EX2 thất bại.
func getProcessMemoryMB() float64 {
	handle, _ := syscall.GetCurrentProcess()

	// Thử PROCESS_MEMORY_COUNTERS_EX2 trước
	var memEx2 processMemoryCountersEx2
	memEx2.CB = uint32(unsafe.Sizeof(memEx2))
	ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&memEx2)),
		uintptr(memEx2.CB),
	)
	if ret != 0 && memEx2.PrivateWorkingSetSize > 0 {
		return float64(memEx2.PrivateWorkingSetSize) / 1024 / 1024
	}

	// Fallback EX (Win2000+) — chỉ có WorkingSetSize (gồm shared pages, lớn hơn TaskMgr)
	var mem processMemoryCounters
	mem.CB = uint32(unsafe.Sizeof(mem))
	ret, _, _ = procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&mem)),
		uintptr(mem.CB),
	)
	if ret == 0 {
		return 0
	}
	return float64(mem.WorkingSetSize) / 1024 / 1024
}
