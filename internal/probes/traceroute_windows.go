// +build windows

package probes

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// setTTL returns a control function to set TTL on Windows.
func setTTL(ttl int) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var opErr error
		err := c.Control(func(fd uintptr) {
			opErr = windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, windows.IP_TTL, ttl)
		})
		if err != nil {
			return err
		}
		return opErr
	}
}
