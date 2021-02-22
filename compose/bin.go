package compose

import (
	"bytes"
	"fmt"
	"os/exec"
)

var composeIsHere bool = false

// EnsureBin will ensure that docker-compose is found in $PATH
func EnsureBin() error {
	var name = "docker-compose"
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("which", name)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		print(stderr.String())
		return fmt.Errorf("%s not found: %s", name, err.Error())
	}
	return nil
}

func lazyEnsureBin() error {
	if composeIsHere {
		return nil
	}
	err := EnsureBin()
	if err != nil {
		return err
	}
	composeIsHere = true
	return nil
}
