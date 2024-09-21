package cubefs

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

var (
	_ csi.IdentityServer = (*IdentityService)(nil)
)

type IdentityService struct {
	Name    string
	Version string
	csi.UnimplementedIdentityServer
}

func NewIdentityService(name, version string) *IdentityService {
	return &IdentityService{
		Name:    name,
		Version: version,
	}
}

func (d *IdentityService) GetPluginInfo(ctx context.Context, request *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	klog.InfoS("GetPluginInfo: called with args", "req", request.String())
	resp := &csi.GetPluginInfoResponse{
		Name:          d.Name,
		VendorVersion: d.Version,
	}
	return resp, nil
}

func (d *IdentityService) GetPluginCapabilities(ctx context.Context, request *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	klog.InfoS("GetPluginCapabilities: called with args", "req", request.String())
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}
	return resp, nil
}

func (d *IdentityService) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	klog.V(6).InfoS("Probe: called", "args", req)
	return &csi.ProbeResponse{}, nil
}
