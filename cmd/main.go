package main

import (
	"flag"

	"github.com/majlu/my-cubefs-csi/pkg/cubefs"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	defaultVersion = "0.0.1"
)

// injected while compile
var (
	CommitID  = ""
	BuildTime = ""
	Branch    = ""
)

var (
	endpoint   string
	nodeId     string
	mode       string
	version    string
	driverName string
	kubeConfig string
)

func init() {
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.PersistentFlags().StringVar(&nodeId, "nodeid", "", "This node's ID")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///csi/csi.sock", "CSI endpoint, must be a UNIX socket")
	cmd.PersistentFlags().StringVar(&version, "version", defaultVersion, "Driver version")
	cmd.PersistentFlags().StringVar(&mode, "mode", string(cubefs.AllMode), "Driver mode, k8s or standalone, supports: controller, node, all")
	cmd.PersistentFlags().StringVar(&driverName, "driver-name", cubefs.DriverName, "Driver name")
	cmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "Kubernetes config file, default we assume in cluster mode")
}

var cmd = &cobra.Command{
	Use:   "cfs-csi-driver --endpoint=<endpoint> --nodeid=<nodeid> --mode=<mode> --kubeconfig=<kubeconfig> --version=<version> --driver-name=<driver-name>",
	Short: "CSI based CFS driver",
	Run: func(cmd *cobra.Command, args []string) {
		opts := cubefs.Options{
			Mode:       cubefs.Mode(mode),
			Kubeconfig: kubeConfig,
			Endpoint:   endpoint,
		}
		drv, err := cubefs.NewCSIDriver(driverName, nodeId, version, &opts)
		if err != nil {
			klog.ErrorS(err, "failed to create driver")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		if err = drv.Run(); err != nil {
			klog.ErrorS(err, "failed to run driver")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	},
}

func main() {
	klog.InitFlags(nil)
	// klog.SetLogger(klog.NewKlogr().WithName("my-cubefs-csi").WithValues("user", "majlu"))
	klog.InfoS("System build info", "BuildTime", BuildTime,
		"Branch", Branch, "CommitID", CommitID)
	if err := cmd.Execute(); err != nil {
		klog.ErrorS(err, "cmd.Execute error")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}
