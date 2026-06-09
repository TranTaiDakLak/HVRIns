// device.go — Android device profile generator
// Đọc dữ liệu từ Config/DeviceInfo/ (cạnh exe). Không còn dùng embed data/.
package fakeinfo

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

// DeviceProfile chứa thông tin thiết bị Android giả
type DeviceProfile struct {
	Manufacturer string // samsung, Google, Xiaomi...
	Brand        string // samsung, google, xiaomi...
	Model        string // SM-S911B, Pixel 8 Pro...
	OSVersion    string // 9, 10, 11, 12, 13
	BuildID      string // SKQ1.210908.001 hoặc Brand-Model
	Density      string // 3.5
	ScreenWidth  int
	ScreenHeight int
	Architecture string // armeabi-v7a
	AndroidID    string // android-{16 hex chars}
}

var (
	deviceListOnce    sync.Once
	carrierListOnce   sync.Once
	buildnumListOnce  sync.Once
	osVersionsOnce    sync.Once
	densitiesOnce     sync.Once
	screenResOnce     sync.Once
	cpuArchOnce       sync.Once

	deviceList   []DeviceProfile
	carrierList  []string
	buildnumList []string
	osVersionsFI []string
	densitiesFI  []string
	screenResFI  [][2]int
	cpuArchFI    []string
)

// loadDeviceInfoLines đọc lines từ Config/DeviceInfo/<filename>, bỏ blank + comment.
func loadDeviceInfoLines(filename string) []string {
	base := uabuilder.GetConfigBaseDir()
	path := filepath.Join(base, "DeviceInfo", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

func getDeviceList() []DeviceProfile {
	deviceListOnce.Do(func() {
		lines := loadDeviceInfoLines("devices.txt")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) != 3 {
				continue
			}
			deviceList = append(deviceList, DeviceProfile{
				Manufacturer: parts[0],
				Brand:        parts[1],
				Model:        parts[2],
			})
		}
	})
	return deviceList
}

func getCarrierList() []string {
	carrierListOnce.Do(func() {
		carrierList = loadDeviceInfoLines("carriers.txt")
	})
	return carrierList
}

func getBuildnumList() []string {
	buildnumListOnce.Do(func() {
		buildnumList = loadDeviceInfoLines("buildnums.txt")
	})
	return buildnumList
}

func getOSVersionList() []string {
	osVersionsOnce.Do(func() {
		osVersionsFI = loadDeviceInfoLines("os_versions.txt")
		if len(osVersionsFI) == 0 {
			osVersionsFI = []string{"9", "10", "11", "12", "13"}
		}
	})
	return osVersionsFI
}

func getDensities() []string {
	densitiesOnce.Do(func() {
		densitiesFI = loadDeviceInfoLines("densitis.txt")
		if len(densitiesFI) == 0 {
			densitiesFI = []string{"2.0", "2.5", "3.0", "3.5", "4.0"}
		}
	})
	return densitiesFI
}

func getScreenResolutions() [][2]int {
	screenResOnce.Do(func() {
		for _, line := range loadDeviceInfoLines("screen_resolution.txt") {
			parts := strings.SplitN(line, "x", 2)
			if len(parts) != 2 {
				continue
			}
			w, errW := strconv.Atoi(parts[0])
			h, errH := strconv.Atoi(parts[1])
			if errW != nil || errH != nil || w == 0 || h == 0 {
				continue
			}
			screenResFI = append(screenResFI, [2]int{w, h})
		}
		if len(screenResFI) == 0 {
			screenResFI = [][2]int{{1080, 2340}, {1080, 2400}, {1440, 3088}}
		}
	})
	return screenResFI
}

func getCPUArchitectures() []string {
	cpuArchOnce.Do(func() {
		cpuArchFI = loadDeviceInfoLines("device_cores.txt")
		if len(cpuArchFI) == 0 {
			cpuArchFI = []string{"armeabi-v7a", "arm64-v8a"}
		}
	})
	return cpuArchFI
}

// RandomCarrier trả về carrier ngẫu nhiên từ C# carriers.txt
func RandomCarrier() string {
	list := getCarrierList()
	if len(list) > 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
		return list[r.Intn(len(list))]
	}
	return "T-Mobile"
}

// RandomDeviceProfile tạo device profile ngẫu nhiên
func RandomDeviceProfile() DeviceProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	var dp DeviceProfile
	devices := getDeviceList()
	builds := getBuildnumList()
	osVers := getOSVersionList()

	if len(devices) > 0 {
		base := devices[r.Intn(len(devices))]
		dp.Manufacturer = base.Manufacturer
		dp.Brand = base.Brand
		dp.Model = base.Model
	} else {
		dp.Manufacturer = "samsung"
		dp.Brand = "samsung"
		dp.Model = "SM-S911B"
	}

	if len(osVers) > 0 {
		dp.OSVersion = osVers[r.Intn(len(osVers))]
	} else {
		dp.OSVersion = "13"
	}

	densities := getDensities()
	dp.Density = densities[r.Intn(len(densities))]

	resolutions := getScreenResolutions()
	res := resolutions[r.Intn(len(resolutions))]
	dp.ScreenWidth = res[0]
	dp.ScreenHeight = res[1]

	arches := getCPUArchitectures()
	dp.Architecture = arches[r.Intn(len(arches))]

	if len(builds) > 0 {
		dp.BuildID = builds[r.Intn(len(builds))]
	} else {
		dp.BuildID = dp.Brand + "-" + dp.Model
	}

	hexChars := "0123456789abcdef"
	androidID := make([]byte, 16)
	for i := range androidID {
		androidID[i] = hexChars[r.Intn(len(hexChars))]
	}
	dp.AndroidID = "android-" + string(androidID)

	return dp
}
