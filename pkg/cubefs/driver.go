package cubefs

import (
	"context"
	"fmt"
	"net"

	"github.com/majlu/my-cubefs-csi/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	DriverName = "mycubefs.csi.cubefs.com"
)

type CSIDriver struct {
	*IdentityService
	cs      *ControllerService
	ns      *NodeService
	gsrv    *grpc.Server
	options *Options
}

func NewCSIDriver(name, nodeId, version string, opts *Options) (*CSIDriver, error) {
	if name == "" {
		klog.Errorf("Driver name missing")
		return nil, fmt.Errorf("Driver name missing")
	}

	if nodeId == "" {
		klog.Errorf("NodeID missing")
		return nil, fmt.Errorf("NodeID missing")
	}

	// TODO version format and validation
	if len(version) == 0 {
		klog.Errorf("Version argument missing")
		return nil, fmt.Errorf("Version argument missing")
	}

	klog.InfoS("Driver Information", "Driver", name, "Version", version, "nodeId", nodeId)
	if err := ValidateDriverOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid driver options: %w", err)
	}

	driver := &CSIDriver{
		IdentityService: NewIdentityService(name, version),
		options:         opts,
	}

	k8sClient, err := driver.NewK8SClientSet()
	if err != nil {
		return nil, err
	}

	switch opts.Mode {
	case ControllerMode:
		driver.cs = NewControllerService(k8sClient)
	case NodeMode:
		driver.ns = NewNodeService(nodeId, k8sClient)
	case AllMode:
		driver.cs = NewControllerService(k8sClient)
		driver.ns = NewNodeService(nodeId, k8sClient)
	}

	return driver, nil
}

func (d *CSIDriver) Run() error {
	scheme, addr, err := util.ParseEndpoint(d.options.Endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return err
	}

	// register GRPC log error
	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			klog.ErrorS(err, "GRPC error")
		}
		return resp, err
	}
	// intercept GRPC error and logging in driver
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}

	d.gsrv = grpc.NewServer(opts...)
	csi.RegisterIdentityServer(d.gsrv, d)

	switch d.options.Mode {
	case ControllerMode:
		csi.RegisterControllerServer(d.gsrv, d.cs)
	case NodeMode:
		csi.RegisterNodeServer(d.gsrv, d.ns)
	case AllMode:
		csi.RegisterControllerServer(d.gsrv, d.cs)
		csi.RegisterNodeServer(d.gsrv, d.ns)
	default:
		return fmt.Errorf("unknown mode: %s", d.options.Mode)
	}
	klog.V(4).InfoS("Listening for connections", "address", listener.Addr())
	return d.gsrv.Serve(listener)
}

func (d *CSIDriver) NewK8SClientSet() (clientset *kubernetes.Clientset, err error) {
	var config *rest.Config
	if d.options.Kubeconfig != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: d.options.Kubeconfig},
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	// protobuf content type
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	return clientset, nil
}
