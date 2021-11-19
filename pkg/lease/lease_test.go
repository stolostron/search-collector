package lease

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	AddonName            = "search-collector"
	LeaseDurationSeconds = 60
	namespace            = "open-cluster-management-agent-addon"
)

func TestCreateLeaseAddon(t *testing.T) {
	client := fake.NewSimpleClientset()

	leaseReconciler := LeaseReconciler{
		LocalKubeClient:      client,
		LeaseName:            AddonName,
		LeaseDurationSeconds: int32(LeaseDurationSeconds),
	}
	leaseReconciler.Reconcile()
	lease, err := client.CoordinationV1().Leases(namespace).Get(AddonName, metav1.GetOptions{})

	assert.Equal(t, lease.Name, AddonName, "Expected created lease to have name search-collector: Got %s", lease.Name)
	assert.Nil(t, err, "Expected no error: Got %v", err)

}

func TestUpdateLeaseAddon(t *testing.T) {
	client := fake.NewSimpleClientset()

	leaseReconciler := LeaseReconciler{
		LocalKubeClient:      client,
		LeaseName:            AddonName,
		LeaseDurationSeconds: int32(LeaseDurationSeconds),
	}
	leaseReconciler.Reconcile()
	lease, err := client.CoordinationV1().Leases(namespace).Get(AddonName, metav1.GetOptions{})

	assert.Equal(t, lease.Name, AddonName, "Expected created lease to have name search-collector: Got %s", lease.Name)
	assert.Nil(t, err, "Expected no error: Got %v", err)
	createdTime := lease.Spec.RenewTime
	leaseReconciler.Reconcile()
	lease, err = client.CoordinationV1().Leases(namespace).Get(AddonName, metav1.GetOptions{})
	assert.Nil(t, err, "Expected no error: Got %v", err)

	updatedTime := lease.Spec.RenewTime
	assert.True(t, createdTime.Before(updatedTime), "Expected lease renewtime to be updated and 'true' value to be returned. Got %b.", createdTime.Before(updatedTime))

}
