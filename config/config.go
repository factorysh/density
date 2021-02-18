package config

import (
	"fmt"
	"os"
	"strings"
)

// GetDataDir is used to fetch server configured data dir
func GetDataDir() string {
	b := os.Getenv("DATA_DIR")
	if b == "" {
		b = "/tmp/batch-scheduler"
	} else {
		b = strings.TrimLeft(b, "/")
	}

	return b
}

// EnsureDataDirs creates needed directories for data
func EnsureDataDirs() error {

	b := GetDataDir()
	subs := [3]string{"validator", "wd", "store"}

	for _, sub := range subs {
		err := os.MkdirAll(fmt.Sprintf("%s/%s", b, sub), 0755)
		if err != nil {
			return err
		}
	}

	return nil
}
