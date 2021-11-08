[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# WORK IN PROGRESS

We are in the process of enabling this repo for community contribution. See wiki [here](https://open-cluster-management.io/concepts/architecture/).

# Open cluster management - Search collector

The `search-collector` is part of the search component in Open Cluster Management. The [search feature spec](https://github.com/open-cluster-management/search/blob/main/feature-spec/search.md) has an overview of all the search components.

This process targets any kubernetes cluster to collect data about its configuration resources and computes relationships between those resources. Then this data is sent to the [search-aggregator](https://github.com/open-cluster-management/search-aggregator), where it gets indexed in graph format.

## Data model

See the [Data model documentation](https://github.com/open-cluster-management/search-collector/blob/pkg/transforms/README.md) for more information.

## Usage and configuration

### Running Locally

> **Pre-requisite:** Go v1.15

1. Fetch Dependencies: `make deps`
    > **TIP 1:** You may need to install mercurial. `brew install mercurial`
    >
    > **TIP 2:** You may need to configure git to use SSH. Use the following command:
    >
    > `git config --global --add url."git@github.com:".insteadOf "https://github.com/"`
2. Log into your development cluster with `oc login ...`.
    > **Alternative:** set the `KUBECONFIG` environment variable to some other kubernetes config file.
3. Run the program with `make run` or `go run main.go`

### Running on a cluster

- Pull image from Quay.io and deploy to your cluster.
- To test code changes, create a PR in this repo, the build process pushes the PR image to Quiay.io. From there you can pull it into your cluster.

### Environment Variables

Control the behavior of this service with these environment variables. For development these could be set on ./config.json

Name               | Required | Default Value            | Description
----               | -------- | -------------            | -----------
AGGREGATOR_URL     | yes      | <https://localhost:3010> | Deprecated. Use host + port instead.
AGGREGATOR_HOST    | yes      | <https://localhost>      | Location of the aggregator service.
AGGREGATOR_PORT    | yes      | 3010                     |
CLUSTER_NAME       | yes      | local-cluster            | Name of cluster where this collector is running.
HEARTBEAT_MS       | no       | 300000  // 5 min         | Interval(ms) to send empty payload to ensure connection
MAX_BACKOFF_MS     | no       | 600000  // 10 min        | Maximum backoff in ms to wait after send error
REDISCOVER_RATE_MS | no       | 120000  // 2 min         | Interval(ms) to poll for changes to CRDs
REPORT_RATE_MS     | no       | 5000    // 5 seconds     | Interval(ms) to queue changes before sending to the aggregator
RUNTIME_MODE       | no       | production               | Running mode (development or production)

### Other Configuration Options

- Environment variables can also be set in the `./config.json` for development. If both provide a value for a specific property, the environment variable overrides the file. You can define your own `config.json` file and pass it to the application with the following command: `-c <config_file>`
- The application can take any flags for [glog](https://github.com/golang/glog), which passes them straight into glog. The glog flag `--logtostderr` is set to true by default.

### Contribution

When you make contributions to this project, fork the project then merge your changes by creating a pull request from your personal fork to the main project:

1. Fork the `search-collector` repository.
2. Clone your fork.
3. Set upstream origin with the following command: `git remote add upstream git@github.com:open-cluster-management/search-collector.git`
4. Make your edits. Test locally with `make test` before creating a pull request to ensure smooth merging :smile:
5. Push your new commits to your personal fork with the following command: `git push origin`
6. Create a pull request from your personal fork again the upstream `search-collector` main branch

Rebuild: 2021-11-05
