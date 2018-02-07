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

	Describe("Using the Terraform Client", func() {

		// ======================================================================
		//       _ _            _
		//   ___| (_) ___ _ __ | |_
		//  / __| | |/ _ \ '_ \| __|
		// | (__| | |  __/ | | | |_
		//  \___|_|_|\___|_| |_|\__|
		//
		// ======================================================================

		Context("Initialzing the Client", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
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
		})

		Context("Destroy the Client with a working directory", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				err = client.ClientInit()
				Expect(err).NotTo(HaveOccurred())

				client.ClientDestroy()
			})
			It("Should remote the temporary working directory", func() {
				Expect(client.Terraform.WorkingDir).NotTo(BeADirectory())
			})
		})

		Context("Initializing the Client with Config content", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				err = client.ClientInit()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
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
		})

		Context("Initializing the Client with no Config content", func() {
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

		Context("Initializing the Client with State content", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.ClientInit()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
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

		Context("Initializing the Client with no State content", func() {
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

		// ======================================================================
		//  _       _ _
		// (_)_ __ (_) |_
		// | | '_ \| | __|
		// | | | | | | |_
		// |_|_| |_|_|\__|
		//
		// ======================================================================

		Context("Running Terraform Init with valid terraform Config", func() {
			It("Should not error", func() {
				client.Terraform.Config = validTerraformConfig
				err = client.Init()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Running Terraform Init with invalid terraform Config", func() {
			It("Should not error", func() {
				client.Terraform.Config = invalidTerraformConfig
				err = client.Init()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Terraform Init with no terraform Config", func() {
			It("Should error", func() {
				client.Terraform.Config = emptyTerraformConfig
				err = client.Init()
				Expect(err).To(HaveOccurred())
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

		Context("Running Terraform Plan with valid Config", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				err = client.Plan()
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
			})
		})

		Context("Running Terraform Plan with invalid Config", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				err = client.Plan()
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("Running Terraform Plan with no Config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				err = client.Plan()
			})
			It("Should not have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).ShouldNot(BeARegularFile())
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("Running Terraform Plan with valid Config and valid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.Plan()
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).Should(BeARegularFile())
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
			})
		})

		Context("Running Terraform Plan with valid Config and invalid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
				err = client.Plan()
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).Should(BeARegularFile())
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
			It("Should not create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).ShouldNot(BeARegularFile())
			})
		})

		Context("Running Terraform Plan with valid Config and no State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
				err = client.Plan()
			})
			It("Should have a config file", func() {
				configfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.ConfigFileName)
				Expect(configfile).Should(BeARegularFile())
			})
			It("Should not have a state file", func() {
				statefile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.StateFileName)
				Expect(statefile).ShouldNot(BeARegularFile())
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a plan file", func() {
				planfile := filepath.Join(client.Terraform.WorkingDir, client.Terraform.PlanFileName)
				Expect(planfile).Should(BeARegularFile())
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

		Context("Running Apply with no Config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Apply with valid Config and valid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Running Apply with valid Config and invalid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
				err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Apply with valid Config and no State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
				err = client.Apply()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Running Apply with invalid Config and valid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Apply with invalid Config and invalid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = invalidTerraformState
				err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Apply with invalid Config and no State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = emptyTerraformState
				err = client.Apply()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
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

		Context("Running Destroy with no Config", func() {
			BeforeEach(func() {
				client.Terraform.Config = emptyTerraformConfig
				err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Destroy with valid Config and valid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Running Destroy with valid Config and invalid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = invalidTerraformState
				err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Destroy with valid Config and no State", func() {
			BeforeEach(func() {
				client.Terraform.Config = validTerraformConfig
				client.Terraform.State = emptyTerraformState
				err = client.Destroy()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Running Destroy with invalid Config and valid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = validTerraformState
				err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Destroy with invalid Config and invalid State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = invalidTerraformState
				err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Running Destroy with invalid Config and no State", func() {
			BeforeEach(func() {
				client.Terraform.Config = invalidTerraformConfig
				client.Terraform.State = emptyTerraformState
				err = client.Destroy()
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
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

		Context("Asking for the version", func() {
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
			It("Should return the expected terraform version", func() {
				Expect(client.Version()).To(Equal(sys_version))
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})
})
