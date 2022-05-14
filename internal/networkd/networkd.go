package networkd

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"sync"
	"time"
)

func isTTY() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TCGETS)
	return err == nil
}

func animateProgress(step int) {
	progress := ((step + 5.0) % 10.0) - 5.0
	if progress < 0 {
		progress = -progress
	}
	fmt.Fprintf(os.Stderr, "\rWaiting on systemd-network to restart [%*s%*s]", progress+1, "=", 5-progress, "")
}

func progressMessage(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	animate := isTTY()
	count := 0
	endLine := false
	interval := time.NewTicker(time.Second * 1)
	defer interval.Stop()

	for {
		select {
		case <-interval.C:
			if animate {
				animateProgress(count)
			} else if !endLine {
				fmt.Print("Waiting on systemd-networkd to restart...")
			}
			endLine = true
			count++
		case <-ctx.Done():
			if endLine {
				fmt.Fprintf(os.Stderr, "\n")
			}
			return
		}
	}
}

func Restart() error {
	cmd := exec.Command("systemctl", "restart", "systemd-networkd")
	waitGroup := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	waitGroup.Add(1)

	go progressMessage(ctx, &waitGroup)

	err := cmd.Run()
	cancel()
	waitGroup.Wait()

	return err
}
