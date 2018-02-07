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
