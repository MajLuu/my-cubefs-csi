package main

import (
	"flag"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
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
	metricPort string
)

func init() {
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.PersistentFlags().StringVar(&nodeId, "nodeid", "", "This node's ID")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///csi/csi.sock", "CSI endpoint, must be a UNIX socket")
	cmd.PersistentFlags().StringVar(&metricPort, "metric-port", "9001", "Metrics port")
}

var cmd = &cobra.Command{
	Use:   "cfs-csi-driver --endpoint=<endpoint> --nodeid=<nodeid>",
	Short: "CSI based CFS driver",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func main() {
	klog.InitFlags(nil)
	// klog.SetLogger(klog.NewKlogr().WithName("my-cubefs-csi").WithValues("user", "majlu"))
	klog.InfoS("System build info", "BuildTime", BuildTime,
		"Branch", Branch, "CommitID", CommitID)
	if err := cmd.Execute(); err != nil {
		klog.ErrorS(err, "cmd.Execute error")
		os.Exit(1)
	}
	os.Exit(0)
}
