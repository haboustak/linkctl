package networkd

import (
    "os/exec"
)

func Restart() error {
    cmd := exec.Command("systemctl", "restart", "systemd-networkd")

    return cmd.Run()
}
