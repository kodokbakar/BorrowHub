package domain

import (
	"testing"
	"time"
)

func TestReservationOverlap(t *testing.T) {
	base := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		start1   time.Time
		end1     time.Time
		start2   time.Time
		end2     time.Time
		overlaps bool
	}{
		{
			name:     "disjoint before",
			start1:   base,
			end1:     base.Add(2 * time.Hour), // 10:00 - 12:00
			start2:   base.Add(3 * time.Hour), // 13:00 - 15:00
			end2:     base.Add(5 * time.Hour),
			overlaps: false,
		},
		{
			name:     "adjacent abutting end1 equals start2",
			start1:   base,
			end1:     base.Add(2 * time.Hour), // 10:00 - 12:00
			start2:   base.Add(2 * time.Hour), // 12:00 - 14:00
			end2:     base.Add(4 * time.Hour),
			overlaps: false,
		},
		{
			name:     "adjacent abutting start1 equals end2",
			start1:   base.Add(2 * time.Hour), // 12:00 - 14:00
			end1:     base.Add(4 * time.Hour),
			start2:   base, // 10:00 - 12:00
			end2:     base.Add(2 * time.Hour),
			overlaps: false,
		},
		{
			name:     "partial overlap start1 before start2, end1 inside",
			start1:   base,
			end1:     base.Add(3 * time.Hour), // 10:00 - 13:00
			start2:   base.Add(2 * time.Hour), // 12:00 - 15:00
			end2:     base.Add(5 * time.Hour),
			overlaps: true,
		},
		{
			name:     "partial overlap start2 before start1, end2 inside",
			start1:   base.Add(2 * time.Hour), // 12:00 - 15:00
			end1:     base.Add(5 * time.Hour),
			start2:   base, // 10:00 - 13:00
			end2:     base.Add(3 * time.Hour),
			overlaps: true,
		},
		{
			name:     "enclosure 2 inside 1",
			start1:   base,
			end1:     base.Add(6 * time.Hour), // 10:00 - 16:00
			start2:   base.Add(2 * time.Hour), // 12:00 - 14:00
			end2:     base.Add(4 * time.Hour),
			overlaps: true,
		},
		{
			name:     "enclosure 1 inside 2",
			start1:   base.Add(2 * time.Hour), // 12:00 - 14:00
			end1:     base.Add(4 * time.Hour),
			start2:   base, // 10:00 - 16:00
			end2:     base.Add(6 * time.Hour),
			overlaps: true,
		},
		{
			name:     "identical interval",
			start1:   base,
			end1:     base.Add(2 * time.Hour), // 10:00 - 12:00
			start2:   base,
			end2:     base.Add(2 * time.Hour),
			overlaps: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DoReservationsOverlap(tt.start1, tt.end1, tt.start2, tt.end2)
			if got != tt.overlaps {
				t.Fatalf("expected overlap %v, got %v", tt.overlaps, got)
			}
		})
	}
}
