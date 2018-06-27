package terraform

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type TerraformClient interface {
	TerraformCmd(args []string) *exec.Cmd
	ClientInit() error
	ClientDestroy() error
	Init() (string, error)
	Plan() (string, error)
	Apply() ([]byte, string, error)
	Destroy() ([]byte, string, error)
	Outputs() ([]byte, error)
	SetProject() string
}

type Client struct {
	Terraform     TerraformInfra
	Command       TerraformCommandRunner
	CommandConfig TerraformCommandConfig
}

type TerraformCommandConfig struct {
	Project     string
	Region      string
	Credentials string
}

func NewTerraformClient() *Client {
	return &Client{
		Terraform: TerraformInfra{},
		Command:   TerraformCommand{},
	}
}

func (client *Client) Config() []byte {
	return client.Terraform.Config
}

func (client *Client) SetConfig(config []byte) {
	client.Terraform.Config = config
}

func (client *Client) State() []byte {
	return client.Terraform.State
}

func (client *Client) SetState(state []byte) {
	client.Terraform.State = state
}

func (client *Client) SetProject(project string) {
	client.CommandConfig.Project = project
}

func (client *Client) Project() string {
	return client.CommandConfig.Project
}

func (client *Client) SetRegion(region string) {
	client.CommandConfig.Region = region
}

func (client *Client) Region() string {
	return client.CommandConfig.Region
}

func (client *Client) SetCredentials(credentials string) {
	client.CommandConfig.Credentials = credentials
}

func (client *Client) Credentials() string {
	return client.CommandConfig.Credentials
}

func (client *Client) Version() (string, error) {
	err, stdout, stderr := client.Command.Run("",
		[]string{
			"-v",
		}, client.Project(), client.Region(), client.Credentials())

	if err != nil {
		return "", fmt.Errorf("Failed to retrieve version.\nError: %s\nOutput: %s", err, stderr)
	}

	// The version returned could have many lines
	// We only care about the first line
	re := regexp.MustCompile(`\A.*`)
	matches := re.FindStringSubmatch(stdout)

	return matches[0], nil
}

// If client Config content is provided, then
//  create the necessary paths and files to
//  allow for terraform commands.
// Nothing is done if the Config content is empty
func (client *Client) ClientInit() error {
	logger := log.WithFields(log.Fields{"package": "terraform", "event": "client_init"})

	if len(client.Terraform.Config) <= 0 {
		logger.Error(ErrorMissingConfig)
		return fmt.Errorf(ErrorMissingConfig)
	}

	if len(client.Project()) == 0 {
		logger.Error(ErrorMissingProject)
		err := errors.New(ErrorMissingProject)
		return err
	}

	if len(client.Region()) == 0 {
		logger.Error(ErrorMissingRegion)
		err := errors.New(ErrorMissingRegion)
		return err
	}

	if len(client.Credentials()) == 0 {
		logger.Error(ErrorMissingCredentials)
		err := errors.New(ErrorMissingCredentials)
		return err
	}

	// Create temporary working directory
	wd, err := ioutil.TempDir("", "terraform_client_workingdir")
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	client.Terraform.WorkingDir = wd

	// Set a name for the plan file
	client.Terraform.PlanFileName = "terraform.plan"

	// Set a name for the config file
	client.Terraform.ConfigFileName = "terraform.tf"

	// Set a name for the state file
	client.Terraform.StateFileName = "terraform.tfstate"

	// Write Config content to config file only if there is content to write
	if len(client.Terraform.Config) > 0 {
		configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
		err = ioutil.WriteFile(configfile, client.Terraform.Config, 0666)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	// Write State content to state file only if there is content to write
	if len(client.Terraform.State) > 0 {
		statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
		err = ioutil.WriteFile(statefile, client.Terraform.State, 0666)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	return nil
}

func (client *Client) ClientDestroy() error {
	_, err := os.Stat(client.Terraform.WorkingDir)
	if os.IsNotExist(err) {
		return errors.New(ErrorClientDestroyNoDir)
	}

	return os.RemoveAll(client.Terraform.WorkingDir)
}

func (client *Client) Init() (string, error) {
	err := client.ClientInit()
	if err != nil {
		return "", err
	}

	initArgs := []string{
		"init",
		"-input=false",
		"-get=true",
		"-backend=false",
	}

	initArgs = append(initArgs, client.Terraform.WorkingDir)
	err, stdout, stderr := client.Command.Run(
		client.Terraform.WorkingDir,
		initArgs,
		client.Project(),
		client.Region(),
		client.Credentials())

	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	re := regexp.MustCompile(`Terraform initialized in an empty directory!`)
	matches := re.FindStringSubmatch(stdout)

	// This is not the stdout or stderr of the terraform command.
	// 	Instead, this is expected to be a crafted error message because
	//  terraform doesn't error when no config is used, only invalid config.
	//  But we want to error when no config is used.
	if len(matches) > 0 {
		return "", fmt.Errorf("terraform init command failed.\nerror: %s", "Terraform initialized in an empty directory!")
	}

	return stdout, nil
}

func (client *Client) Plan() (string, error) {
	_, err := client.Init()
	if err != nil {
		return "", err
	}

	planArgs := []string{
		"plan",
		"-input=false", // do not prompt for inputs
	}

	client.Terraform.PlanFile = filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)

	planArgs = append(planArgs, fmt.Sprintf("-out=%s", client.Terraform.PlanFile))
	planArgs = append(planArgs, client.Terraform.WorkingDir)

	err, stdout, stderr := client.Command.Run(client.Terraform.WorkingDir, planArgs,
		client.Project(),
		client.Region(),
		client.Credentials())

	if err != nil {
		return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	return stdout, nil
}

func (client *Client) Apply() ([]byte, string, error) {
	_, err := client.Plan()
	if err != nil {
		return nil, "", err
	}

	applyArgs := []string{
		"apply",
		"-auto-approve",
		"-input=false", // do not prompt for inputs
	}

	applyArgs = append(applyArgs, client.Terraform.PlanFile)

	err, stdout, stderr := client.Command.Run(client.Terraform.WorkingDir, applyArgs,
		client.Project(),
		client.Region(),
		client.Credentials())

	if err != nil {
		return nil, "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	// Read the state file in order to return its contents
	statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
	state, err := ioutil.ReadFile(statefile)
	if err != nil {
		return nil, "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	return state, strings.TrimSpace(stdout), nil
}

func (client *Client) Destroy() ([]byte, string, error) {
	_, err := client.Plan()
	if err != nil {
		return nil, "", err
	}

	destroyArgs := []string{
		"destroy",
		"-force",
	}

	statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)

	destroyArgs = append(destroyArgs, fmt.Sprintf("-state=%s", statefile))
	destroyArgs = append(destroyArgs, client.Terraform.WorkingDir)

	err, stdout, stderr := client.Command.Run(client.Terraform.WorkingDir, destroyArgs,
		client.Project(),
		client.Region(),
		client.Credentials())

	if err != nil {
		return nil, "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	// Read the state file in order to return its contents
	state, err := ioutil.ReadFile(statefile)
	if err != nil {
		return nil, "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
	}

	return state, stdout, nil
}

func (client *Client) Outputs() (string, error) {
	logger := log.WithFields(log.Fields{"package": "terraform", "event": "terraform_outputs"})

	_, err := client.Init()
	if err != nil {
		return "", err
	}

	outputsArgs := []string{
		"output",
		"-json",
	}

	statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)

	outputsArgs = append(outputsArgs, fmt.Sprintf("-state=%s", statefile))

	err, stdout, stderr := client.Command.Run(client.Terraform.WorkingDir, outputsArgs,
		client.Project(),
		client.Region(),
		client.Credentials())

	if err != nil {

		re := regexp.MustCompile(`The state file either has no outputs defined`)
		matches := re.FindStringSubmatch(stderr)

		// No outputs being defined in the Terraform config is not an error, it's
		//  more of a warning situation because lacking outputs makes the cluster
		//  fairly useless (outputs are used to connect).
		// Therefore doesn't return an error, just return empty outputs
		if len(matches) > 0 {
			logger.Warn("no outputs defined in Terraform config")
			return "", nil
		} else {
			return "", errors.New(fmt.Sprint(fmt.Sprint(err) + ": " + stderr))
		}
	}

	return stdout, nil
}
