package submitmessage

type Command struct {
	DoctorProfileID int64   `json:"-"` // from URL
	SenderName      string  `json:"sender_name"  validate:"required,min=1,max=200"`
	SenderEmail     string  `json:"sender_email" validate:"required,email,max=320"`
	SenderPhone     *string `json:"sender_phone" validate:"omitempty,min=5,max=30"`
	Content         string  `json:"content"      validate:"required,min=1,max=4000"`
	SenderIP        string  `json:"-"` // from request
}
