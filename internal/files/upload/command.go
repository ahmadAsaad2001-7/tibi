package upload

import "io"

// Command carries the multipart file plus the target scope. Parsed from
// the request by the handler; the service does not touch *multipart.FileHeader.
type Command struct {
	UploaderID  int64
	Scope       string
	Filename    string
	ContentType string
	Size        int64
	Body        io.Reader
}
