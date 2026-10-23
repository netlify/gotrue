package api

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPrivateIP(t *testing.T) {
	for _, addr := range []string{
		"0.0.0.0", "0.1.2.3", "::", "::ffff:0.0.0.0",
		"127.0.0.1", "::1", "::ffff:127.0.0.1",
		"10.0.0.1", "172.16.0.1", "192.168.1.1", "100.64.0.1",
		"169.254.169.254", "fe80::1", "fd00::1",
	} {
		assert.True(t, isPrivateIP(net.ParseIP(addr)), addr)
	}

	for _, addr := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		assert.False(t, isPrivateIP(net.ParseIP(addr)), addr)
	}
}
