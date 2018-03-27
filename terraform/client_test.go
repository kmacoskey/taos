package terraform_test

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	. "github.com/kmacoskey/taos/terraform"
)

var _ = Describe("Client", func() {

	var (
		client                        *Client
		terraform                     *Terraform
		validTerraformConfig          []byte
		invalidTerraformConfig        []byte
		emptyTerraformConfig          []byte
		validTerraformState           []byte
		invalidTerraformState         []byte
		emptyTerraformState           []byte
		validNoOutputsTerraformConfig []byte
		validNoOutputsTerraformState  []byte
		state                         []byte
		stdout                        string
		outputs                       []byte
		err                           error
	)

	BeforeEach(func() {
		log.SetLevel(log.FatalLevel)

		terraform = &Terraform{}
		client = &Client{
			Terraform: terraform,
		}

		validTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}},"output":{"foo":{"value":"bar"}}}`)
		validNoOutputsTerraformConfig = []byte(`{"provider":{"google":{"project":"data-gp-toolsmiths","region":"us-central1"}}}`)
		invalidTerraformConfig = []byte(`NotTheJsonYouAreLookingFor`)
		emptyTerraformConfig = []byte(``)

		validTerraformState = []byte(`{"version":3,"terraform_version":"0.11.3","serial":2,"lineage":"26655d4c-852a-41e4-b6f1-7b31ff2b2981","modules":[{"path":["root"],"outputs":{"foo":{"sensitive":false,"type":"string","value":"bar"}},"resources":{},"depends_on":[]}]}`)
		validNoOutputsTerraformState = []byte(`{"version":3,"terraform_version":"0.11.3","serial":1,"lineage":"68c63875-913c-4d6a-9f87-c006b9d030a4","modules":[{"path":["root"],"outputs":{},"resources":{},"depends_on":[]}]}`)
		invalidTerraformState = []byte(`NotTheJsonYouAreLookingFor`)
		emptyTerraformState = []byte(``)
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
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
				client.Terraform.Config = emptyTerraformConfig
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
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

	})

	Describe("Destroying the Terraform Client", func() {

		Context("When everything goes ok", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
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
				client.Terraform.Config = invalidTerraformConfig
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
				client.Terraform.Config = validTerraformConfig
				stdout, err = client.Init()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command stdout", func() {
				Expect(stdout).NotTo(BeNil())
			})
			It("Should initialize successfully", func() {
				Expect(stdout).To(ContainSubstring("Terraform has been successfully initialized!"))
			})
		})

		Context("With invalid Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
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
				client.Terraform.Config = emptyTerraformConfig
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
				client.Terraform.Config = validTerraformConfig
				stdout, err = client.Plan()
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
				Expect(stdout).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				client.Terraform.Config = invalidTerraformConfig
				stdout, err = client.Plan()
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
				client.Terraform.Config = emptyTerraformConfig
				stdout, err = client.Plan()
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				stdout, err = client.Plan()
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
				Expect(stdout).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
				stdout, err = client.Plan()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command stdout", func() {
				Expect(stdout).To(BeEmpty())
			})
			It("Should return the expected error message", func() {
				Expect(err.Error()).To(ContainSubstring("Error refreshing state:"))
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
				stdout, err = client.Plan()
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
				Expect(stdout).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
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
				Expect(stdout).To(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
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
				Expect(err.Error()).To(ContainSubstring("Error refreshing state:"))
			})
		})

		Context("With valid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
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
				Expect(stdout).To(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
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
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = invalidTerraformState
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
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = emptyTerraformState
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
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
				Expect(stdout).To(ContainSubstring("Destroy complete! Resources: 0 destroyed."))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
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
				Expect(err.Error()).To(ContainSubstring("Error refreshing state:"))
			})
		})

		Context("With valid Terraform config and no Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
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
				Expect(stdout).To(ContainSubstring("Destroy complete! Resources: 0 destroyed."))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
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
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = invalidTerraformState
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
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = emptyTerraformState
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
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				outputs, err = client.Outputs()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the Terraform outputs", func() {
				Expect(outputs).NotTo(BeNil())
			})
			It("Should return the expected outputs", func() {
				Expect(outputs).To(ContainSubstring("foo"))
				Expect(outputs).To(ContainSubstring("bar"))
			})
		})

		Context("When no outputs are defined in the Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = validNoOutputsTerraformConfig
				client.Terraform.State = validNoOutputsTerraformState
				outputs, err = client.Outputs()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
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
			var (
				sys_version string
				version     string
				err         error
			)
			BeforeEach(func() {
				// Get the reported version of the system terraform
				sys_terraform := exec.Command("/usr/local/bin/terraform", "-version")
				stdout, err := sys_terraform.Output()
				Expect(err).NotTo(HaveOccurred())

				// Get the first line of stdout which is expected to contain the version
				re := regexp.MustCompile(`\A.*`)
				stdout_string := string(stdout)
				matches := re.FindStringSubmatch(stdout_string)
				Expect(matches).ShouldNot(BeEmpty())

				sys_version = matches[0]
				version, err = client.Version()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the expected terraform version", func() {
				Expect(client.Version()).To(Equal(sys_version))
			})
		})

	})

})
