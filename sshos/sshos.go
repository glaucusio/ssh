package sshos

import (
	"github.com/glaucusio/xerrors"
)

func is(err error, errs ...error) bool {
	for _, e := range errs {
		if xerrors.Is(err, e) {
			return true
		}
	}
	return false
}
