package cubefs

import (
	"os/exec"

	"k8s.io/klog/v2"
)

const (
	CfsClientBin = "/cfs/bin/cfs-client"
)

func MountVolume(configFilePath string) error {
	cmd := exec.Command(CfsClientBin, "-c", configFilePath)
	output, err := cmd.CombinedOutput()
	klog.InfoS("Cfs Client bin run output", "output", output)
	return err
}
