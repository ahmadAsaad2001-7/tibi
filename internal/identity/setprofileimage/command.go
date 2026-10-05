package setprofileimage

type Command struct {
	FileID int64 `json:"file_id" validate:"required,gt=0"`
}
