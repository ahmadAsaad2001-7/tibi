package scope

import "context"

// ProfileImageRule always permits reads. Profile images appear on
// anonymous doctor search results; making them require auth would break
// the public listing pages.
type ProfileImageRule struct{}

func New() *ProfileImageRule { return &ProfileImageRule{} }

func (*ProfileImageRule) AuthorizeFileRead(
	ctx context.Context, userID, fileID, uploaderID int64,
) (bool, error) {
	return true, nil
}
