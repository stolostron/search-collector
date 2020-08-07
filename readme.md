# Open Cluster Management - Search Collector

The `search collector` is part of the search component in Open Cluster Management. The [search feature spec](https://github.com/open-cluster-management/search/blob/master/feature-spec/search.md) has an overview of all the search components.

This process targets any kubernetes cluster to collect data about its configuration resources and computes relationships between those resources. Then this data is sent to the [search-aggregator](https://github.com/open-cluster-management/search-aggregator), where it gets indexed in graph format.


## Data Model
The data model is documented at ./pkg/transforms/README.md


## Config and Usage
- The application can read from a json config file, and from environment variables. If both provide a value for a specific property, the environment variable will override the file.
- The application can take any flags for [glog](https://github.com/golang/glog), it will pass them straight into glog. The glog flag `--logtostderr` is set to true by default.

### Running Locally
> **Pre-requisite:** Go v1.13
1. Fetch Dependencies: `make deps`
    > **TIP 1:** You may need to install mercurial. `brew install mercurial`
    >
    > **TIP 2:** You may need to configure git to use SSH. Use the following command: 
    >
    > `git config --global --add url."git@github.com:".insteadOf "https://github.com/"`
2. Log into your development cluster with `oc login ...`.
    > **Alternative:** set the `KUBECONFIG` environment variable to some other kubernetes config file.
3. Run the program with `make run` or
    ```
    GO111MODULE=go run main.go
    ```

### Running on a cluster
- Pull image from quay.io and deploy to your cluster.
- To test code changes, create a PR in this repo, the build process will push the PR image to quiay.io. From there you can pull it into your cluster.

### Config File
A default config file for local development is provided in this repo. The file assumes the [aggregator](https://github.com/open-cluster-management/search-aggregator) to be running at `http://localhost:3010`. 

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
| -------------    | -------------        |
| AggregatorURL    | AGGREGATOR_URL       |
| ClusterName      | CLUSTER_NAME         |
