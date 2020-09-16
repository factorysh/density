package action

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	cs "github.com/compose-spec/compose-go/loader"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Compose represent a struct containing a raw docker-compose.yml file
type Compose struct {
	Raw    string
	Parsed map[string]interface{}
}

// NewCompose inits a new compose file struct
func NewCompose(input []byte) Compose {
	return Compose{
		Raw: string(input),
	}

}

// Parse ensures a docker-compose file, ensure content is valid
func (c *Compose) Parse() error {

	parsed, err := cs.ParseYAML([]byte(c.Raw))
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error when validating compose file: %v", err))
	}

	c.Parsed = parsed

	return nil
}

// Recompose rewrite the file back
func (c *Compose) Recompose() (string, error) {

	ret, err := yaml.Marshal(&c.Parsed)
	if err != nil {
		return "", err
	}

	return string(ret), nil

}

// Run this compose instance
func (c Compose) Run(uuid string) error {

	// ensure binary is present
	if err := ensureBin("docker-compose"); err != nil {
		return err
	}

	// TODO: runs dir comes from env
	rundir := fmt.Sprintf("/tmp/runs/%s", uuid)

	// create run per uuid dir if not exists
	if _, err := os.Stat(rundir); os.IsNotExist(err) {
		err := os.Mkdir(rundir, 755)
		if err != nil {
			return err

		}
	}

	// create temp file
	tmpfile, err := ioutil.TempFile(rundir, fmt.Sprintf("%s-", uuid))
	if err != nil {
		log.Fatal(err)
	}

	// recompose the compose file
	recompose, err := c.Recompose()
	if err != nil {
		return err
	}

	// write the recoposed composed file in temp file
	if _, err := tmpfile.Write([]byte(recompose)); err != nil {
		return err
	}

	// close it
	if err := tmpfile.Close(); err != nil {
		return err
	}

	// run the compose project
	cmd := exec.Command("docker-compose", "-f", tmpfile.Name(), "up", "-d")
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func ensureBin(name string) error {
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
