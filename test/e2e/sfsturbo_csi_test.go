package e2e

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	storageV1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/common"
	"github.com/huaweicloud/huaweicloud-csi-driver/test/e2e/framework"
	"github.com/huaweicloud/huaweicloud-csi-driver/test/e2e/helper"
)

var _ = ginkgo.Describe("SFS Turbo CSI STANDARD testing", func() {
	sfsTurboTest("STANDARD")
})

//
//var _ = ginkgo.Describe("SFS Turbo CSI PERFORMANCE testing", func() {
//	sfsTurboTest("PERFORMANCE")
//})

func sfsTurboTest(shareType string) {
	var sc *storageV1.StorageClass
	var pvc *corev1.PersistentVolumeClaim

	ginkgo.BeforeEach(func() {
		sc = createSC(shareType)
		pvc = createPvc(sc)
	})

	ginkgo.AfterEach(func() {
		framework.RemovePVC(kubeClient, pvc.Namespace, pvc.Name)
		framework.RemoveStorageClass(kubeClient, sc.Name)
	})

	ginkgo.It(fmt.Sprintf("Mount SFS Trubo[%s] to a Pod testing", shareType), func() {
		pod := newPodWithPvc(pvc)
		framework.CreatePod(kubeClient, pod)
		framework.WaitPodPresentOnClusterFitWith(kubeClient, pod.Namespace, pod.Name, func(pod *corev1.Pod) bool {
			return pod.Status.Phase == corev1.PodRunning
		})
		framework.RemovePod(kubeClient, pod.Namespace, pod.Name)
		framework.WaitPodDisappearOnCluster(kubeClient, pod.Namespace, pod.Name)
	})

	ginkgo.It(fmt.Sprintf("Expand SFS Trubo[%s] testing", shareType), func() {
		pod := newPodWithPvc(pvc)
		framework.CreatePod(kubeClient, pod)
		framework.WaitPodPresentOnClusterFitWith(kubeClient, pod.Namespace, pod.Name, func(pod *corev1.Pod) bool {
			return pod.Status.Phase == corev1.PodRunning
		})

		pvc = framework.GetPVC(kubeClient, pvc.Namespace, pvc.Name)

		newStorage := int64(600 * common.GbByteSize)
		pvc.Spec.Resources.Requests = corev1.ResourceList{
			"storage": *resource.NewQuantity(newStorage, resource.BinarySI),
		}

		framework.UpdatePVC(kubeClient, pvc)
		framework.WaitPVCPresentOnClusterFitWith(kubeClient, pvc.Namespace, pvc.Name,
			func(pvc *corev1.PersistentVolumeClaim) bool {
				return pvc.Status.Capacity.Storage().Value() == newStorage
			},
		)
		framework.RemovePod(kubeClient, pod.Namespace, pod.Name)
		framework.WaitPodDisappearOnCluster(kubeClient, pod.Namespace, pod.Name)
	})
}

func createSC(shareType string) *storageV1.StorageClass {
	scName := storageClassNamePrefix + rand.String(RandomStrLength)
	provisioner := "sfsturbo.csi.huaweicloud.com"
	parameters := map[string]string{
		"shareType": shareType,
	}
	sc := helper.NewStorageClass(testNamespace, scName, provisioner, parameters)
	framework.CreateStorageClass(kubeClient, sc)
	return sc
}

func createPvc(sc *storageV1.StorageClass) *corev1.PersistentVolumeClaim {
	pvc := helper.NewPVC(
		testNamespace,
		pvcNamePrefix+rand.String(RandomStrLength),
		sc.Name,
		corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"storage": *resource.NewQuantity(500*common.GbByteSize, resource.BinarySI),
			},
		},
		corev1.ReadWriteMany,
	)
	framework.CreatePVC(kubeClient, pvc)
	return pvc
}

func newPodWithPvc(pvc *corev1.PersistentVolumeClaim) *corev1.Pod {
	volumeMountName := "csi-data"
	pod := helper.NewPod(testNamespace, podNamePrefix+rand.String(RandomStrLength))
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: volumeMountName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
					ReadOnly:  true,
				},
			},
		},
	}

	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
		Name:      volumeMountName,
		MountPath: "/var/lib/www/html",
	}}

	return pod
}
