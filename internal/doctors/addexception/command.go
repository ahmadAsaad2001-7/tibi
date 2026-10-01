package addexception

type Command struct {
	Date     string  `json:"date"      validate:"required"`
	FromTime string  `json:"from_time" validate:"required"`
	ToTime   string  `json:"to_time"   validate:"required"`
	Type     string  `json:"type"      validate:"required,oneof=Closed OpenEarly OpenLate ModifiedHours"`
	Reason   *string `json:"reason"    validate:"omitempty,max=500"`
}
