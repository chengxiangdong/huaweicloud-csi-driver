package services

import (
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"

	"github.com/chnsz/golangsdk/openstack/evs/v2/snapshots"
)

const (
	SnapshotCreating      = "creating"
	SnapshotAvailable     = "available"
	SnapshotError         = "error"
	SnapshotDeleting      = "deleting"
	SnapshotErrorDeleting = "error_deleting"
	SnapshotRollbacking   = "rollbacking"
	SnapshotBackingUp     = "backing-up"
)

func GetSnapshot(c *config.CloudCredentials, id string) (*snapshots.Snapshot, error) {
	client, err := getEvsV2Client(c)
	if err != nil {
		return nil, err
	}

	return snapshots.Get(client, id).Extract()
}
