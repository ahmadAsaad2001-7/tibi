package register

type Command struct {
	Email       string `json:"email"       validate:"required,email,max=255"`
	Password    string `json:"password"    validate:"required,min=12,max=128"`
	Role        string `json:"role"        validate:"required,oneof=Patient Doctor"`
	FullName    string `json:"full_name"   validate:"required,min=2,max=200"`
	PhoneNumber string `json:"phone_number" validate:"required,e164"`
}
