//go:build !windows

package main

import (
	"log"
	"os/exec"
	"runtime"
)

func openAppWindow(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
func notifyError(message string) { log.Print(message) }
