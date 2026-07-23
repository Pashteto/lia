package handlers

import "testing"

func TestIsWithdrawOnly(t *testing.T) {
	cancelled := "cancelled"
	published := "published"
	cases := []struct {
		name string
		in   *string
		want bool
	}{
		{"cancel is withdraw", &cancelled, true},
		{"publish is not withdraw", &published, false},
		{"nil status is not withdraw", nil, false},
	}
	for _, c := range cases {
		if got := isWithdraw(c.in); got != c.want {
			t.Fatalf("%s: isWithdraw=%v want %v", c.name, got, c.want)
		}
	}
}
