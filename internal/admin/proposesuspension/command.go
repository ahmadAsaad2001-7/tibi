package proposesuspension

type Command struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
	Days   int    `json:"days"   validate:"min=0,max=3650"` // 0 = permanent
}