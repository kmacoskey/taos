package terraform

type TerraformInfra struct {
	Config         []byte `json:"terraform_configuration"`
	State          []byte `json:"terraform_state"`
	WorkingDir     string
	PlanFileName   string
	ConfigFileName string
	StateFileName  string
	PlanFile       string
}

const (
	ErrorMissingProject     = "No project specified for running terraform client actions"
	ErrorMissingRegion      = "No region specified for running terraform client actions"
	ErrorMissingCredentials = "No credentials specified for running terraform client actions"
	ErrorMissingConfig      = "refusing to create client without terraform configuration content"
	ErrorClientDestroyNoDir = "Failed to destroy Client: Working directory does not exist."
	ErrorInvalidConfig      = "The Terraform configuration must be valid before initialization"
	ErrorMissingOutputs     = "The state file either has no outputs defined, or all the defined\noutputs are empty."
	ErrorBadState           = "Error refreshing state:"
	InitSuccess             = "Terraform has been successfully initialized!"
	PlanNoChangesSuccess    = "No changes. Infrastructure is up-to-date."
	ApplySuccess            = "Apply complete! Resources:"
	DestroySuccess          = "Destroy complete! Resources: 0 destroyed."
)
