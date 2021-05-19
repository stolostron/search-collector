package lease

import (
	"time"

	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	agentLabel = map[string]string{"app": "search"}
)

// LeaseReconciler reconciles a Secret object
type LeaseReconciler struct {
	KubeClient           kubernetes.Interface
	LeaseName            string
	LeaseDurationSeconds int32
	componentNamespace   string
}

func (r *LeaseReconciler) Reconcile() {
	glog.Info("In lease Reconcile")
	if len(r.componentNamespace) == 0 {
		r.componentNamespace = config.Cfg.ClusterNamespace
	}
	glog.Info("Cluster namespace is ", r.componentNamespace)
	lease, err := config.GetKubeClient().CoordinationV1().Leases(r.componentNamespace).Get(r.LeaseName, metav1.GetOptions{})

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
			glog.Errorf("unable to create addon lease %q/%q on local managed cluster. error:%v", r.componentNamespace, r.LeaseName, err)
		} else {
			glog.Infof("addon lease %q/%q on local managed cluster created", r.componentNamespace, r.LeaseName)
		}

		return
	case err != nil:
		glog.Errorf("unable to get addon lease %q/%q on local managed cluster. error:%v", r.componentNamespace, r.LeaseName, err)

		return
	default:
		// update lease
		lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
		if _, err = r.KubeClient.CoordinationV1().Leases(r.componentNamespace).Update(lease); err != nil {
			glog.Errorf("unable to update cluster lease %q/%q on local managed cluster. error:%v", r.componentNamespace, r.LeaseName, err)
		} else {
			glog.Infof("addon lease %q/%q on local managed cluster updated", r.componentNamespace, r.LeaseName)
		}
		glog.Info("Exiting lease Reconcile")

		return
	}
}
