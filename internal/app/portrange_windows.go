package app

import (
	"log/slog"
	"os/exec"
)

// expandEphemeralPortRange mở rộng Windows dynamic port range từ 16k (mặc định: 49152-65535)
// lên ~48k (16384-65535). Giảm WSAEADDRINUSE khi chạy nhiều thread concurrent qua proxy.
// Lệnh tương đương: netsh int ipv4 set dynamicport tcp start=16384 num=49152
// Thay đổi có hiệu lực ngay, không cần restart. Yêu cầu quyền admin.
func ExpandEphemeralPortRange() {
	cmd := exec.Command("netsh", "int", "ipv4", "set", "dynamicport", "tcp", "start=16384", "num=49152")
	if err := cmd.Run(); err != nil {
		// Không fatal — app vẫn chạy bình thường nếu netsh fail (ví dụ thiếu quyền admin)
		slog.Warn("expandEphemeralPortRange: netsh failed (có thể thiếu quyền admin)", "err", err)
	}
}
