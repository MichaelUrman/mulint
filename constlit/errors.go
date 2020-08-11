package constlit

import (
	"errors"
	"strings"
)

func wraperr(errs ...error) error {
	nonnil := errs[:0]
	for _, err := range errs {
		if err != nil {
			nonnil = append(nonnil, err)
		}
	}
	if len(nonnil) == 0 {
		return nil
	}
	return multiError(nonnil)
}

type multiError []error

func (e multiError) Error() string {
	b := strings.Builder{}
	for i, err := range e {
		if i > 0 {
			b.WriteRune('\n')
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (e multiError) Is(err error) bool {
	for _, er := range e {
		if errors.Is(er, err) {
			return true
		}
	}
	return false
}

func (e multiError) As(target interface{}) bool {
	for _, er := range e {
		if errors.As(er, target) {
			return true
		}
	}
	return false
}
