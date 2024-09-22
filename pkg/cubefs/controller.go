package cubefs

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
)

type ControllerService struct {
	ClientSet *kubernetes.Clientset
	csi.UnimplementedControllerServer
}

var _ csi.ControllerServer = (*ControllerService)(nil)

func NewControllerService(clientSet *kubernetes.Clientset) *ControllerService {
	return &ControllerService{
		ClientSet: clientSet,
	}
}

func (cs ControllerService) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(4).InfoS("CreateVolume: called", "args", request)
	if err := cs.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		return nil, err
	}

	start := time.Now()
	capRange := request.GetCapacityRange()
	if capRange == nil {
		return nil, status.Error(codes.InvalidArgument, "apply for capacity range is nil")
	}
	// GB
	capacityGB := (capRange.GetRequiredBytes()) >> 30
	if capacityGB == 0 {
		return nil, status.Error(codes.InvalidArgument, "apply for at least 1GB of space")
	}

	volName := request.GetName()
	klog.InfoS("Get request vol name", "volName", volName)
	cfsServer, err := NewCfsServer(volName, request.Parameters)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err = cfsServer.createVolume(capacityGB); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	duration := time.Since(start)
	klog.InfoS("Created volume success", "volName", volName, "cost", duration)
	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volName,
			CapacityBytes: capRange.RequiredBytes,
			VolumeContext: cfsServer.clientConf,
		},
	}
	klog.InfoS("Create vol resp", "CreateVolumeResponse", resp)
	return resp, nil
}

func (cs ControllerService) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(4).InfoS("DeleteVolume: called", "args", request)
	if err := cs.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		return nil, err
	}

	volumeName := request.VolumeId
	persistentVolume, err := cs.queryPersistentVolumes(ctx, volumeName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "not found PersistentVolume[%v], error:%v", volumeName, err)
	}

	param := persistentVolume.Spec.CSI.VolumeAttributes
	cfsServer, err := NewCfsServer(volumeName, param)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = cfsServer.deleteVolume()
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	} else {
		klog.V(0).InfoS("Deleted volume", "volName", volumeName)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs ControllerService) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ValidateVolumeCapabilities(ctx context.Context, request *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ListVolumes(ctx context.Context, request *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) GetCapacity(ctx context.Context, request *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ControllerGetCapabilities(ctx context.Context, request *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).InfoS("ControllerGetCapabilities: called", "args", request)
	var caps []*csi.ControllerServiceCapability
	for _, controllerCap := range controllerCaps {
		csc := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: controllerCap,
				},
			},
		}
		caps = append(caps, csc)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (cs ControllerService) CreateSnapshot(ctx context.Context, request *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) DeleteSnapshot(ctx context.Context, request *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ListSnapshots(ctx context.Context, request *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ControllerExpandVolume(ctx context.Context, request *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ControllerGetVolume(ctx context.Context, request *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ControllerModifyVolume(ctx context.Context, request *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) mustEmbedUnimplementedControllerServer() {
	//TODO implement me
	panic("implement me")
}

func (cs ControllerService) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, ccap := range controllerCaps {
		if c == ccap {
			return nil
		}
	}
	return status.Error(codes.InvalidArgument, fmt.Sprintf("%s", c))
}

func (cs ControllerService) queryPersistentVolumes(ctx context.Context, pvName string) (*corev1.PersistentVolume, error) {
	persistentVolume, err := cs.ClientSet.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if persistentVolume == nil {
		return nil, status.Error(codes.Unknown, fmt.Sprintf("not found PersistentVolume[%v]", pvName))
	}

	return persistentVolume, nil
}
