package lease

import (
	"os"
	"time"

	"github.com/golang/glog"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LeaseReconciler reconciles a Secret object
type LeaseReconciler struct {
	KubeClient           kubernetes.Interface
	LeaseName            string
	LeaseDurationSeconds int32
	componentNamespace   string
}

func (r *LeaseReconciler) Reconcile() {
	if len(r.componentNamespace) == 0 {
		r.componentNamespace = getPodNamespace()
	}
	lease, err := r.KubeClient.CoordinationV1().Leases(r.componentNamespace).Get(r.LeaseName, metav1.GetOptions{})

	switch {
	case errors.IsNotFound(err):
		// create lease
		lease := &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.LeaseName,
				Namespace: r.componentNamespace,
			},
			Spec: coordinationv1.LeaseSpec{
				LeaseDurationSeconds: &r.LeaseDurationSeconds,
				RenewTime: &metav1.MicroTime{
					Time: time.Now(),
				},
			},
		}
		if _, err := r.KubeClient.CoordinationV1().Leases(r.componentNamespace).Create(lease); err != nil {
			glog.Errorf("Unable to create addon lease %q/%q on managed cluster. error:%v",
				r.componentNamespace, r.LeaseName, err)
		} else {
			glog.Infof("Addon lease %q/%q on managed cluster created for Search", r.componentNamespace, r.LeaseName)
		}

		return
	case err != nil:
		glog.Errorf("Unable to get addon lease %q/%q on managed cluster. error:%v", r.componentNamespace, r.LeaseName, err)

		return
	default:
		// update lease
		lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
		if _, err = r.KubeClient.CoordinationV1().Leases(r.componentNamespace).Update(lease); err != nil {
			glog.Errorf("Unable to update cluster lease %q/%q on managed cluster. error:%v",
				r.componentNamespace, r.LeaseName, err)
		} else {
			glog.V(2).Infof("Addon lease %q/%q on managed cluster updated for Search", r.componentNamespace, r.LeaseName)
		}
		return
	}
}

func getPodNamespace() string {
	if collectorPodNamespace, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return collectorPodNamespace
	}
	return "open-cluster-management-agent-addon"
}
