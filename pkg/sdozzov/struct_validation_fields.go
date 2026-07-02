package sdozzov

import (
	ozzov "github.com/go-ozzo/ozzo-validation/v4"
)

type StructValidationFields []string

var Pass ozzov.Rule = ozzov.By(func(_ interface{}) error {
	return nil
})

func (fields StructValidationFields) IfInclude(field string) ozzov.Rule {
	if len(fields) <= 0 {
		return Pass
	} else {
		for _, f := range fields {
			if f == field {
				return Pass
			}
		}
		return ozzov.Skip
	}
}
