package evs

import "time"

const (
	// LvmVolumeType lvm volume type
	LvmVolumeType = "LVM"
	// PmemVolumeType lvm volume type
	PmemVolumeType = "PMEM"
	// QuotaPathVolumeType ...
	QuotaPathVolumeType = "QuotaPath"
	// MountPointType type
	MountPointType = "MountPoint"
	// DeviceVolumeType type
	DeviceVolumeType = "Device"
	// DeviceVolumeKey type
	DeviceVolumeKey = "device"
	// VolumeTypeKey volume type key words
	VolumeTypeKey = "volumeType"
	// connection timeout
	connectTimeout = 3 * time.Second
	// TopologyNodeKey define host name of node
	TopologyNodeKey = "kubernetes.io/hostname"
	// PvcNameTag in annotations
	PvcNameTag = "csi.storage.k8s.io/pvc/name"
	// PvcNsTag in annotations
	PvcNsTag = "csi.storage.k8s.io/pvc/namespace"
	// PvNameKey key
	PvNameKey = "csi.storage.k8s.io/pv/name"

	VolumeSnapshotNameKey = "csi.storage.k8s.io/volumesnapshot/name"
	VolumeSnapshotNsKey = "csi.storage.k8s.io/volumesnapshot/namespace"
	VolumeSnapshotContentKey = "csi.storage.k8s.io/volumesnapshotcontent/name"

	// NodeSchedueTag in annotations
	NodeSchedueTag = "volume.kubernetes.io/selected-node"
	// StorageSchedueTag in annotations
	StorageSchedueTag = "volume.kubernetes.io/selected-storage"
	// LastAppliyAnnotationTag tag
	LastAppliyAnnotationTag = "kubectl.kubernetes.io/last-applied-configuration"
	// CsiProvisionerIdentity tag
	CsiProvisionerIdentity = "storage.kubernetes.io/csiProvisionerIdentity"
	// CsiProvisionerTag tag
	CsiProvisionerTag = "volume.beta.kubernetes.io/storage-provisioner"
	// QuotaRootPath tag
	QuotaRootPath = "rootPath"
)

