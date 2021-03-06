package compose

import (
	"errors"
	"fmt"
	"strings"

	"github.com/factorysh/density/task"
)

var StandardValidtator *ComposeValidator

func init() {
	StandardValidtator = NewComposeValidtor()
	StandardValidtator.UseServiceValidator(ValidateNoBuild)
	StandardValidtator.UseServiceValidator(ValidateNoLogging)
	StandardValidtator.UseVolumeValidator(ValidateVolumeInplace)
	StandardValidtator.UseVolumeValidator(ValidateNoDotDot)
	StandardValidtator.UseVolumeValidator(ValidateNotAsDeep(8))

	// FIXME registering the ComposeValidator add a link to task module.
	// Maybe in another module ?
	task.TaskValidatorRegistry["compose"] = func(cfg map[string]interface{}) task.TaskValidator {
		return StandardValidtator
	}
}

type VolumeValidator func(source, destination string, readOnly bool) error
type ServiceValidator func(service map[string]interface{}) error

func ValidateNoBuild(service map[string]interface{}) error {
	_, ok := service["build"]
	if ok {
		return errors.New("Do not build inplace")
	}
	return nil
}

func ValidateNoLogging(service map[string]interface{}) error {
	_, ok := service["logging"]
	if ok {
		return errors.New("The logging is handled by the supervisor")
	}
	return nil
}

func ValidateVolumeInplace(src, dest string, ro bool) error {
	if !strings.HasPrefix(src, "./") {
		return fmt.Errorf("Relative volume only %v", src)
	}
	return nil
}

func ValidateNoDotDot(src, dest string, ro bool) error {
	for _, slug := range strings.Split(src, "/") {
		if strings.HasPrefix(slug, "..") {
			return fmt.Errorf("Path with .. : %s", src)
		}
	}
	return nil
}

func ValidateNotAsDeep(deep int) VolumeValidator {
	return func(src, dest string, ro bool) error {
		if len(strings.Split(src, "/")) > deep {
			return fmt.Errorf("Path is too deep %d : %s", deep, src)
		}
		return nil
	}
}

type ComposeValidator struct {
	volumeValidators  []VolumeValidator
	serviceValidators []ServiceValidator
}

func NewComposeValidtor() *ComposeValidator {
	return &ComposeValidator{
		volumeValidators:  make([]VolumeValidator, 0),
		serviceValidators: make([]ServiceValidator, 0),
	}
}

func (cv *ComposeValidator) UseVolumeValidator(v VolumeValidator) {
	cv.volumeValidators = append(cv.volumeValidators, v)
}

func (cv *ComposeValidator) UseServiceValidator(s ServiceValidator) {
	cv.serviceValidators = append(cv.serviceValidators, s)
}

func castVolumes(volumesRaw interface{}) ([]string, error) {
	volumes, ok := volumesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad volumes format %v", volumesRaw)
	}
	vv := make([]string, len(volumes))
	for i, volumeRaw := range volumes {
		volume, ok := volumeRaw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong volume format: %v", volumeRaw)
		}
		vv[i] = volume
	}
	return vv, nil
}

func (cv *ComposeValidator) Validate(c *Compose) []error {
	errs := make([]error, 0)
	c.WalkServices(func(name string, value map[string]interface{}) error {
		for _, service := range cv.serviceValidators {
			err := service(value)
			if err != nil {
				errs = append(errs, err)
			}
		}
		volumesRaw, ok := value["volumes"]
		if !ok {
			return nil
		}
		volumes, err := castVolumes(volumesRaw)
		if err != nil {
			return err
		}
		for _, volume := range volumes {
			slugs := strings.Split(volume, ":")
			if len(slugs) == 1 || len(slugs) > 3 {
				return fmt.Errorf("Wrong volume format : %s", volume)
			}
			ro := false
			if len(slugs) == 3 {
				ro = slugs[2] == "ro"
			}
			for _, v := range cv.volumeValidators {
				err := v(slugs[0], slugs[1], ro)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
		return nil
	})
	return errs
}

func (cv *ComposeValidator) ValidateTask(t *task.Task) []error {
	c, ok := t.Action.(*Compose)
	if !ok {
		// FIXME nil or error?
		return nil
	}
	return cv.Validate(c)
}
