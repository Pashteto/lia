package validator

import (
	"errors"

	"github.com/gofrs/uuid"
)

type Rule func() error

func Check(rules ...Rule) error {
	for _, rule := range rules {
		if err := rule(); err != nil {
			return err
		}
	}

	return nil
}

func CheckForm(rules ...Rule) []error {
	var res []error
	for _, rule := range rules {
		if err := rule(); err != nil {
			res = append(res, err)
		}
	}

	if len(res) != 0 {
		return res
	}

	return nil
}

func Request[K comparable](in *K) error {
	if in == nil {
		return ErrRequestEmpty
	}
	return nil
}

func Optional[T any](v *T, next Rule) Rule {
	return func() error {
		if v == nil {
			return nil
		}
		return next()
	}
}

func RuleUUID(s string, err error) Rule {
	return func() error {
		if s == "" {
			return ErrEmptyString
		}

		_, parseErr := uuid.FromString(s)
		if parseErr != nil {
			return errors.Join(parseErr, err)
		}

		return nil
	}
}
