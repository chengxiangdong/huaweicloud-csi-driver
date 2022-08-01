package services

import (
	"fmt"

	"github.com/chnsz/golangsdk"
	"github.com/chnsz/golangsdk/openstack/ecs/v1/block_devices"
	"github.com/chnsz/golangsdk/openstack/ecs/v1/cloudservers"
	"github.com/chnsz/golangsdk/openstack/ecs/v1/jobs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/common"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"
)

func GetServer(c *config.CloudCredentials, serverID string) (*cloudservers.CloudServer, error) {
	client, err := getEcsV11Client(c)
	if err != nil {
		return nil, err
	}
	return cloudservers.Get(client, serverID).Extract()
}

func AttachVolume(c *config.CloudCredentials, serverID, volumeID string) (*jobs.Job, error) {
	client, err := getEcsV11Client(c)
	if err != nil {
		return nil, err
	}
	opts := block_devices.AttachOpts{
		ServerId: serverID,
		VolumeId: volumeID,
	}
	return block_devices.Attach(client, opts)
}

func DetachVolume(c *config.CloudCredentials, serverID, volumeID string) (*jobs.Job, error) {
	client, err := getEcsV11Client(c)
	if err != nil {
		return nil, err
	}
	opts := block_devices.DetachOpts{
		ServerId: serverID,
	}
	return block_devices.Detach(client, volumeID, opts)
}

func WaitForVolumeAttached(c *config.CloudCredentials, jobID string) error {
	condition, err := waitForCondition("attaching", c, jobID)
	if err != nil {
		return err
	}
	return common.WaitForCompleted(condition)
}

func WaitForVolumeDetached(c *config.CloudCredentials, jobID string) error {
	condition, err := waitForCondition("detaching", c, jobID)
	if err != nil {
		return err
	}
	return common.WaitForCompleted(condition)
}

func waitForCondition(title string, c *config.CloudCredentials, jobID string) (wait.ConditionFunc, error) {
	client, err := getEcsV11Client(c)
	if err != nil {
		return nil, err
	}

	return func() (bool, error) {
		job, err := jobs.Get(client, jobID)
		if err != nil {
			err = status.Error(codes.Internal,
				fmt.Sprintf("Error waiting for the %s job to be complete, jobId = %s", title, jobID))
			return false, err
		}
		if job.Status == "SUCCESS" {
			return true, nil
		}

		if job.Status == "FAIL" {
			return false, status.Error(codes.Internal,
				fmt.Sprintf("Error %s volume to server, job = %#v", title, job))
		}

		return false, nil
	}, nil
}

func getEcsV11Client(c *config.CloudCredentials) (*golangsdk.ServiceClient, error) {
	client, err := c.EvsV1Client()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed create ECS V1.1 client: %s", err))
	}
	return client, nil
}
