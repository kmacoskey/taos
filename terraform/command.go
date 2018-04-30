package terraform

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

type TerraformCommandRunner interface {
	Run(string, []string) (error, string, string)
}

type TerraformCommand struct{}

func (tc TerraformCommand) Run(directory string, args []string) (error, string, string) {
	logger := log.WithFields(log.Fields{"package": "terraform", "event": "run_command"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	defaultArgs := []string{
		"-no-color",
	}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("terraform %s %s", strings.Join(args, " "), strings.Join(defaultArgs, " ")))

	logger.Debug(cmd)

	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")),
		"CHECKPOINT_DISABLE=1",
		// "TF_LOG=DEBUG",
	}

	if len(directory) == 0 {
		temp_work_dir, err := ioutil.TempDir("", "terraform_client_workingdir")
		if err != nil {
			return err, "", ""
		}
		cmd.Dir = temp_work_dir
	} else {
		cmd.Dir = directory
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	logger.Debug(stdout.String())

	return err, stdout.String(), stderr.String()
}
