package transforms

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

// Takes a *v1.PersistentVolume and yields a Node
func TransformPersistentVolume(resource *v1.PersistentVolume) Node {

	persistentVolume := TransformCommon(resource) // Start off with the common properties

	// Extract the properties specific to this type
	persistentVolume.Properties["kind"] = "PersistentVolume"
	persistentVolume.Properties["accessModes"] = resource.Spec.AccessModes
	persistentVolume.Properties["capacity"], _ = resource.Spec.Capacity.StorageEphemeral().AsInt64()
	persistentVolume.Properties["reclaimPolicy"] = resource.Spec.PersistentVolumeReclaimPolicy
	persistentVolume.Properties["status"] = resource.Status.Phase
	persistentVolume.Properties["type"] = getType(&resource.Spec)

	claimRefNamespace := resource.Spec.ClaimRef.Namespace
	claimRefName := resource.Spec.ClaimRef.Name
	if claimRefNamespace != "" && claimRefName != "" {
		s := []string{claimRefNamespace, claimRefName}
		persistentVolume.Properties["claimRef"] = strings.Join(s, "/")
	}

	if resource.Spec.Local != nil {
		persistentVolume.Properties["path"] = resource.Spec.Local.Path
	} else {
		persistentVolume.Properties["path"] = resource.Spec.HostPath.Path
	}

	return persistentVolume
}

func getType(spec *v1.PersistentVolumeSpec) string {
	if spec.HostPath != nil {
		return "Hostpath"
	}
	if spec.GCEPersistentDisk != nil {
		return "GCEPersistentDisk"
	}
	if spec.AWSElasticBlockStore != nil {
		return "AWSElasticBlockStore"
	}
	if spec.NFS != nil {
		return "NFS"
	}
	if spec.ISCSI != nil {
		return "iSCSI"
	}
	if spec.Glusterfs != nil {
		return "Glusterfs"
	}
	if spec.RBD != nil {
		return "RBD"
	}
	if spec.Local != nil {
		return "LocalVolume"
	}
	if spec.VsphereVolume != nil {
		return "vSphere"
	}

	return ""
}
