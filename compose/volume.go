package compose

import (
	"fmt"
	"os"
	"strings"
)

const volumePrefix = "./volumes"

// Volume represent a basic docker compose volume struct
type Volume struct {
	hostPath      string
	containerPath string
	service       string
}

func (v Volume) checkVolumeRules() error {
	maxDeepness := 10

	// should start with "./"
	if !strings.HasPrefix(v.hostPath, "./") {
		return fmt.Errorf("Volume %v is not a local volume", v)
	}

	// split host path on separator
	hostParts := strings.Split(v.hostPath, string(os.PathSeparator))

	// check max deepness
	if len(hostParts) > maxDeepness {
		return fmt.Errorf("Volume description %v reach deepnees max level %d", v, maxDeepness)
	}

	// inside part (parts[0]) can't contain '..'
	for _, part := range hostParts {
		if part == ".." {
			return fmt.Errorf("Path %v contains `..`", v.hostPath)
		}
	}

	return nil
}

// addPrefix needs to be idempotent, if ./volumes is present, to prepend it another time
func (v *Volume) addPrefix() {

	if !strings.HasPrefix(v.hostPath, "./volumes") {
		v.hostPath = fmt.Sprintf("./volumes/%s", strings.TrimLeft(v.hostPath, "./"))
	}
}

// toVolumeString returns the content of a volume struct to a compose volume string
func (v Volume) toVolumeString() string {
	return fmt.Sprintf("%s:%s", v.hostPath, v.containerPath)
}
