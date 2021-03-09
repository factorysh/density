package compose

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

var (
	StandardValidtator *ComposeValidator
	StandardConfig     map[string]interface{}
	badConfig          map[string]string
)

func init() {
	badConfig = make(map[string]string)
	for _, config := range []string{
		"container_name",
		"cgroup_parent",
		"logging",
		"cap_add",
		"build",
		"domainname",
		"hostname",
		"ipc",
		"mac_address",
		"privileged",
		"stdin_open",
		"tty",
	} {
		badConfig[SnakeToCamel(config)] = config
	}

	StandardConfig = map[string]interface{}{
		"VolumeInplace": nil,
		"NoDotDot":      nil,
		"NotAsDeep":     8,
	}
	for bad := range badConfig {
		StandardConfig["No"+bad] = nil
	}

	var err error
	StandardValidtator, err = NewComposeValidtor(StandardConfig)
	if err != nil {
		panic(err)
	}
}

type VolumeValidator func(source, destination string, readOnly bool) error
type ServiceValidator func(service map[string]interface{}) error

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
	badConfig         []string
}

func NewComposeValidtor(cfg map[string]interface{}) (*ComposeValidator, error) {
	validator := &ComposeValidator{
		volumeValidators:  make([]VolumeValidator, 0),
		serviceValidators: make([]ServiceValidator, 0),
		badConfig:         make([]string, 0),
	}
	for k, v := range cfg {
		switch k {
		case "VolumeInplace":
			validator.UseVolumeValidator(ValidateVolumeInplace)
		case "NoDotDot":
			validator.UseVolumeValidator(ValidateNoDotDot)
		case "NotAsDeep":
			deep, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("NotAsDeep argument must be an int : %v", v)
			}
			validator.UseVolumeValidator(ValidateNotAsDeep(deep))
		default:
			ok := false
			if strings.HasPrefix(k, "No") {
				var c string
				c, ok = badConfig[k[2:]]
				if ok {
					validator.badConfig = append(validator.badConfig, c)
				}
			}
			if !ok {
				return nil, fmt.Errorf("Unknown validator: %s", k)
			}
		}
	}
	return validator, nil
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
		for _, bad := range cv.badConfig {
			_, ok := value[bad]
			if ok {
				errs = append(errs, fmt.Errorf("the %s config is not available", bad))
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

func SnakeToCamel(txt string) string {
	out := bytes.Buffer{}
	up := true
	for _, a := range txt {
		if a == '_' {
			up = true
		} else {
			if up {
				out.WriteRune(unicode.ToUpper(a))
				up = false
			} else {
				out.WriteRune(a)
			}
		}
	}
	return out.String()
}
