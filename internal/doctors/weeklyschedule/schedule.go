package weeklyschedule

import (
	"fmt"
	"sort"
	"time"

	"tibi/internal/doctors/doctorprofile"
)

// Block is one working block on a given day-of-week.
type Block struct {
	DayOfWeek           int
	StartTime           doctorprofile.TimeOfDay
	EndTime             doctorprofile.TimeOfDay
	SlotDurationMinutes int
	IsActive            bool
}

// Schedule is the aggregate: the full set of weekly blocks for one doctor.
// Identity is the doctor_profile_id; there is no Schedule row in the DB.
// The invariant (no overlaps on the same active day) is a property of the
// set, not of any single block.
type Schedule struct {
	DoctorProfileID int64
	Blocks          []Block
}

func New(doctorProfileID int64, blocks []Block) *Schedule {
	return &Schedule{DoctorProfileID: doctorProfileID, Blocks: blocks}
}

func (s *Schedule) Validate() error {
	byDay := map[int][]Block{}
	for _, b := range s.Blocks {
		if !b.IsActive {
			continue
		}
		if !b.StartTime.Before(b.EndTime) {
			return fmt.Errorf("block start must be before end (day %d, %s-%s)",
				b.DayOfWeek, b.StartTime, b.EndTime)
		}
		byDay[b.DayOfWeek] = append(byDay[b.DayOfWeek], b)
	}
	for day, blocks := range byDay {
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].StartTime.Before(blocks[j].StartTime)
		})
		for i := 1; i < len(blocks); i++ {
			if blocks[i].StartTime.Minutes() < blocks[i-1].EndTime.Minutes() {
				return fmt.Errorf("overlapping blocks on day %d: %s-%s and %s-%s",
					day,
					blocks[i-1].StartTime, blocks[i-1].EndTime,
					blocks[i].StartTime, blocks[i].EndTime)
			}
		}
	}
	return nil
}

// ActiveBlocksFor returns active blocks for a day-of-week, sorted by start.
func (s *Schedule) ActiveBlocksFor(dayOfWeek int) []Block {
	var out []Block
	for _, b := range s.Blocks {
		if b.IsActive && b.DayOfWeek == dayOfWeek {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartTime.Before(out[j].StartTime)
	})
	return out
}

// Used only to satisfy the "time" import for future use.
var _ = time.Now
