package compose

import (
	"fmt"
	"strings"

	"github.com/docker/docker/client"
)

func StandardRecomposator(docker *client.Client) (*Recomposator, error) {
	return NewRecomposator(docker, map[string]interface{}{
		"VolumeInVolumes": "./volumes",
	})
}

type VolumePatcher func(src string) (string, error)
type ServicePatcher func(service map[string]interface{}) error

type Recomposator struct {
	docker          *client.Client
	networks        *Networks
	volumePatchers  []VolumePatcher
	servicePatchers []ServicePatcher
}

func (r *Recomposator) UseVolumePatcher(p VolumePatcher) {
	r.volumePatchers = append(r.volumePatchers, p)
}

func (r *Recomposator) UseServicePatcher(p ServicePatcher) {
	r.servicePatchers = append(r.servicePatchers, p)
}

func PatchVolumeInVolumes(target string) (VolumePatcher, error) {
	if !strings.HasPrefix(target, "./") {
		return nil, fmt.Errorf("Target must be relative: %s", target)
	}
	return func(volume string) (string, error) {
		slugs := strings.Split(volume, ":")
		src := slugs[0]
		if !strings.HasPrefix(src, "./") {
			return "", fmt.Errorf("Wrong volume prefix: %s", src)
		}
		ro := ""
		if len(slugs) == 3 {
			ro = fmt.Sprintf(":%s", slugs[2])
		}
		if !strings.HasPrefix(src, fmt.Sprintf("%s/", target)) {
			return fmt.Sprintf("%s/%s:%s%s", target, src[2:], slugs[1], ro), nil
		}
		return src, nil
	}, nil
}

func NewRecomposator(docker *client.Client, cfg map[string]interface{}) (*Recomposator, error) {
	n, err := NewNetworks(docker)
	if err != nil {
		return nil, err
	}
	r := &Recomposator{
		docker:          docker,
		networks:        n,
		volumePatchers:  make([]VolumePatcher, 0),
		servicePatchers: make([]ServicePatcher, 0),
	}
	for k, v := range cfg {
		switch k {
		case "VolumeInVolumes":
			path, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("VolumeInVolume argument is a string: %v", v)
			}
			patcher, err := PatchVolumeInVolumes(path)
			if err != nil {
				return nil, err
			}
			r.UseVolumePatcher(patcher)
		default:
			return nil, fmt.Errorf("Unknown patch: %s", k)
		}
	}
	return r, nil
}

// Recompose take a naive and validated Compose and return a Compose as it will be run
func (r *Recomposator) Recompose(name string, c *Compose) (*Compose, error) {
	networkName, err := r.networks.New(name)
	if err != nil {
		return nil, err
	}
	prod := &Compose{
		Services: copyMap(c.Services),
		Version:  c.Version,
		X:        copyMap(c.X),
		Networks: map[string]interface{}{
			"default": map[string]interface{}{
				"external": map[string]interface{}{
					"name": networkName,
				},
			},
		},
	}
	prod.WalkServices(func(name string, service map[string]interface{}) error {
		volumesRaw, ok := service["volumes"]
		if ok {
			volumes, err := castVolumes(volumesRaw)
			if err != nil {
				return err
			}
			vv := make([]string, len(volumes))
			for i, volume := range volumes {
				v := volume
				for _, patcher := range r.volumePatchers {
					v, err = patcher(v)
					if err != nil {
						return err
					}
				}
				vv[i] = v
			}
			service["volumes"] = vv
		}
		labelsRaw, ok := service["labels"]
		if !ok {
			service["labels"] = map[string]string{
				"batch": name,
			}
			return nil
		}
		labels, ok := labelsRaw.(map[string]string)
		if !ok {
			return fmt.Errorf("labels is not a map %v", labelsRaw)
		}
		labels["batch"] = name
		return nil
	})
	return prod, nil
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
