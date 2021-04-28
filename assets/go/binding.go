package vgg

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/StirlingMarketingGroup/go-namecase"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"
)

type Validator struct {
	once      sync.Once
	validate  *validator.Validate
	modifiers *mold.Transformer
}

type sliceValidateError []error

func (err sliceValidateError) Error() string {
	var errMsgs []string
	for i, e := range err {
		if e == nil {
			continue
		}
		errMsgs = append(errMsgs, fmt.Sprintf("[%d]: %s", i, e.Error()))
	}
	return strings.Join(errMsgs, "\n")
}

var _ binding.StructValidator = &Validator{}

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *Validator) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		v.lazyinit()
		err := v.modifiers.Struct(context.Background(), obj)
		if err != nil {
			return err
		}

		return v.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return v.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(sliceValidateError, 0)
		for i := 0; i < count; i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

// validateStruct receives struct type
func (v *Validator) validateStruct(obj interface{}) error {
	v.lazyinit()
	return v.validate.Struct(obj)
}

// Engine returns the underlying validator engine which powers the default
// Validator instance. This is useful if you want to register custom validations
// or struct level validations. See validator GoDoc for more info -
// https://godoc.org/gopkg.in/go-playground/validator.v8
func (v *Validator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

var rawBase64URLRegex = regexp.MustCompile("^[A-Za-z0-9-_]*$")
var xidRegex = regexp.MustCompile("^[0-9a-v]{20}$")

func (v *Validator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
		v.validate.RegisterValidation("rawbase64url", func(fl validator.FieldLevel) bool {
			return rawBase64URLRegex.MatchString(fl.Field().String())
		})
		v.validate.RegisterValidation("xid", func(fl validator.FieldLevel) bool {
			return xidRegex.MatchString(fl.Field().String())
		})
		v.validate.RegisterValidation("tel", func(fl validator.FieldLevel) bool {
			p, err := phonenumbers.Parse("tel:+"+strings.TrimSuffix(fl.Field().String(), "tel:+"), "US")
			if err != nil {
				return false
			}

			return phonenumbers.IsValidNumber(p)
		})

		v.modifiers = modifiers.New()
		v.modifiers.SetTagName("mod")
		v.modifiers.Register("name", func(ctx context.Context, fl mold.FieldLevel) error {
			fl.Field().SetString(namecase.New().NameCase(fl.Field().String()))
			return nil
		})
		v.modifiers.Register("tel", func(ctx context.Context, fl mold.FieldLevel) error {
			p, err := phonenumbers.Parse("tel:+"+strings.TrimPrefix(fl.Field().String(), "tel:+"), "US")
			if err != nil {
				return nil
			}

			if !phonenumbers.IsValidNumber(p) {
				return nil
			}

			fl.Field().SetString(strings.TrimPrefix(phonenumbers.Format(p, phonenumbers.RFC3966), "tel:"))

			return nil
		})
	})
}
