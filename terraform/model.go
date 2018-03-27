package terraform

type Terraform struct {
	Config         []byte `json:"terraform_configuration"`
	State          []byte `json:"terraform_state"`
	WorkingDir     string
	PlanFileName   string
	ConfigFileName string
	StateFileName  string
	PlanFile       string
}

const (
	ErrorMissingConfig      = "refusing to create client without terraform configuration content"
	ErrorClientDestroyNoDir = "Failed to destroy Client: Working directory does not exist."
	ErrorInvalidConfig      = "The Terraform configuration must be valid before initialization"
	ErrorMissingOutputs     = "The state file either has no outputs defined, or all the defined\noutputs are empty."
)
