package ipconv_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gateguard/internal/pkg/converters/ipconv"
)

func TestIpToUint32AndBack(t *testing.T) {
	testCases := []struct {
		ipStr string
		want  uint32
	}{
		{
			"192.168.1.1",
			3232235777,
		},
		{
			"255.255.255.255",
			4294967295,
		},
		{
			"0.0.0.0",
			0,
		},
	}

	for _, tc := range testCases {
		got, err := ipconv.IpToUint32(tc.ipStr)
		require.NoErrorf(t, err, "ipToUint32(%s) returned an error: %v", tc.ipStr, err)

		require.Equalf(t, tc.want, got, "ipToUint32(%s) = %d, want %d", tc.ipStr, got, tc.want)
	}
}
