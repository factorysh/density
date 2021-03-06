package task

import (
	"fmt"
)

var ActionValidatorRegistry map[string]func(map[string]interface{}) (ActionValidator, error)

func init() {
	if ActionValidatorRegistry == nil {
		ActionValidatorRegistry = make(map[string]func(map[string]interface{}) (ActionValidator, error))
	}
	ActionValidatorRegistry["dummy"] = func(map[string]interface{}) (ActionValidator, error) {
		return &DummyActionValidator{}, nil
	}
}

type ActionValidator interface {
	ValidateAction(a Action) []error
}

type DummyActionValidator struct{}

func (d *DummyActionValidator) ValidateAction(a Action) []error {
	return nil
}

type Validator struct {
	Validators   map[string]map[string]interface{} `yaml:"validators"`
	myValidators map[string]ActionValidator
}

func (val *Validator) Register() error {
	val.myValidators = make(map[string]ActionValidator)
	for k, v := range val.Validators {
		validator, ok := ActionValidatorRegistry[k]
		if !ok {
			return fmt.Errorf("No config validator for %s", k)
		}
		var err error
		val.myValidators[k], err = validator(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) ValidateAction(a Action) []error {
	errs := make([]error, 0)
	validator, ok := v.myValidators[a.RegisteredName()]
	if !ok {
		errs = append(errs, fmt.Errorf("No config for %s", a.RegisteredName()))
		return errs
	}

	errz := validator.ValidateAction(a)
	if errz != nil {
		for _, err := range errz {
			errs = append(errs, err)
		}
	}
	return errs
}
