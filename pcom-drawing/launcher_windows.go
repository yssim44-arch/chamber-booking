//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func openAppWindow(url string) error {
	candidates := []string{filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe")}
	for _, p := range candidates {
		if _, e := os.Stat(p); e == nil {
			return exec.Command(p, "--app="+url, "--window-size=1500,980", "--no-first-run").Start()
		}
	}
	shell := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	target, _ := syscall.UTF16PtrFromString(url)
	result, _, _ := shell.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if result <= 32 {
		return fmt.Errorf("기본 브라우저 실행 오류(%d)", result)
	}
	return nil
}
func notifyError(message string) {
	title, _ := syscall.UTF16PtrFromString("pCOM Drawing")
	text, _ := syscall.UTF16PtrFromString(message)
	_, _, _ = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}
