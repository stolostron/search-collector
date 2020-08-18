## Informers

This package provides similar functionality to the informers in the client-go library, but optimizing to reduce memory consumption. The search-collector needs to watch every resource in the cluster, but we don't the full yaml. Kubernetes resources can contain up to 2MB of data, so this is way too much data for us to cache. And we don't use it anyways.

This package is inspired by the [informers on the Kubernetes client-go library.](https://github.com/kubernetes/client-go/tree/master/informers)

The client-go informers are built around the idea that the current revision of each resource is cached locally. So modifying the existing library to remove the cache doesn't seem plausible.