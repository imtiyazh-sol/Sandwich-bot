package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func Parse(_data []byte, _struct any) error {
	if err := json.Unmarshal(_data, &_struct); err != nil {
		return err
	}

	_validator := validator.New()
	// if err := _validator.RegisterValidation("", ); err != nil {
	// 	return err
	// }

	_validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	if err := _validator.Struct(_struct); err != nil {
		_errTxt := []string{}
		for _, _e := range err.(validator.ValidationErrors) {
			_errTxt = append(_errTxt, "Field "+_e.Field())
			switch _e.Tag() {
			case "required":
				_errTxt = append(_errTxt, "is required")
			case "min":
				_errTxt = append(_errTxt, "should not be less than "+_e.Param()+" symbols.")
			case "max":
				fmt.Println(_e.Error())
				_errTxt = append(_errTxt, "should not be more than "+_e.Param()+" symbols.")
			case "gte":
				_errTxt = append(_errTxt, "should be >= "+_e.Param()+".")
			case "lte":
				_errTxt = append(_errTxt, "should be <= "+_e.Param()+".")
			case "slugValid":
				_errTxt = append(_errTxt, "contains forbidden characters.")
			case "slugvalid":
				_errTxt = append(_errTxt, "contains forbidden characters.")
			}
		}

		return errors.New(strings.Join(_errTxt, " "))
	}

	return nil
}
