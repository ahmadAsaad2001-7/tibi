package sendmessage

type Command struct {
	Content string `json:"content" validate:"required,min=1,max=4000"`
}
