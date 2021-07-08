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
	HubKubeClient        kubernetes.Interface
	KubeClient           kubernetes.Interface
	LeaseName            string
	LeaseDurationSeconds int32
	ClusterName          string
	componentNamespace   string
}

func (r *LeaseReconciler) Reconcile() {
	if len(r.componentNamespace) == 0 {
		r.componentNamespace = getPodNamespace()
	}
	// Create/update lease on managed cluster first. If it fails, it could mean lease resource kind
	// is not supported on the managed cluster. Create/update lease on the hub then.
	err := r.updateLease(r.componentNamespace, r.KubeClient)

	if err != nil {
		// Try to create or update the lease on in the managed cluster's namespace on the hub cluster.
		if errors.IsNotFound(err) && r.HubKubeClient != nil {
			glog.V(2).Infof("Trying to update lease on the hub")

			if err := r.updateLease(r.ClusterName, r.HubKubeClient); err != nil {
				glog.Errorf("Failed to update lease %s/%s: %v on hub cluster", r.LeaseName, r.ClusterName, err)
			}
		} else {
			glog.Errorf("Failed to update lease %s/%s: %v on managed cluster", r.LeaseName, r.componentNamespace, err)
		}
	}
	glog.Info("Creating lease in hub default namespace")
	if err := r.updateLease("default", r.HubKubeClient); err != nil {
		glog.Errorf("Failed to update lease %s/%s: %v on hub cluster", r.LeaseName, r.ClusterName, err)
	}
}

func (r *LeaseReconciler) updateLease(namespace string, client kubernetes.Interface) error {
	glog.V(2).Infof("Trying to update lease %q/%q", namespace, r.LeaseName)

	lease, err := client.CoordinationV1().Leases(namespace).Get(r.LeaseName, metav1.GetOptions{})

	switch {
	case errors.IsNotFound(err):
		// create lease
		lease := &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.LeaseName,
				Namespace: namespace,
			},
			Spec: coordinationv1.LeaseSpec{
				LeaseDurationSeconds: &r.LeaseDurationSeconds,
				RenewTime: &metav1.MicroTime{
					Time: time.Now(),
				},
			},
		}
		if _, err := client.CoordinationV1().Leases(namespace).Create(lease); err != nil {
			glog.Errorf("Unable to create addon lease %q/%q . error:%v", namespace, r.LeaseName, err)

			return err
		}

		glog.V(2).Infof("Addon lease %q/%q created", namespace, r.LeaseName)

		return nil
	case err != nil:
		glog.Errorf("Unable to get addon lease %q/%q . error:%v", namespace, r.LeaseName, err)

		return err
	default:
		// update lease
		lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
		if _, err = client.CoordinationV1().Leases(namespace).Update(lease); err != nil {
			glog.Errorf("Unable to update cluster lease %q/%q . error:%v", namespace, r.LeaseName, err)

			return err
		}

		glog.V(2).Infof("Addon lease %q/%q updated", namespace, r.LeaseName)

		return nil
	}
}

func getPodNamespace() string {
	if collectorPodNamespace, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return collectorPodNamespace
	}
	return "open-cluster-management-agent-addon"
}
