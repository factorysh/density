package compose

import (
	"fmt"
	"strings"
)

type ComposeValidator struct {
}

func (cv *ComposeValidator) Validate(c *Compose) []error {
	errs := make([]error, 0)
	c.WalkServices(func(name string, value map[string]interface{}) error {
		volumesRaw, ok := value["volumes"]
		if !ok {
			return nil
		}
		volumes, ok := volumesRaw.([]string)
		if !ok {
			return fmt.Errorf("bad volumes format %v", volumesRaw)
		}
		for _, volume := range volumes {
			err := validateServiceVolume(volume)
			if err != nil {
				errs = append(errs, fmt.Errorf("On service %s : %s", name, err.Error()))
			}
		}
		_, ok = value["build"]
		if ok {
			errs = append(errs, fmt.Errorf("On service %s : can't buid here", name))
		}
		_, ok = value["logging"]
		if ok {
			errs = append(errs, fmt.Errorf("On service %s : can't logging here", name))
		}
		return nil
	})
	return errs
}

func validateServiceVolume(volume string) error {
	if !strings.HasPrefix(volume, "./") {
		return fmt.Errorf("Relative volume only %v", volume)
	}
	return nil
}
