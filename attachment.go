package mailman

import "io"

type Attachment interface {
	Filename() string
	ContentType() string
	Data() io.Reader
}
