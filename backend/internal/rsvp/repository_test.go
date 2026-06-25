package rsvp

import (
	"testing"

	"github.com/Pashteto/lia/internal/models"
)

func TestSeatAvailable(t *testing.T) {
	cap2 := 2
	cases := []struct {
		name     string
		taken    int
		capacity *int
		want     bool
	}{
		{"unlimited", 100, nil, true},
		{"room left", 1, &cap2, true},
		{"exactly full", 2, &cap2, false},
		{"over full", 3, &cap2, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := seatAvailable(c.taken, c.capacity); got != c.want {
				t.Fatalf("seatAvailable(%d,%v)=%v want %v", c.taken, c.capacity, got, c.want)
			}
		})
	}
	_ = models.RsvpGoing
}
