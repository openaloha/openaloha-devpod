package runfunc

import (
	"os"
)

type InitFunc func() error
type RefreshFunc func(files []*os.File) error
