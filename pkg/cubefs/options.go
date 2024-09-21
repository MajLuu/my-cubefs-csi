package cubefs

// Mode is the operating mode of the CSI driver.
type Mode string

const (
	// ControllerMode is the mode that only starts the controller service.
	ControllerMode Mode = "controller"
	// NodeMode is the mode that only starts the node service.
	NodeMode Mode = "node"
	// AllMode is the mode that only starts both the controller and the node service.
	AllMode Mode = "all"
)

type Options struct {
	Mode Mode

	// Kubeconfig is an absolute path to a kubeconfig file.
	// If empty, the in-cluster config will be loaded.
	Kubeconfig string

	//Endpoint is the endpoint for the CSI driver server
	Endpoint string
	// HttpEndpoint is the TCP network address where the HTTP server for metrics will listen
	HttpEndpoint string
}
