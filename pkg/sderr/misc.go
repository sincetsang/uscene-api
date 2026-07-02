package sderr

func ToErr(v interface{}) error {
	switch err := v.(type) {
	case nil:
		return nil
	case error:
		return err
	case string:
		return New(err)
	default:
		return Newf("%v", err)
	}
}

func Multi(errs []error) error {
	var notNilErrs []error
	for _, err := range errs {
		if err != nil {
			notNilErrs = append(notNilErrs, err)
		}
	}
	if len(notNilErrs) <= 0 {
		return nil
	}
	return Append(nil, notNilErrs...)
}

func Combine(errs ...error) error {
	return Multi(errs)
}
