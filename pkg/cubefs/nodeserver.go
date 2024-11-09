package cubefs

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/majlu/my-cubefs-csi/pkg/mounter"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	mountutils "k8s.io/mount-utils"
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
	mutex     sync.Mutex
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
	n.mutex.Lock()
	defer n.mutex.Unlock()

	klog.V(4).InfoS("NodeStageVolume: called", "args", request)
	volumeID := request.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	stagingTargetPath := request.GetStagingTargetPath()
	klog.V(10).InfoS("NodeStageVolume StagingTargetPath", "StagingTargetPath", stagingTargetPath)
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	pathExists, err := n.mounter.PathExists(stagingTargetPath)
	if err != nil {
		klog.ErrorS(err, "PathExists fail", "stagingTargetPath", stagingTargetPath)
		return nil, status.Error(codes.Internal, err.Error())
	} else if pathExists {
		isMountPoint, err := n.mounter.IsMountPoint(stagingTargetPath)
		if err != nil {
			klog.ErrorS(err, "IsMountPoint fail", "stagingTargetPath", stagingTargetPath)
			return nil, status.Error(codes.Internal, err.Error())
		}
		if isMountPoint {
			klog.InfoS("NodeStageVolume: volume already mounted", "StagingTargetPath", stagingTargetPath)
			return &csi.NodeStageVolumeResponse{}, nil
		}
	} else {
		klog.InfoS("NodeStageVolume: creating target path", "StagingTargetPath", stagingTargetPath)
		if err := n.mounter.MakeDir(stagingTargetPath); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
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
	n.mutex.Lock()
	defer n.mutex.Unlock()

	klog.V(4).InfoS("NodePublishVolume: called", "args", request)
	start := time.Now()
	stagingTargetPath := request.GetStagingTargetPath()
	targetPath := request.GetTargetPath()
	klog.V(10).InfoS("NodePublishVolume", "stagingTargetPath",
		stagingTargetPath, "targetPath", targetPath)

	if exist, err := n.mounter.PathExists(stagingTargetPath); err != nil {
		klog.ErrorS(err, "Failed to check staging path", "stagingTargetPath", stagingTargetPath)
		return nil, status.Error(codes.Internal, err.Error())
	} else if !exist {
		klog.ErrorS(nil, "Staging target path does not exist", "stagingTargetPath", stagingTargetPath)
		return nil, status.Error(codes.NotFound, "Staging target path not found")
	}

	if exist, err := n.mounter.PathExists(targetPath); err != nil {
		klog.ErrorS(err, "Failed to check target path", "targetPath", targetPath)
		return nil, status.Error(codes.Internal, err.Error())
	} else if exist {
		klog.V(4).InfoS("Target path already exists", "targetPath", targetPath)
		isNotMountPoint, err := n.mounter.IsLikelyNotMountPoint(targetPath)
		if err != nil {
			klog.ErrorS(err, "Failed to check target path", "targetPath", targetPath)
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !isNotMountPoint {
			klog.V(4).InfoS("Target path is a mount point, skip", "targetPath", targetPath)
			return &csi.NodePublishVolumeResponse{}, nil
		}
	} else {
		if err := n.mounter.MakeDir(targetPath); err != nil {
			klog.ErrorS(err, "Failed to create target path", "targetPath", targetPath)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if err := n.mounter.Mount(stagingTargetPath, targetPath, "", []string{"bind"}); err != nil {
		klog.ErrorS(err, "Failed to bind mount staging target to target path", "stagingTargetPath", stagingTargetPath, "targetPath", targetPath)
		return nil, status.Error(codes.Internal, err.Error())
	}

	duration := time.Since(start)
	klog.InfoS("NodePublishVolume success", "stagingTargetPath", stagingTargetPath, "targetPath", targetPath, "cost", duration)

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeService) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	klog.V(10).InfoS("NodeUnpublishVolume", "targetPath", request.GetTargetPath())
	if err := mountutils.CleanupMountPoint(request.GetTargetPath(), n.mounter, false); err != nil {
		klog.ErrorS(err, "Failed to unmount target path", "targetPath", request.GetTargetPath())
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeService) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return &csi.NodeGetVolumeStatsResponse{}, nil
}

func (n *NodeService) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return &csi.NodeExpandVolumeResponse{}, nil
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
