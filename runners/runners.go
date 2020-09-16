package runners

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// EnsureBin check if a bin is available in path
func EnsureBin(name string) error {
	var out bytes.Buffer

	cmd := exec.Command("whereis", "-b", name)
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return err
	}

	sanitized := strings.TrimRight(out.String(), "\n")
	matched, err := regexp.Match(fmt.Sprintf("%s: .*/%s", name, name), []byte(sanitized))
	if !matched {
		return errors.Errorf("Executable %s not found", name)
	}

	return nil

}
