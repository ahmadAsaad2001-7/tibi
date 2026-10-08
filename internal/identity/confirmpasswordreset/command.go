package confirmpasswordreset

type Command struct {
	Token       string `json:"token"        validate:"required,min=16,max=200"`
	NewPassword string `json:"new_password" validate:"required,min=12,max=128"`
}