## Informers

This package is inspired by the [informers on the Kubernetes client-go library.](https://github.com/kubernetes/client-go/tree/master/informers)

It provides similar functionality to the informers in the client-go library, but optimizing to reduce memory consumption. Our search-collector needs to watch every resource in the cluster, but we don't need the full yaml. Kubernetes resources can contain up to 2 MB of data, so this is too much data for us to keep cached in memory with no use for it.

The client-go informers are built around the idea that the current revision of each resource is cached locally. So, modifying the existing library to remove the cache doesn't seem plausible.