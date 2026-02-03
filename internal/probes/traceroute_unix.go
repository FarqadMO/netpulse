// +build !windows

package probes

import (
	"syscall"
)

// setTTL returns a control function to set TTL on Unix systems.
func setTTL(ttl int) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var opErr error
		err := c.Control(func(fd uintptr) {
			opErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
		})
		if err != nil {
			return err
		}
		return opErr
	}
}
