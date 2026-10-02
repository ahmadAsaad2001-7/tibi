package book

type Command struct {
	DoctorProfileID int64   `json:"doctor_profile_id" validate:"required,gt=0"`
	ScheduledAt     string  `json:"scheduled_at"      validate:"required"`
	IsUrgent        bool    `json:"is_urgent"`
	PaymentChannel  string  `json:"payment_channel"   validate:"required,oneof=Card MobileWallet InstaPay"`
	Notes           *string `json:"notes"             validate:"omitempty,max=1000"`
	ReturnURL       string  `json:"return_url" validate:"required,url"`
	CancelURL       string  `json:"cancel_url" validate:"required,url"`
}
