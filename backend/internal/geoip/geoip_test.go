package geoip

import "testing"

func TestSlugForLocation(t *testing.T) {
	cases := []struct {
		name    string
		country string
		cityID  uint
		subs    []string
		want    string
	}{
		{"moscow city id", "RU", 524901, nil, "msk"},
		{"spb city id", "RU", 498817, nil, "spb"},
		{"moscow oblast", "RU", 0, []string{"MOS"}, "msk"},
		{"leningrad oblast", "RU", 0, []string{"LEN"}, "spb"},
		{"spb subdivision", "RU", 0, []string{"SPE"}, "spb"},
		{"other russian city", "RU", 1496747, []string{"NVS"}, ""},
		{"not russia", "DE", 2950159, []string{"BE"}, ""},
		{"unknown everything", "", 0, nil, ""},
	}
	for _, c := range cases {
		if got := SlugForLocation(c.country, c.cityID, c.subs); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
