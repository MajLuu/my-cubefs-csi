package cubefs

import (
	"context"
	"k8s.io/client-go/kubernetes"
	"os"

	mountutils "k8s.io/mount-utils"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/majlu/my-cubefs-csi/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// Supported access modes
const (
	SingleNodeWriter = csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER
)

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
)

type NodeService struct {
	NodeId    string
	mounter   mounter.Mounter
	ClientSet *kubernetes.Clientset
	csi.UnimplementedNodeServer
}

var _ csi.NodeServer = (*NodeService)(nil)

func NewNodeService(nodeId string, clientSet *kubernetes.Clientset) *NodeService {
	return &NodeService{
		NodeId:    nodeId,
		mounter:   mounter.NewNodeMounter(),
		ClientSet: clientSet,
	}
}

func (n *NodeService) NodeStageVolume(ctx context.Context, request *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.V(4).InfoS("NodeStageVolume: called", "args", request)
	volumeID := request.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	stagingTargetPath := request.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	volCap := request.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	if err := n.mount(stagingTargetPath, request.GetVolumeId(), request.GetVolumeContext()); err != nil {
		return nil, err
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeService) mount(targetPath, volumeName string, param map[string]string) (retErr error) {
	defer func() {
		if retErr != nil {
			klog.ErrorS(retErr, "targetPath", targetPath)
			if err := os.Remove(targetPath); err != nil {
				klog.ErrorS(err, "targetPath remove failed")
				return
			}
			klog.InfoS("removed target path", "targetPath", targetPath)
		}
	}()

	pathExists, pathErr := n.mounter.PathExists(targetPath)
	corruptedMnt := n.mounter.IsCorruptedMnt(pathErr)
	if pathExists && !corruptedMnt {
		klog.InfoS("volume already mounted correctly", "stagingTargetPath", targetPath)
		return
	}

	if err := mountutils.CleanupMountPoint(targetPath, n.mounter, false); err != nil {
		retErr = status.Errorf(codes.Internal, "CleanupMountPoint fail, stagingTargetPath: %v error: %v", targetPath, err)
		return
	}

	if err := createMountPoint(targetPath); err != nil {
		retErr = status.Errorf(codes.Internal, "createMountPoint fail, stagingTargetPath: %v error: %v", targetPath, err)
		return
	}

	cfsServer, err := NewCfsServer(volumeName, param)
	if err != nil {
		retErr = status.Errorf(codes.InvalidArgument, "new cfs server failed: %v", err)
		return
	}

	if err := cfsServer.persistClientConf(targetPath); err != nil {
		retErr = status.Errorf(codes.Internal, "persist client config file failed: %v", err)
		return
	}

	if err := cfsServer.runClient(); err != nil {
		retErr = status.Errorf(codes.Internal, "mount failed: %v", err)
		return
	}

	return
}

func createMountPoint(root string) error {
	return os.MkdirAll(root, 0750)
}

func isValidVolumeCapabilities(v []*csi.VolumeCapability) bool {
	for _, c := range v {
		if !isValidCapability(c) {
			return false
		}
	}
	return true
}

func isValidCapability(c *csi.VolumeCapability) bool {
	accessMode := c.GetAccessMode().GetMode()

	switch accessMode {
	case SingleNodeWriter:
		return true
	default:
		klog.InfoS("isValidCapability: access mode is not supported", "accessMode", accessMode)
		return false
	}
}

func (n *NodeService) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	stagingTargetPath := request.GetStagingTargetPath()
	err := mountutils.CleanupMountPoint(stagingTargetPath, n.mounter, false)
	if err != nil {
		return nil, err
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *NodeService) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *NodeService) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *NodeService) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *NodeService) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n *NodeService) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(4).InfoS("NodeGetCapabilities: called", "args", request)
	var caps []*csi.NodeServiceCapability
	for _, cap := range nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (n *NodeService) NodeGetInfo(ctx context.Context, request *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(4).InfoS("NodeGetInfo: called", "args", request)
	return &csi.NodeGetInfoResponse{
		NodeId: n.NodeId,
	}, nil
}
