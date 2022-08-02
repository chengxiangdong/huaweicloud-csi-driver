package evs

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/utils/metadata"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/utils/mount"
)

const (
	driverName  = "evs.csi.huaweicloud.com"
	topologyKey = "topology." + driverName + "/zone"
)

var (
	// CSI spec version
	specVersion = "1.3.0"
	// Driver version
	Version = "1.0.0"
)

type EvsDriver struct {
	name       string
	nodeID     string
	version    string
	endpoint   string
	cluster    string
	shareProto string
	cloud      config.CloudCredentials

	ids *identityServer
	cs  *ControllerServer
	ns  *nodeServer

	vcap  []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
}

func NewDriver(cloud *config.CloudCredentials, endpoint, cluster, nodeID string) *EvsDriver {
	d := &EvsDriver{}
	d.name = driverName
	d.version = fmt.Sprintf("%s@%s", Version, specVersion)
	d.endpoint = endpoint
	d.cluster = cluster
	d.nodeID = nodeID
	d.cloud = *cloud

	klog.Info("Driver: ", d.name)
	klog.Info("Driver version: ", d.version)
	klog.Info("CSI Spec version: ", specVersion)

	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
			csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
			csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
			csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
			csi.ControllerServiceCapability_RPC_GET_VOLUME,
		})
	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	d.AddNodeServiceCapabilities(
		[]csi.NodeServiceCapability_RPC_Type{
			csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
			csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		})

	d.ids = &identityServer{Driver: d}
	d.cs = &ControllerServer{Driver: d}
	d.ns = &nodeServer{Driver: d}

	return d
}

func (d *EvsDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability

	for _, c := range cl {
		klog.Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: c,
				},
			},
		})
	}

	d.cscap = csc

	return
}

func (d *EvsDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		klog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	d.vcap = vca
	return vca
}

func (d *EvsDriver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) error {
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		klog.Infof("Enabling node service capability: %v", n.String())
		nsc = append(nsc, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: n,
				},
			},
		})
	}
	d.nscap = nsc
	return nil
}

func (d *EvsDriver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, cap := range d.cscap {
		if c == cap.GetRpc().GetType() {
			return nil
		}
	}
	return status.Error(codes.InvalidArgument, fmt.Sprintf("%s", c))
}

func (d *EvsDriver) GetControllerServer() *ControllerServer {
	return d.cs
}

func (d *EvsDriver) GetIdentityServer() *identityServer {
	return d.ids
}

func (d *EvsDriver) GetNodeServer() *nodeServer {
	return d.ns
}

func (d *EvsDriver) GetVolumeCapabilityAccessModes() []*csi.VolumeCapability_AccessMode {
	return d.vcap
}

func (d *EvsDriver) SetupDriver(mount mount.IMount, metadata metadata.IMetadata) {
	d.ns.Mount = mount
	d.ns.Metadata = metadata
}

func (d *EvsDriver) Run() {
	s := NewNonBlockingGRPCServer()
	s.Start(d.endpoint, d.ids, d.cs, d.ns)
	s.Wait()
}
