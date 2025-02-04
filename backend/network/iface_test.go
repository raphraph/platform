package network

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIPv4(t *testing.T) {

	ip, err := LocalIPv4()

	assert.True(t, err != nil || ip != nil)
}

func TestIPv6(t *testing.T) {

	ip, err := IPv6()

	assert.True(t, err != nil || ip != nil)
}
