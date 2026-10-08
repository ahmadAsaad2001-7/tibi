package confirmemailverification

type Command struct {
	Token string `json:"token" validate:"required,min=16,max=200"`
}