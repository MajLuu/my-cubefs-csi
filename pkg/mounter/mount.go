package mounter

import (
	"os"

	mountutils "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

// Mounter is the interface implemented by NodeMounter.
// A mix & match of functions defined in upstream libraries. (FormatAndMount
// from struct SafeFormatAndMount, PathExists from an old edition of
// mount.Interface). Define it explicitly so that it can be mocked and to
// insulate from oft-changing upstream interfaces/structs
type Mounter interface {
	mountutils.Interface

	FormatAndMountSensitiveWithFormatOptions(source string, target string, fstype string, options []string, sensitiveOptions []string, formatOptions []string) error
	IsCorruptedMnt(err error) bool
	GetDeviceNameFromMount(mountPath string) (string, int, error)
	MakeFile(path string) error
	MakeDir(path string) error
	PathExists(path string) (bool, error)
	NeedResize(devicePath string, deviceMountPath string) (bool, error)
}

// NodeMounter implements Mounter.
// A superstruct of SafeFormatAndMount.
type NodeMounter struct {
	*mountutils.SafeFormatAndMount
}

func (nm *NodeMounter) IsCorruptedMnt(err error) bool {
	return mountutils.IsCorruptedMnt(err)
}

func (nm *NodeMounter) GetDeviceNameFromMount(mountPath string) (string, int, error) {
	return mountutils.GetDeviceNameFromMount(nm, mountPath)
}

func (nm *NodeMounter) MakeFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE, os.FileMode(0644))
	if err != nil {
		return err
	}
	return f.Close()
}

func (nm *NodeMounter) MakeDir(path string) error {
	return os.Mkdir(path, os.FileMode(0755))
}

func (nm *NodeMounter) PathExists(path string) (bool, error) {
	return mountutils.PathExists(path)
}

// NeedResize called at NodeStage to ensure file system is the correct size
func (nm *NodeMounter) NeedResize(devicePath string, deviceMountPath string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

// NewNodeMounter returns a new intsance of NodeMounter.
func NewNodeMounter() Mounter {
	safeMounter := NewSafeMounter()
	return &NodeMounter{safeMounter}
}

func NewSafeMounter() *mountutils.SafeFormatAndMount {
	return &mountutils.SafeFormatAndMount{
		Interface: mountutils.New(""),
		Exec:      utilexec.New(),
	}
}
