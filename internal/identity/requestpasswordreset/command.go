package requestpasswordreset

type Command struct {
	Email    string `json:"email"    validate:"required,email,max=320"`
	ClientIP string `json:"-"`
}