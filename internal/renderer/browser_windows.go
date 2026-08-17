//go:build windows

package renderer

import "syscall"

// hideWindowProcAttr prevents a console window flash when shelling out to
// the browser on Windows.
func hideWindowProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
