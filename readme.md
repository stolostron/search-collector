# search-collector

This process runs on remotely managed clusters and collects data about the resources which exist there and how those resources relate to each other, then sends that data back to the hub cluster. In conjunction with the [aggregator](https://github.ibm.com/IBMPrivateCloud/search-aggregator), provides backend functionality supporting the [search features of ICP](https://github.ibm.com/IBMPrivateCloud/roadmap/blob/master/feature-specs/hcm/search/search.md).


## Config and Usage
- The application can read from a json config file, and from environment variables. If both provide a value for a specific property, the environment variable will override the file.
- The application can take any flags for [glog](https://github.com/golang/glog), it will pass them straight into glog. The glog flag `--logtostderr` is set to true by default.

### Running Locally
1. Fetch Dependencies: `make deps`
2. Build Binary: `make build`
3. Configure `~/.kube/config` to point to a cluster. Or, set the `KUBECONFIG` environment variable to some other kubernetes config file.
4. `./output/search-collector [-c alternate config file] [any glog flags]`
	- Or run with [glc](https://github.ibm.com/Ethan-Swartzentruber/GlogColor): `./output/search-collector [-c alternate config file] [any glog flags] 2>&1 | glc`

### Running on a cluster (by pushing to scratch repo)
1. Export your github username (w3id) and personal access token as `GITHUB_USER` and `GITHUB_TOKEN`
2. Fetch build harness: `make init`
3. Build and tag image: `make build-linux docker:build docker:tag-arch`
4. Export your `ARTIFACTORY_USER` and `ARTIFACTORY_TOKEN`
5. Push to scratch repo in Artifactory `make docker:login docker:push-arch`

### Config File
A default config file for local development is provided in this repo. The file assumes the [aggregator](https://github.ibm.com/IBMPrivateCloud/search-aggregator) to be running at `http://localhost:3010`. 

You can also make your own config file, defining the following values, and pass your config file to the application with `-c <config file>`. 

```
{
    "AggregatorURL":<Aggregator URL, includes protocol and port but no path>,
    "ClusterName":<name of cluster>
}
```

### Environment Variables
Instead of using a config file, you may set config properties using the following environment variables. You may also use both, the environment variables will override the values in the file.

| Config Property  | Environment Variable |
| ------------- | ------------- |
| AggregatorURL  | AGGREGATOR_URL  |
| ClusterName  |  CLUSTER_NAME  |
