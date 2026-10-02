package upload

const maxUploadBytes = 20 * 1024 * 1024 // 20 MB

var allowedMIME = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

func IsAllowedMIME(m string) bool { return allowedMIME[m] }

func IsAllowedSize(n int64) bool { return n > 0 && n <= maxUploadBytes }

func ScopeAllowsMIME(scope, mime string) bool {
	switch scope {
	case "ProfileImage":
		return mime == "image/jpeg" || mime == "image/png" || mime == "image/webp"
	case "PostAttachment", "MedicalAttachment":
		return IsAllowedMIME(mime)
	}
	return false
}
