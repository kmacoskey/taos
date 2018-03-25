package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Client struct {
	Terraform *Terraform
}

func (c Client) Version() (string, error) {
	outputCmd := c.terraformCmd([]string{
		"-v",
	})

	output, err := outputCmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve version.\nError: %s\nOutput: %s", err, output)
	}

	// The version returned could have many lines
	// We only care about the first line
	re := regexp.MustCompile(`\A.*`)
	matches := re.FindStringSubmatch(string(output))

	return matches[0], nil
}

func (c Client) terraformCmd(args []string) *exec.Cmd {

	defaultArgs := []string{
		"-no-color",
	}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("terraform %s %s", strings.Join(args, " "), strings.Join(defaultArgs, " ")))

	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		"CHECKPOINT_DISABLE=1",
		// "TF_LOG=DEBUG",
	}

	return cmd
}

// If client Config content is provided, then
//  create the necessary paths and files to
//  allow for terraform commands.
// Nothing is done if the Config content is empty
func (c Client) ClientInit() error {
	if len(c.Terraform.Config) <= 0 {
		return fmt.Errorf(ErrorMissingConfig)
	}

	// Create temporary working directory
	wd, err := ioutil.TempDir("", "terraform_client_workingdir")
	if err != nil {
		return err
	}
	c.Terraform.WorkingDir = wd

	// Set a name for the plan file
	c.Terraform.PlanFileName = "terraform.plan"

	// Set a name for the config file
	c.Terraform.ConfigFileName = "terraform.tf"

	// Set a name for the state file
	c.Terraform.StateFileName = "terraform.tfstate"

	// Write Config content to config file only if there is content to write
	if len(c.Terraform.Config) > 0 {
		configfile := filepath.Join(c.Terraform.WorkingDir, c.Terraform.ConfigFileName)
		err = ioutil.WriteFile(configfile, c.Terraform.Config, 0666)
		if err != nil {
			return err
		}
	}

	// Write State content to state file only if there is content to write
	if len(c.Terraform.State) > 0 {
		statefile := filepath.Join(c.Terraform.WorkingDir, c.Terraform.StateFileName)
		err = ioutil.WriteFile(statefile, c.Terraform.State, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c Client) ClientDestroy() error {
	_, err := os.Stat(c.Terraform.WorkingDir)
	if os.IsNotExist(err) {
		return errors.New(ErrorClientDestroyNoDir)
	}

	return os.RemoveAll(c.Terraform.WorkingDir)
}

func (c Client) Init() (string, error) {
	err := c.ClientInit()
	if err != nil {
		return "", err
	}

	initArgs := []string{
		"init",
		"-input=false",
		"-get=true",
		"-backend=false",
	}

	initArgs = append(initArgs, c.Terraform.WorkingDir)
	initCmd := c.terraformCmd(initArgs)

	// Perform terraform actions from within the temporary working directory
	initCmd.Dir = c.Terraform.WorkingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	initCmd.Stdout = &stdout
	initCmd.Stderr = &stderr
	err = initCmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr.String()))
	}

	re := regexp.MustCompile(`Terraform initialized in an empty directory!`)
	matches := re.FindStringSubmatch(stdout.String())

	// This is not the stdout or stderr of the terraform command.
	// 	Instead, this is expected to be a crafted error message because
	//  terraform doesn't error when no config is used, only invalid config.
	//  But we want to error when no config is used.
	if len(matches) > 0 {
		return "", fmt.Errorf("terraform init command failed.\nerror: %s", "Terraform initialized in an empty directory!")
	}

	return stdout.String(), nil
}

func (c Client) Plan() (string, error) {
	_, err := c.Init()
	if err != nil {
		return "", err
	}

	planArgs := []string{
		"plan",
		"-input=false", // do not prompt for inputs
	}

	c.Terraform.PlanFile = filepath.Join(c.Terraform.WorkingDir, c.Terraform.PlanFileName)

	planArgs = append(planArgs, fmt.Sprintf("-out=%s", c.Terraform.PlanFile))
	planArgs = append(planArgs, c.Terraform.WorkingDir)

	planCmd := c.terraformCmd(planArgs)
	planCmd.Dir = c.Terraform.WorkingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	planCmd.Stdout = &stdout
	planCmd.Stderr = &stderr
	err = planCmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr.String()))
	}

	return stdout.String(), nil
}

func (c Client) Apply() (string, error) {
	_, err := c.Plan()
	if err != nil {
		return "", err
	}

	applyArgs := []string{
		"apply",
		"-auto-approve",
		"-input=false", // do not prompt for inputs
	}

	applyArgs = append(applyArgs, c.Terraform.PlanFile)

	applyCmd := c.terraformCmd(applyArgs)
	applyCmd.Dir = c.Terraform.WorkingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	applyCmd.Stdout = &stdout
	applyCmd.Stderr = &stderr
	err = applyCmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr.String()))
	}

	return stdout.String(), nil
}

func (c Client) Destroy() (string, error) {
	_, err := c.Plan()
	if err != nil {
		return "", err
	}

	destroyArgs := []string{
		"destroy",
		"-force",
	}

	statefile := filepath.Join(c.Terraform.WorkingDir, c.Terraform.StateFileName)

	destroyArgs = append(destroyArgs, fmt.Sprintf("-state=%s", statefile))
	destroyArgs = append(destroyArgs, c.Terraform.WorkingDir)

	destroyCmd := c.terraformCmd(destroyArgs)
	destroyCmd.Dir = c.Terraform.WorkingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	destroyCmd.Stdout = &stdout
	destroyCmd.Stderr = &stderr
	err = destroyCmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr.String()))
	}

	return stdout.String(), nil
}
