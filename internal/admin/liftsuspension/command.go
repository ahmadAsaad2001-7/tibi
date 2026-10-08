package liftsuspension

type Command struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}