/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

type PersistentVolumeResource struct {
	*v1.PersistentVolume
}

func (p PersistentVolumeResource) BuildNode() Node {
	node := transformCommon(p)         // Start off with the common properties
	apiGroupVersion(p.TypeMeta, &node) // add kind, apigroup and version
	// Extract the properties specific to this type
	node.Properties["reclaimPolicy"] = string(p.Spec.PersistentVolumeReclaimPolicy)
	node.Properties["status"] = string(p.Status.Phase)
	node.Properties["type"] = getType(&p.Spec)

	node.Properties["capacity"] = ""
	storage, ok := p.Spec.Capacity["storage"]
	if ok {
		node.Properties["capacity"] = storage.String()
	}

	// can't cast []PersistentVolumeAccessMode to []string without unsafe
	accessModes := make([]string, len(p.Spec.AccessModes))
	for i := 0; i < len(p.Spec.AccessModes); i++ {
		accessModes[i] = string(p.Spec.AccessModes[i])
	}
	node.Properties["accessMode"] = accessModes

	node.Properties["claimRef"] = ""
	if p.Spec.ClaimRef != nil {
		claimRefNamespace := p.Spec.ClaimRef.Namespace
		claimRefName := p.Spec.ClaimRef.Name
		if claimRefNamespace != "" && claimRefName != "" {
			s := []string{claimRefNamespace, claimRefName}
			node.Properties["claimRef"] = strings.Join(s, "/")
		}
	}

	if p.Spec.Local != nil {
		node.Properties["path"] = p.Spec.Local.Path
	} else if p.Spec.HostPath != nil {
		node.Properties["path"] = p.Spec.HostPath.Path
	} else if p.Spec.Glusterfs != nil {
		node.Properties["path"] = p.Spec.Glusterfs.Path
	} else if p.Spec.NFS != nil {
		node.Properties["path"] = p.Spec.NFS.Path
	} else if p.Spec.VsphereVolume != nil {
		node.Properties["path"] = p.Spec.VsphereVolume.VolumePath
	}

	return node
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

func (p PersistentVolumeResource) BuildEdges(ns NodeStore) []Edge {
	//no op for now to implement interface
	return []Edge{}
}
