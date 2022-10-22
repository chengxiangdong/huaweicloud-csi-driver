module github.com/huaweicloud/huaweicloud-csi-driver

go 1.16

require (
	github.com/chnsz/golangsdk v0.0.0-20220927014619-cd8064513cee
	github.com/container-storage-interface/spec v1.5.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/kubernetes-csi/csi-lib-utils v0.11.0
	github.com/spf13/cobra v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.0
	github.com/unknwon/com v1.0.1
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/gcfg.v1 v1.2.3
	k8s.io/apimachinery v0.24.8-rc.0
	k8s.io/component-base v0.24.8-rc.0
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.80.1
	k8s.io/mount-utils v0.24.7
	k8s.io/utils v0.0.0-20221012122500-cfd413dd9e85
)

require (
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
