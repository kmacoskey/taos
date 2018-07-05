package terraform_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	. "github.com/kmacoskey/taos/terraform"
)

var _ = Describe("Client", func() {

	var (
		client                        *Client
		validTerraformConfig          []byte
		invalidTerraformConfig        []byte
		emptyTerraformConfig          []byte
		validTerraformState           []byte
		invalidTerraformState         []byte
		emptyTerraformState           []byte
		validNoOutputsTerraformConfig []byte
		validNoOutputsTerraformState  []byte
		validTerraformOutputs         string
		validProject                  string
		emptyProject                  string
		validRegion                   string
		emptyRegion                   string
		validCredentials              string
		emptyCredentials              string
		state                         []byte
		stdout                        string
		outputs                       string
		err                           error
		version                       string
	)

	BeforeEach(func() {
		log.SetLevel(log.FatalLevel)

		client = new(Client)

		validTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}},"output":{"foo":{"value":"bar"}}}`)
		validNoOutputsTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}}}`)
		invalidTerraformConfig = []byte(`NotTheJsonYouAreLookingFor`)
		emptyTerraformConfig = []byte(``)

		validTerraformState = []byte(`{"version":3,"terraform_version":"0.11.3","serial":2,"lineage":"26655d4c-852a-41e4-b6f1-7b31ff2b2981","modules":[{"path":["root"],"outputs":{"foo":{"sensitive":false,"type":"string","value":"bar"}},"resources":{},"depends_on":[]}]}`)
		validNoOutputsTerraformState = []byte(`{"version":3,"terraform_version":"0.11.3","serial":1,"lineage":"68c63875-913c-4d6a-9f87-c006b9d030a4","modules":[{"path":["root"],"outputs":{},"resources":{},"depends_on":[]}]}`)
		invalidTerraformState = []byte(`NotTheJsonYouAreLookingFor`)
		emptyTerraformState = []byte(``)

		validProject = "gcp-project-foo"
		emptyProject = ""
		validRegion = "gcp-region-foo"
		emptyRegion = ""
		validCredentials = "gcp-credentials-foo"
		emptyCredentials = ""

		validTerraformOutputs = "{\"bar\":{\"sensitive\":false,\"type\":\"string\",\"value\":\"foo\" }"
	})

	AfterEach(func() {
		client.ClientDestroy()
	})

	// ======================================================================
	//       _ _            _
	//   ___| (_) ___ _ __ | |_
	//  / __| | |/ _ \ '_ \| __|
	// | (__| | |  __/ | | | |_
	//  \___|_|_|\___|_| |_|\__|
	//
	// ======================================================================

	Describe("Initializing the Terraform Client", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(validTerraformState)
				err = client.ClientInit()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a temporary working directory", func() {
				Expect(client.Terraform.WorkingDir).To(BeADirectory())
			})
			It("Should set a plan file name", func() {
				Expect(client.Terraform.PlanFileName).To(Equal("terraform.plan"))
			})
			It("Should set a config file name", func() {
				Expect(client.Terraform.ConfigFileName).To(Equal("terraform.tf"))
			})
			It("Should set a state file name", func() {
				Expect(client.Terraform.StateFileName).To(Equal("terraform.tfstate"))
			})
			It("Should create a config file in the working directory", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should make the Config content available in the config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				config, readerr := ioutil.ReadFile(configfile)
				Expect(readerr).NotTo(HaveOccurred())
				Expect(config).To(Equal(validTerraformConfig))
			})
			It("Should create a state file in the working directory", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).Should(BeARegularFile())
			})
			It("Should make the State content available in the state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				state, readerr := ioutil.ReadFile(statefile)
				Expect(readerr).NotTo(HaveOccurred())
				Expect(state).To(Equal(validTerraformState))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(emptyTerraformConfig)
				err = client.ClientInit()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not create a temporary working directory", func() {
				Expect(client.Terraform.WorkingDir).NotTo(BeADirectory())
			})
			It("Should not set a plan file name", func() {
				Expect(client.Terraform.PlanFileName).NotTo(Equal("terraform.plan"))
			})
			It("Should not set a config file name", func() {
				Expect(client.Terraform.ConfigFileName).NotTo(Equal("terraform.tf"))
			})
			It("Should not create a config file in the working directory", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).NotTo(BeARegularFile())
			})
		})

		Context("With no State content", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(emptyTerraformState)
				err = client.ClientInit()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should not create a state file in the working directory", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).ShouldNot(BeARegularFile())
			})
		})

		Context("With no Project specified", func() {
			BeforeEach(func() {
				client.SetProject(emptyProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				err = client.ClientInit()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With no Region specified", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(emptyRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				err = client.ClientInit()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With no Credentials specified", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(emptyCredentials)
				client.SetConfig(validTerraformConfig)
				err = client.ClientInit()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

	})

	Describe("Destroying the Terraform Client", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				init_err := client.ClientInit()
				Expect(init_err).NotTo(HaveOccurred())

				err = client.ClientDestroy()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should remove the temporary working directory", func() {
				Expect(client.Terraform.WorkingDir).NotTo(BeADirectory())
			})
		})

		Context("When there is no working directory", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				init_err := client.ClientInit()
				client.Terraform.WorkingDir = "/tmp/this-should-not-be-an-existing-directory"
				Expect(init_err).NotTo(HaveOccurred())

				err = client.ClientDestroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorClientDestroyNoDir))
			})
		})

	})

	// ======================================================================
	//  _       _ _
	// (_)_ __ (_) |_
	// | | '_ \| | __|
	// | | | | | | |_
	// |_|_| |_|_|\__|
	//
	// ======================================================================

	Describe("Running Terraform init", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Init()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should initialize successfully", func() {
				Expect(stdout).To(ContainSubstring(InitSuccess))
			})
		})

		Context("With invalid Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.Command = new(FailingTerraformCommand)
				stdout, err = client.Init()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the stderr of the failed command", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(emptyTerraformConfig)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Init()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorMissingConfig))
			})
		})

	})

	// ======================================================================
	//        _
	//  _ __ | | __ _ _ __
	// | '_ \| |/ _` | '_ \
	// | |_) | | (_| | | | |
	// | .__/|_|\__,_|_| |_|
	// |_|
	//
	// ======================================================================

	Describe("Running Terraform Plan", func() {

		Context("When everything does ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned stdout matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(stdout).To(ContainSubstring(PlanNoChangesSuccess))
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
			})
		})

		Context("With invalid Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.Command = new(FailingTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the stderr of the failed command", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(emptyTerraformConfig)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorMissingConfig))
			})
			It("Should not have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).ShouldNot(BeARegularFile())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("With valid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned stdout matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(stdout).To(ContainSubstring(PlanNoChangesSuccess))
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).Should(BeARegularFile())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
			})
		})

		Context("With valid Terraform config and invalid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(invalidTerraformState)
				client.Command = new(FailingTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorBadState))
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).Should(BeARegularFile())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("With valid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(emptyTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				stdout, err = client.Plan(false)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned stdout matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(stdout).To(ContainSubstring(PlanNoChangesSuccess))
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should not have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).ShouldNot(BeARegularFile())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
			})
		})

	})

	// ======================================================================
	//                    _
	//   __ _ _ __  _ __ | |_   _
	//  / _` | '_ \| '_ \| | | | |
	// | (_| | |_) | |_) | | |_| |
	//  \__,_| .__/| .__/|_|\__, |
	//       |_|   |_|      |___/
	//
	// ======================================================================

	Describe("Running Terraform Apply", func() {

		Context("When everything does ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(stdout).To(ContainSubstring(ApplySuccess))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.Terraform.Config = emptyTerraformConfig
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorMissingConfig))
			})
		})

		Context("With valid Terraform config and invalid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(invalidTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorBadState))
			})
		})

		Context("With valid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(emptyTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(stdout).To(ContainSubstring(ApplySuccess))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With invalid Terraform config and invalid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(invalidTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With invalid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(emptyTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

	})

	// ======================================================================
	//      _           _
	//   __| | ___  ___| |_ _ __ ___  _   _
	//  / _` |/ _ \/ __| __| '__/ _ \| | | |
	// | (_| |  __/\__ \ |_| | | (_) | |_| |
	//  \__,_|\___||___/\__|_|  \___/ \__, |
	//                                |___/
	//
	// ======================================================================

	Describe("Running Terraform Destroy", func() {

		Context("When everything does ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(stdout).To(ContainSubstring(DestroySuccess))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(emptyTerraformConfig)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorMissingConfig))
			})
		})

		Context("With valid Terraform config and invalid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(invalidTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorBadState))
			})
		})

		Context("With valid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(emptyTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should destroy successfully", func() {
				Expect(stdout).To(ContainSubstring(DestroySuccess))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With invalid Terraform config and invalid Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(invalidTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With invalid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(invalidTerraformConfig)
				client.SetState(emptyTerraformState)
				client.Command = new(FailingTerraformCommand)
				state, stdout, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should not return any Terraform state", func() {
				Expect(state).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

	})

	// ======================================================================
	//              _               _
	//   ___  _   _| |_ _ __  _   _| |_ ___
	//  / _ \| | | | __| '_ \| | | | __/ __|
	// | (_) | |_| | |_| |_) | |_| | |_\__ \
	//  \___/ \__,_|\__| .__/ \__,_|\__|___/
	//                 |_|
	//
	// ======================================================================

	Describe("Running Terraform Outputs", func() {

		Context("When everything does ok", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validTerraformConfig)
				client.SetState(validTerraformState)
				client.Command = new(SuccessfulTerraformCommand)
				outputs, err = client.Outputs()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the Terraform outputs", func() {
				Expect(outputs).NotTo(BeNil())
			})
			It("Should return the expected outputs", func() {
				Expect(outputs).To(Equal(validTerraformOutputs))
			})
		})

		Context("When no outputs are defined in the Terraform state", func() {
			BeforeEach(func() {
				client.SetProject(validProject)
				client.SetRegion(validRegion)
				client.SetCredentials(validCredentials)
				client.SetConfig(validNoOutputsTerraformConfig)
				client.SetState(validNoOutputsTerraformState)
				client.Command = new(FailingTerraformCommand)
				outputs, err = client.Outputs()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any Terraform outputs", func() {
				Expect(outputs).To(BeEmpty())
			})
		})

	})

	// ======================================================================
	//                     _
	// __   _____ _ __ ___(_) ___  _ __
	// \ \ / / _ \ '__/ __| |/ _ \| '_ \
	//  \ V /  __/ |  \__ \ | (_) | | | |
	//   \_/ \___|_|  |___/_|\___/|_| |_|
	//
	// ======================================================================

	Describe("Requesting the Terraform Version", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				client.Command = new(SuccessfulTerraformCommand)
				version, err = client.Version()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the expected terraform version", func() {
				Expect(client.Version()).To(ContainSubstring("Terraform v"))
			})
		})

	})

})

type SuccessfulTerraformCommand struct {
	Project     string
	Region      string
	Credentials string
}

func (tc *SuccessfulTerraformCommand) Run(directory string, args []string, project string, region string, credentials string) (error, string, string) {

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// args[0] is expected to be the terraform subcommand (e.g. init, apply, destroy, etc.)
	switch args[0] {
	case "init":
		stdout.WriteString(InitSuccess)
	case "plan":
		stdout.WriteString(PlanNoChangesSuccess)
		// Create empty plan file to coincide with successful terraform plan
		plan_file := filepath.Join(directory, "terraform.plan")
		err := ioutil.WriteFile(plan_file, []byte(`{}`), 0666)
		if err != nil {
			panic(fmt.Sprintf("Failed to write to '%s'", plan_file))
		}
	case "apply":
		stdout.WriteString(ApplySuccess)
		// Create empty tfstate file to coincide with successful terraform apply
		state_files := filepath.Join(directory, "terraform.tfstate")
		err := ioutil.WriteFile(state_files, []byte(`{}`), 0666)
		if err != nil {
			panic(fmt.Sprintf("Failed to write to '%s'", state_files))
		}
	case "destroy":
		stdout.WriteString(DestroySuccess)
		// Create empty tfstate file to coincide with successful terraform destroy
		state_files := filepath.Join(directory, "terraform.tfstate")
		err := ioutil.WriteFile(state_files, []byte(`{}`), 0666)
		if err != nil {
			panic(fmt.Sprintf("Failed to write to '%s'", state_files))
		}
	case "output":
		stdout.WriteString(`{"bar":{"sensitive":false,"type":"string","value":"foo" }`)
	case "-v":
		stdout.WriteString("Terraform v0.11.5")
	default:
		stderr.WriteString("Unknown Subcommand")
	}

	return nil, stdout.String(), stderr.String()
}

type FailingTerraformCommand struct {
	Project     string
	Region      string
	Credentials string
}

func (tc *FailingTerraformCommand) Run(directory string, args []string, project string, region string, credentials string) (error, string, string) {

	err := new(exec.ExitError)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// args[0] is expected to be the terraform subcommand (e.g. init, apply, destroy, etc.)
	switch args[0] {
	case "init":
		stderr.WriteString(ErrorInvalidConfig + ErrorBadState)
	case "plan":
		stderr.WriteString("foo")
	case "apply":
		stderr.WriteString("foo")
	case "destroy":
		stderr.WriteString("foo")
	case "output":
		stderr.WriteString("foo")
	}

	return err, stdout.String(), stderr.String()
}
