package handlers

type ClusterRequest struct {
	TerraformConfig string `json:"config"`
	Timeout         string `json:"timeout"`
	Project         string `json:"project"`
	Region          string `json:"region"`
}

type ClusterResponse struct {
	RequestId string              `json:"request_id"`
	Status    string              `json:"status"`
	Data      ClusterResponseData `json:"data"`
}

type ClusterResponseData struct {
	Type       string `json:"type"`
	Attributes ClusterResponseAttributes
}

type ClustersResponse struct {
	RequestId string               `json:"request_id"`
	Status    string               `json:"status"`
	Data      ClustersResponseData `json:"data"`
}

type ClustersResponseData struct {
	Type       string `json:"type"`
	Attributes []ClusterResponseAttributes
}

type ClusterResponseAttributes struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	TerraformOutputs map[string]TerraformOutput
}

type TerraformOutput struct {
	Sensitive bool   `json:"sensitive"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type ErrorResponse struct {
	RequestId string            `json:"request_id"`
	Status    string            `json:"status"`
	Data      ErrorResponseData `json:"data"`
}

type ErrorResponseData struct {
	Type       string `json:"type"`
	Attributes *ErrorResponseAttributes
}

type ErrorResponseAttributes struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}
