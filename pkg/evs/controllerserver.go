package evs

import (
	"fmt"

	"github.com/chnsz/golangsdk/openstack/evs/v2/cloudvolumes"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"k8s.io/klog/v2"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/common"
	services "github.com/huaweicloud/huaweicloud-csi-driver/pkg/evs/services"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/utils"
)

const (
	csiClusterNodeIDKey  = "evs.csi.huaweicloud.com/nodeId"
	dssIDKey             = "dedicated_storage_id"
	createForVolumeIDKey = "create_for_volume_id"
	hwPassthroughKey     = "hw:passthrough"
)

type controllerServer struct {
	Driver *EvsDriver
}

func (cs *controllerServer) CreateVolume(_ context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse,
	error) {
	log.Infof("CreateVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	volName := req.GetName()
	volCapabilities := req.GetVolumeCapabilities()

	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[CreateVolume] missing Volume Name")
	}

	if volCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "[CreateVolume] missing Volume capability")
	}

	// Volume Size - Default is 1 GiB
	volSizeBytes := int64(1 * 1024 * 1024 * 1024)
	if req.GetCapacityRange() != nil {
		volSizeBytes = int64(req.GetCapacityRange().GetRequiredBytes())
	}
	volSizeGB := int(utils.RoundUpSize(volSizeBytes, 1024*1024*1024))

	parameters := req.GetParameters()
	// Volume Type
	volType := parameters["type"]

	var volAvailability string

	// First check if volAvailability is already specified, if not get preferred from Topology
	// Required, incase vol AZ is different from node AZ
	volAvailability = parameters["availability"]
	dssID := parameters["dssId"]
	scsi := parameters["scsi"]

	if len(volAvailability) == 0 {
		// Check from Topology
		if req.GetAccessibilityRequirements() != nil {
			volAvailability = getAZFromTopology(req.GetAccessibilityRequirements())
			log.Infof("Get AZ By GetAccessibilityRequirements Availability Zone: %s", volAvailability)
		}
	}

	credentials := &cs.Driver.cloud

	listOpts := cloudvolumes.ListOpts{
		Name: volName,
	}
	// Verify a volume with the provided name doesn't already exist for this tenant
	volumes, err := services.ListVolumes(credentials, listOpts)
	if err != nil {
		log.Errorf("Failed to query for existing Volume during CreateVolume: %v", err)
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("Failed to query for existing Volume during CreateVolume: %s", err))
	}

	if len(volumes) == 1 {
		if volSizeGB != volumes[0].Size {
			return nil, status.Error(codes.AlreadyExists, "Volume Already exists with same name and different capacity")
		}
		log.Infof("Volume %s already exists in Availability Zone: %s of size %d GiB", volumes[0].ID, volumes[0].AvailabilityZone, volumes[0].Size)
		return getCreateVolumeResponse(&volumes[0], dssID, req.GetAccessibilityRequirements()), nil
	} else if len(volumes) > 1 {
		log.Infof("found multiple existing volumes with selected name (%s) during create", volName)
		return nil, status.Error(codes.AlreadyExists, "Multiple volumes reported by Cinder with same name")
	}

	// the metadata of create option
	metadata := make(map[string]string)
	metadata[csiClusterNodeIDKey] = cs.Driver.nodeID
	metadata[createForVolumeIDKey] = "true"
	metadata[dssIDKey] = dssID

	if scsi != "" && (scsi == "true" || scsi == "false") {
		metadata[hwPassthroughKey] = scsi
	}

	for _, mKey := range []string{PvcNameTag, PvcNsTag, PvNameKey} {
		if v, ok := parameters[mKey]; ok {
			metadata[mKey] = v
		}
	}

	content := req.GetVolumeContentSource()
	var snapshotID string
	var sourceVolID string

	if content != nil && content.GetSnapshot() != nil {
		snapshotID = content.GetSnapshot().GetSnapshotId()
		_, err := services.GetSnapshot(credentials, snapshotID)
		if err != nil {
			if common.IsNotFound(err) {
				return nil, status.Errorf(codes.NotFound, "VolumeContentSource Snapshot %s not found", snapshotID)
			}
			return nil, status.Errorf(codes.Internal, "Failed to retrieve the snapshot %s: %v", snapshotID, err)
		}
	}

	if content != nil && content.GetVolume() != nil {
		sourceVolID = content.GetVolume().GetVolumeId()
		_, err := services.GetVolume(credentials, sourceVolID)
		if err != nil {
			if common.IsNotFound(err) {
				return nil, status.Errorf(codes.NotFound, "Source Volume %s not found", sourceVolID)
			}
			return nil, status.Errorf(codes.Internal, "Failed to retrieve the source volume %s: %v", sourceVolID, err)
		}
	}

	// Create volume
	createOpts := cloudvolumes.CreateOpts{
		Volume: cloudvolumes.VolumeOpts{
			Name:             volName,
			Size:             volSizeGB,
			VolumeType:       volType,
			AvailabilityZone: volAvailability,
			SnapshotID:       snapshotID,
			Metadata:         metadata,
		},
	}
	log.Infof("Create EVS volume opts: %#v", createOpts)
	createJob, err := services.CreateVolume(credentials, &createOpts)
	if err != nil {
		errLog := fmt.Sprintf("CreateVolume failed with error %v", err)
		log.Errorf(errLog)
		return nil, status.Error(codes.Internal, errLog)
	}

	// we need wait for the volume to be available
	volumeID, err := services.WaitForCreateEvsJob(credentials, createJob.JobID)

	if err != nil {
		log.Errorf(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	volume, err := services.GetVolume(credentials, volumeID)
	if err != nil {
		errLog := fmt.Sprintf("Failed to query volume detail by id: %#v", err)
		log.Errorf(errLog)
		return nil, status.Error(codes.Internal, errLog)
	}

	log.Infof("CreateVolume: Successfully created volume %s in Availability Zone: %s of size %d GiB",
		volume.ID, volume.AvailabilityZone, volume.Size)
	return getCreateVolumeResponse(volume, dssID, req.GetAccessibilityRequirements()), nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	log.Infof("DeleteVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	// Volume Delete
	volumeId := req.GetVolumeId()
	if len(volumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "DeleteVolume Volume ID must be provided")
	}

	credentials := &cs.Driver.cloud
	if err := services.DeleteVolume(credentials, volumeId); err != nil {
		if common.IsNotFound(err) {
			log.Infof("Volume %s is already deleted.", volumeId)
			return &csi.DeleteVolumeResponse{}, nil
		}
		log.Errorf("Failed to DeleteVolume, id: %s, error: %v", volumeId, err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("DeleteVolume failed with error %v", err))
	}
	log.Infof("DeleteVolume: Successfully deleted volume %s", volumeId)

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	klog.Infof("ControllerGetVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	volumeID := req.GetVolumeId()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	credentials := &cs.Driver.cloud
	volume, err := services.GetVolume(credentials, volumeID)
	if err != nil {
		if common.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("ControllerGetVolume failed with error %v", err))
	}

	status := &csi.ControllerGetVolumeResponse_VolumeStatus{}
	for _, attachment := range volume.Attachments {
		status.PublishedNodeIds = append(status.PublishedNodeIds, attachment.ServerID)
	}

	response := csi.ControllerGetVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: int64(volume.Size * 1024 * 1024 * 1024),
		},
		Status: status,
	}

	return &response, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	log.Infof("ControllerPublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	// Volume Attach
	instanceID := req.GetNodeId()
	volumeID := req.GetVolumeId()
	volumeCapability := req.GetVolumeCapability()

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[ControllerPublishVolume] Volume ID must be provided")
	}
	if len(instanceID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[ControllerPublishVolume] Instance ID must be provided")
	}
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "[ControllerPublishVolume] Volume capability must be provided")
	}

	credentials := &cs.Driver.cloud
	_, err := services.GetVolume(credentials, volumeID)
	if err != nil {
		if common.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "[ControllerPublishVolume] Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerPublishVolume] get volume failed with error %v", err))
	}

	_, err = services.GetServer(credentials, instanceID)
	if err != nil {
		if common.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "[ControllerPublishVolume] Instance %s not found", instanceID)
		}
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerPublishVolume] GetInstanceByID failed with error %v", err))
	}

	job, err := services.AttachVolume(credentials, instanceID, volumeID)
	if err != nil {
		klog.Errorf("Failed to AttachVolume: %v", err)
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerPublishVolume] Attach Volume failed with error %v", err))
	}

	err = services.WaitForVolumeAttached(credentials, job.ID)
	if err != nil {
		klog.Errorf("Failed to wait EVS volume be attached: %v", err)
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerPublishVolume] Failed to wait EVS volume be attached: %v", err))
	}

	volume, err := services.GetVolume(credentials, volumeID)
	if err != nil {
		klog.Errorf("Failed to GetAttachmentDiskPath: %v", err)
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerPublishVolume] failed to get device path of attached volume : %v", err))
	}

	devicePath := ""
	for _, attach := range volume.Attachments {
		if attach.ServerID == instanceID {
			devicePath = attach.Device
			break
		}
	}

	klog.Infof("ControllerPublishVolume %s on %s is successful", volumeID, instanceID)

	// Publish Volume Info
	pvInfo := map[string]string{}
	pvInfo["DevicePath"] = devicePath

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: pvInfo,
	}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(_ context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.Infof("ControllerUnpublishVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	// Volume Detach
	serverID := req.GetNodeId()
	volumeID := req.GetVolumeId()

	if len(serverID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "[ControllerUnpublishVolume] Volume ID must be provided")
	}

	credentials := &cs.Driver.cloud
	_, err := services.GetServer(credentials, serverID)
	if err != nil {
		if common.IsNotFound(err) {
			klog.Infof("ControllerUnpublishVolume assuming volume %s is detached, because node %s does not exist", volumeID, serverID)
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("[ControllerUnpublishVolume] GetInstanceByID failed with error %v", err))
	}

	job, err := services.DetachVolume(credentials, serverID, volumeID)
	if err != nil {
		if common.IsNotFound(err) {
			klog.Infof("ControllerUnpublishVolume assuming volume %s is detached, because it does not exist", volumeID)
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		klog.Errorf("Failed to DetachVolume: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("ControllerUnpublishVolume Detach Volume failed with error %v", err))
	}

	err = services.WaitForVolumeDetached(credentials, job.ID)
	if err != nil {
		klog.Errorf("Failed to WaitDiskDetached: %v", err)
		if common.IsNotFound(err) {
			klog.Infof("ControllerUnpublishVolume assuming volume %s is detached, because it was deleted in the meanwhile", volumeID)
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("ControllerUnpublishVolume failed with error %v", err))
	}

	klog.Infof("ControllerUnpublishVolume %s on %s", volumeID, serverID)

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ListVolumes(_ context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse,
	error) {
	if req.MaxEntries < 0 {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("[ListVolumes] Invalid max entries request %v, must not be negative ", req.MaxEntries))
	}

	listOpts := cloudvolumes.ListOpts{
		Marker: req.StartingToken,
		Limit:  int(req.MaxEntries),
	}
	volumes, err := services.ListVolumes(&cs.Driver.cloud, listOpts)
	if err != nil {
		klog.Errorf("Failed to ListVolumes: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("ListVolumes failed with error %v", err))
	}

	entries := make([]*csi.ListVolumesResponse_Entry, len(volumes))
	for _, vol := range volumes {
		publishedNodeIds := make([]string, len(vol.Attachments))
		for _, attachment := range vol.Attachments {
			publishedNodeIds = append(publishedNodeIds, attachment.ServerID)
		}

		entry := csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId:      vol.ID,
				CapacityBytes: int64(vol.Size * 1024 * 1024 * 1024),
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				PublishedNodeIds: publishedNodeIds,
			},
		}

		entries = append(entries, &entry)
	}

	nextToken := ""
	if len(volumes) > 0 {
		nextToken = volumes[len(volumes)-1].ID
	}

	return &csi.ListVolumesResponse{
		Entries:   entries,
		NextToken: nextToken,
	}, nil
}

func (cs *controllerServer) CreateSnapshot(_ context.Context, req *csi.CreateSnapshotRequest) (
	*csi.CreateSnapshotResponse, error) {

	return &csi.CreateSnapshotResponse{
	}, nil
}

func (cs *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (
	*csi.DeleteSnapshotResponse, error) {

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (
	*csi.ListSnapshotsResponse, error) {
	return &csi.ListSnapshotsResponse{
	}, nil
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *controllerServer) ControllerGetCapabilities(_ context.Context, req *csi.ControllerGetCapabilitiesRequest) (
	*csi.ControllerGetCapabilitiesResponse, error) {

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.cscap,
	}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(_ context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (
	*csi.ValidateVolumeCapabilitiesResponse, error) {

	reqVolCap := req.GetVolumeCapabilities()
	volumeID := req.GetVolumeId()

	if reqVolCap == nil || len(reqVolCap) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume Capabilities must be provided")
	}

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ValidateVolumeCapabilities Volume ID must be provided")
	}

	_, err := services.GetVolume(&cs.Driver.cloud, volumeID)
	if err != nil {
		if common.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("ValidateVolumeCapabiltites Volume %s not found", volumeID))
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("ValidateVolumeCapabiltites %v", err))
	}

	for _, cap := range reqVolCap {
		if cap.GetAccessMode().GetMode() != cs.Driver.vcap[0].Mode {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: "Requested Volume Capability not supported"}, nil
		}
	}

	// Cinder CSI driver currently supports one mode only
	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessMode: cs.Driver.vcap[0],
				},
			},
		},
	}

	return resp, nil
}

func (cs *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (
	*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, fmt.Sprintf("GetCapacity is not yet implemented"))
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (
	*csi.ControllerExpandVolumeResponse, error) {

	klog.Infof("ControllerExpandVolume: called with args %+v", protosanitizer.StripSecrets(*req))

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}
	cap := req.GetCapacityRange()
	if cap == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity range not provided")
	}

	cc := &cs.Driver.cloud
	volSizeBytes := int64(req.GetCapacityRange().GetRequiredBytes())
	volSizeGB := int(utils.RoundUpSize(volSizeBytes, 1024*1024*1024))
	maxVolSize := cap.GetLimitBytes()

	if maxVolSize > 0 && maxVolSize < volSizeBytes {
		return nil, status.Error(codes.OutOfRange, "After round-up, volume size exceeds the limit specified")
	}

	volume, err := services.GetVolume(cc, volumeID)
	if err != nil {
		if common.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "Volume not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("GetVolume failed with error %v", err))
	}

	if volume.Size >= volSizeGB {
		// a volume was already resized
		klog.Infof("Volume %q has been already expanded to %d, requested %d", volumeID, volume.Size, volSizeGB)
		return &csi.ControllerExpandVolumeResponse{
			CapacityBytes:         int64(volume.Size * 1024 * 1024 * 1024),
			NodeExpansionRequired: true,
		}, nil
	}

	err = services.ExpandVolume(cc, volumeID, volSizeGB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Could not resize volume %q to size %v: %v", volumeID, volSizeGB, err))
	}

	klog.Infof("ControllerExpandVolume resized volume %v to size %v", volumeID, volSizeGB)

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         volSizeBytes,
		NodeExpansionRequired: true,
	}, nil
}

func getAZFromTopology(requirement *csi.TopologyRequirement) string {
	for _, topology := range requirement.GetPreferred() {
		zone, exists := topology.GetSegments()[topologyKey]
		if exists {
			return zone
		}
	}

	for _, topology := range requirement.GetRequisite() {
		zone, exists := topology.GetSegments()[topologyKey]
		if exists {
			return zone
		}
	}
	return ""
}

func getCreateVolumeResponse(vol *cloudvolumes.Volume, dssID string,
	accessibleTopologyReq *csi.TopologyRequirement) *csi.CreateVolumeResponse {
	var volsrc *csi.VolumeContentSource
	if vol.SnapshotID != "" {
		volsrc = &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Snapshot{
				Snapshot: &csi.VolumeContentSource_SnapshotSource{
					SnapshotId: vol.SnapshotID,
				},
			},
		}
	}

	if vol.SourceVolID != "" {
		volsrc = &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Volume{
				Volume: &csi.VolumeContentSource_VolumeSource{
					VolumeId: vol.SourceVolID,
				},
			},
		}
	}

	// If ignore-volume-az is true , dont set the accessible topology to volume az,
	// use from preferred topologies instead.
	accessibleTopology := []*csi.Topology{
		{
			Segments: map[string]string{topologyKey: vol.AvailabilityZone},
		},
	}

	VolumeContext := make(map[string]string)
	if dssID != "" {
		VolumeContext[dssIDKey] = dssID
	}
	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:           vol.ID,
			CapacityBytes:      int64(vol.Size * 1024 * 1024 * 1024),
			AccessibleTopology: accessibleTopology,
			ContentSource:      volsrc,
			VolumeContext:      VolumeContext,
		},
	}
	return resp
}
