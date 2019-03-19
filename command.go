package semvertag

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Execute runs the passed command, waits for it to complete and returns errors, if any
func Execute(dir string, cmd string, args ...string) error {
	_, err := Capture(dir, cmd, args...)
	return err
}

// Capture executes the command and returns its output
func Capture(dir string, cmd string, args ...string) (string, error) {
	log.Printf("Executing `%s`", fmt.Sprintf("%s %s", cmd, strings.Join(args, " ")))
	c := exec.Command(cmd, args...)
	c.Dir = dir

	output, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Error executing %v, output was %s", err, string(output))
	}

	log.Printf("...ok")

	return string(output), nil
}
