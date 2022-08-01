package evs

import (
	"log"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/common"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/evs"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/evs/services"
	acceptance "github.com/huaweicloud/huaweicloud-csi-driver/test"

	"github.com/stretchr/testify/assert"
)

var (
	name    = "k8s-test-controller"
	size    = int64(10 * 1024 * 1024 * 1024)
	newSize = int64(12 * 1024 * 1024 * 1024)

	availability = "ap-southeast-1a"
	volumeType   = "SSD"
	dssID        = ""
	scsi         = "false"

	cc     *config.CloudCredentials
	cs     *evs.ControllerServer
	ctx    = context.Background()
	driver *evs.EvsDriver
)

func TestVolume(t *testing.T) {
	nodeID := ""
	cc = acceptance.LoadConfig(t)
	driver = evs.NewDriver(cc, "unix://csi/csi.sock", "kubernetes", nodeID)
	ctx = context.Background()
	cs = driver.GetControllerServer()

	volumeID := createVolume(t)
	defer deleteVolume(t, volumeID)
	readVolume(t, volumeID)
	listVolume(t)
	resizeVolume(t, volumeID)
}

func createVolume(t *testing.T) string {
	req := &csi.CreateVolumeRequest{
		Name: name,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: size,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			&csi.VolumeCapability{},
		},
		Parameters: map[string]string{
			"type":         volumeType,
			"availability": availability,
			"dssID":        dssID,
			"scsi":         scsi,
		},
	}

	createRsp, err := cs.CreateVolume(ctx, req)
	assert.Nilf(t, err, "Error creating volume, error: %s", err)
	log.Printf("Volume detail: %#v", createRsp)

	volumeID := createRsp.GetVolume().VolumeId
	assert.NotEmptyf(t, volumeID, "Error creating volume, we could not get the volume ID.")

	return volumeID
}

func listVolume(t *testing.T) {
	req := &csi.ListVolumesRequest{
		StartingToken: "",
	}
	rsp, err := cs.ListVolumes(ctx, req)
	assert.Nilf(t, err, "Error list volume, error: %s", err)
	log.Printf("aaa %#v", rsp)
}

func readVolume(t *testing.T, volumeID string) {
	volume, err := cs.ControllerGetVolume(ctx, &csi.ControllerGetVolumeRequest{
		VolumeId: volumeID,
	})
	if err != nil && !common.IsNotFound(err) {
		t.Fatal("the evs volume is not exists")
	}
	assert.Equalf(t, volumeID, volume.Volume.VolumeId, "The EVS volume ID is not expected.")
	assert.Equalf(t, size, volume.Volume.CapacityBytes, "The EVS volume capacity is not expected.")
}

func resizeVolume(t *testing.T, volumeID string) {
	expandRsp, err := cs.ControllerExpandVolume(ctx, &csi.ControllerExpandVolumeRequest{
		VolumeId: volumeID,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: newSize,
		},
	})
	assert.Nilf(t, err, "Error expanding volume, error: %#v", err)
	assert.EqualValues(t, newSize, expandRsp.CapacityBytes)
}

func deleteVolume(t *testing.T, volumeID string) {
	delReq := &csi.DeleteVolumeRequest{
		VolumeId: volumeID,
	}
	_, err := cs.DeleteVolume(context.Background(), delReq)
	assert.Nilf(t, err, "Error deleting volume, error: %s", err)

	_, err = services.GetVolume(cc, volumeID)
	if err == nil || !common.IsNotFound(err) {
		t.Fatal("the evs volume is still exists")
	}
}
