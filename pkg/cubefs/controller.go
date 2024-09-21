package cubefs

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
)

type ControllerService struct {
	csi.UnimplementedControllerServer
}

var _ csi.ControllerServer = (*ControllerService)(nil)

func NewControllerService() *ControllerService {
	return &ControllerService{}
}

func (c ControllerService) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ValidateVolumeCapabilities(ctx context.Context, request *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ListVolumes(ctx context.Context, request *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) GetCapacity(ctx context.Context, request *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerGetCapabilities(ctx context.Context, request *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
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

func (c ControllerService) CreateSnapshot(ctx context.Context, request *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) DeleteSnapshot(ctx context.Context, request *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ListSnapshots(ctx context.Context, request *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerExpandVolume(ctx context.Context, request *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerGetVolume(ctx context.Context, request *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) ControllerModifyVolume(ctx context.Context, request *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c ControllerService) mustEmbedUnimplementedControllerServer() {
	//TODO implement me
	panic("implement me")
}
