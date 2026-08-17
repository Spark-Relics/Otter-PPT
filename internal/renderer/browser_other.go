//go:build !windows

package renderer

import "syscall"

// hideWindowProcAttr is a no-op on non-Windows platforms.
func hideWindowProcAttr() *syscall.SysProcAttr { return nil }
