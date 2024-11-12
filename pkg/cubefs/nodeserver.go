package cubefs

import (
	"context"
	"os"
	"path/filepath"
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
	nodeCaps []csi.NodeServiceCapability_RPC_Type
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

func (n *NodeService) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *NodeService) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	klog.V(4).InfoS("NodePublishVolume: called", "args", request)
	start := time.Now()
	// we mount the cubefs volume to /mnt dir firstly and then mount to pods volume path
	//stagingTargetPath := request.GetStagingTargetPath()
	targetPath := request.GetTargetPath()
	klog.V(10).InfoS("NodePublishVolume", "targetPath", targetPath)

	// 1. mount the cubefs volume to /mnt dir
	mntDir := filepath.Join("/mnt", request.GetVolumeId())
	if exist, err := n.mounter.PathExists(mntDir); err != nil {
		klog.ErrorS(err, "Failed to check /mnt path", "mntPath", mntDir)
		return nil, status.Error(codes.Internal, err.Error())
	} else if !exist { // make new mnt dir
		klog.InfoS("NodePublishVolume: creating mnt path", "mntDir", mntDir)
		if err := n.mounter.MakeDir(mntDir); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else if isMnt, err := n.mounter.IsMountPoint(mntDir); err == nil {
		if isMnt {
			klog.InfoS("NodePublishVolume: volume already mounted", "mntDir", mntDir)
		} else {
			klog.InfoS("NodePublishVolume: mount dir is already exist, but is not mount point, skip create it", "mntPath", mntDir)
		}
	} else if n.mounter.IsCorruptedMnt(err) {
		// TODO: remount mntDir mount point as corrupted
		klog.ErrorS(err, "NodePublishVolume: mount point is corrupted", "mntDir", mntDir)
		return nil, err
	}

	if err := n.mount(mntDir, request.GetVolumeId(), request.GetVolumeContext()); err != nil {
		return nil, err
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

	if err := n.mounter.Mount(mntDir, targetPath, "", []string{"bind"}); err != nil {
		klog.ErrorS(err, "Failed to bind mount mnt path to target path", "mntPath", mntDir, "targetPath", targetPath)
		return nil, status.Error(codes.Internal, err.Error())
	}

	duration := time.Since(start)
	klog.InfoS("NodePublishVolume success", "mntPath", mntDir, "targetPath", targetPath, "cost", duration)

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
	err := mountutils.CleanupMountPoint(filepath.Join("/mnt", request.GetVolumeId()), n.mounter, false)
	if err != nil {
		klog.ErrorS(err, "Failed to unmount mnt path", "mntDir", filepath.Join("/mnt", request.GetVolumeId()))
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
