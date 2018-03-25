package terraform_test

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/kmacoskey/taos/terraform"
	// "os"
)

var _ = Describe("Client", func() {

	var (
		client                 *Client
		terraform              *Terraform
		validTerraformConfig   []byte
		invalidTerraformConfig []byte
		emptyTerraformConfig   []byte
		validTerraformState    []byte
		invalidTerraformState  []byte
		emptyTerraformState    []byte
		state                  []byte
		output                 string
		err                    error
	)

	BeforeEach(func() {
		terraform = &Terraform{}
		client = &Client{
			Terraform: terraform,
		}

		validTerraformConfig = []byte(`{"provider":{"google":{}}}`)
		invalidTerraformConfig = []byte(`NotTheJsonYouAreLookingFor`)
		emptyTerraformConfig = []byte(``)

		validTerraformState = []byte(`{"version":3,"terraform_version":"0.11.3","serial":2,"lineage":"a1f48d83-76dc-48d9-9181-f274799603ef","modules":[{"path":["root"],"outputs":{},"resources":{},"depends_on":[]}]}`)
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
				output, err = client.Init()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should initialize successfully", func() {
				Expect(output).To(ContainSubstring("Terraform has been successfully initialized!"))
			})
		})

		Context("With invalid Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				output, err = client.Init()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
			})
			It("Should return the stderr of the failed command", func() {
				Expect(err.Error()).To(ContainSubstring(ErrorInvalidConfig))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				output, err = client.Init()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				output, err = client.Plan()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned output matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(output).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				output, err = client.Plan()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return command output", func() {
				Expect(output).To(BeEmpty())
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
				output, err = client.Plan()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				output, err = client.Plan()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned output matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(output).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				output, err = client.Plan()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				output, err = client.Plan()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should plan successfully", func() {
				// The returned output matches the conditions that no new infrastructure is being created
				//  based on validTerraformConfig not being meant to actualy create infrastructure.
				// This is still a successful terraform plan
				Expect(output).To(ContainSubstring("No changes. Infrastructure is up-to-date."))
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
				state, output, err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(output).To(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				state, output, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(output).To(ContainSubstring("Apply complete! Resources: 0 added, 0 changed, 0 destroyed."))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
				state, output, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should apply successfully", func() {
				Expect(output).To(ContainSubstring("Destroy complete! Resources: 0 destroyed."))
			})
		})

		Context("With no Terraform config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				state, output, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return the command output", func() {
				Expect(output).NotTo(BeNil())
			})
			It("Should return the Terraform state", func() {
				Expect(state).NotTo(BeNil())
			})
			It("Should destroy successfully", func() {
				Expect(output).To(ContainSubstring("Destroy complete! Resources: 0 destroyed."))
			})
		})

		Context("With invalid Terraform config and valid Terraform state", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
				state, output, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				state, output, err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not return any command output", func() {
				Expect(output).To(BeEmpty())
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
				output, err := sys_terraform.Output()
				Expect(err).NotTo(HaveOccurred())

				// Get the first line of output which is expected to contain the version
				re := regexp.MustCompile(`\A.*`)
				output_string := string(output)
				matches := re.FindStringSubmatch(output_string)
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
