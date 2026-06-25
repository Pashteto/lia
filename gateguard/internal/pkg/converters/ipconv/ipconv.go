package ipconv

import (
	"encoding/binary"
	"fmt"
	"net"
)

// IpToUint32 converts a string representation of an IPv4 address to uint32.
// It returns the uint32 representation and any error encountered.
// Errors can occur if the input string is not a valid IP or is not an IPv4 address.
func IpToUint32(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address")
	}

	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("not an IPv4 address")
	}

	return binary.BigEndian.Uint32(ip), nil
}
