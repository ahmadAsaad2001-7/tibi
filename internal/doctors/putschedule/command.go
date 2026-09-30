package putschedule

type Command struct {
	Weekly []BlockInput `json:"weekly" validate:"dive"`
}

type BlockInput struct {
	DayOfWeek           int    `json:"day_of_week"            validate:"min=0,max=6"`
	StartTime           string `json:"start_time"             validate:"required"`
	EndTime             string `json:"end_time"               validate:"required"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"  validate:"min=5,max=240"`
	IsActive            bool   `json:"is_active"`
}
