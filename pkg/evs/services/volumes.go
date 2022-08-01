package services

import (
	"fmt"

	"github.com/chnsz/golangsdk"
	"github.com/chnsz/golangsdk/openstack/evs/v1/jobs"
	"github.com/chnsz/golangsdk/openstack/evs/v2/cloudvolumes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/common"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"
)

const (
	EvsAvailableStatus = "available"
)

func CreateVolume(c *config.CloudCredentials, createOpts *cloudvolumes.CreateOpts) (*cloudvolumes.JobResponse, error) {
	client, err := getEvsV2Client(c)
	if err != nil {
		return nil, err
	}

	return cloudvolumes.Create(client, *createOpts).Extract()
}

func GetVolume(c *config.CloudCredentials, id string) (*cloudvolumes.Volume, error) {
	client, err := getEvsV2Client(c)
	if err != nil {
		return nil, err
	}

	return cloudvolumes.Get(client, id).Extract()
}

func GetVolumeDevicePath(c *config.CloudCredentials, id string) (string, error) {
	volume, err := GetVolume(c, id)
	if err != nil {
		if common.IsNotFound(err) {
			return "", status.Errorf(codes.NotFound, "[getDevicePathV2] Volume %s not found", id)
		}
		return "", status.Error(codes.Internal, fmt.Sprintf("[getDevicePathV2] get volume failed with error %v", err))
	}

	if len(volume.Attachments) > 0 {
		return volume.Attachments[0].Device, nil
	}
	return "", status.Error(codes.Internal, fmt.Sprintf("get volume device path failed with error %v", err))
}

func ExpandVolume(c *config.CloudCredentials, id string, newSize int) error {
	client, err := getEvsV2Client(c)
	if err != nil {
		return err
	}

	opt := cloudvolumes.ExtendOpts{
		SizeOpts: cloudvolumes.ExtendSizeOpts{
			NewSize: newSize,
		},
	}

	job, err := cloudvolumes.ExtendSize(client, id, opt).Extract()
	if err != nil {
		return status.Error(codes.Internal,
			fmt.Sprintf("Error expanding volume, volume: %s, newSize: %v", id, newSize))
	}
	_, err = WaitForCreateEvsJob(c, job.JobID)
	return err
}

func DeleteVolume(c *config.CloudCredentials, id string) error {
	client, err := getEvsV2Client(c)
	if err != nil {
		return err
	}

	return cloudvolumes.Delete(client, id, nil).Err
}

func ListVolumes(c *config.CloudCredentials, opts cloudvolumes.ListOpts) ([]cloudvolumes.Volume, error) {
	client, err := getEvsV2Client(c)
	if err != nil {
		return nil, err
	}

	volumes, err := cloudvolumes.ListWithPage(client, opts)
	if err != nil {
		return nil, fmt.Errorf("An error occurred while fetching the pages of the EVS disks: %s", err)
	}
	return volumes, nil
}

func WaitForCreateEvsJob(c *config.CloudCredentials, jobID string) (string, error) {
	client, err := getJobV1Client(c)
	if err != nil {
		return "", err
	}

	var volumeID string
	err = common.WaitForCompleted(func() (bool, error) {
		job, err := jobs.GetJobDetails(client, jobID).ExtractJob()
		if err != nil {
			err = status.Error(codes.Internal,
				fmt.Sprintf("Error waiting for the creation job to be complete, jobId = %s", jobID))
			return false, err
		}

		if job.Status == "SUCCESS" {
			volumeID = job.Entities.VolumeID
			return true, nil
		}

		if job.Status == "FAIL" {
			err = status.Error(codes.Unavailable, fmt.Sprintf("Error in job creating volume, jobId = %s", jobID))
			return false, err
		}

		return false, nil
	})

	return volumeID, err
}

func getEvsV2Client(c *config.CloudCredentials) (*golangsdk.ServiceClient, error) {
	client, err := c.EvsV21Client()
	if err != nil {
		logMsg := fmt.Sprintf("Failed create EVS V2 client: %s", err)
		return nil, status.Error(codes.Internal, logMsg)
	}
	return client, nil
}

func getEvsV21Client(c *config.CloudCredentials) (*golangsdk.ServiceClient, error) {
	client, err := c.EvsV21Client()
	if err != nil {
		logMsg := fmt.Sprintf("Failed create EVS V2.1 client: %s", err)
		return nil, status.Error(codes.Internal, logMsg)
	}
	return client, nil
}

func getJobV1Client(c *config.CloudCredentials) (*golangsdk.ServiceClient, error) {
	client, err := c.EvsV1Client()
	if err != nil {
		logMsg := fmt.Sprintf("Failed create JOB client V1 client: %s", err)
		return nil, status.Error(codes.Internal, logMsg)
	}
	return client, nil
}
