[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# WORK IN PROGRESS

We are in the process of enabling this repo for community contribution. See wiki [here](https://open-cluster-management.io/concepts/architecture/).

# Open cluster management - Search collector

The `search-collector` is part of the search component in Open Cluster Management. The [search feature spec](https://github.com/stolostron/search/blob/main/feature-spec/search.md) has an overview of all the search components.

This process targets any kubernetes cluster to collect data about its configuration resources and computes relationships between those resources. Then this data is sent to the [search-aggregator](https://github.com/stolostron/search-aggregator), where it gets indexed in graph format.

## Data model

See the [Data model documentation](https://github.com/stolostron/search-collector/blob/pkg/transforms/README.md) for more information.

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

### Dev Preview (Search Configurable Collection)

Control the Kubernetes resources that get collected from the cluster by referencing an allow and deny list within a configmap with the name search-collector-config. Create the configmap following the format in the sample template below:


```yaml
apiVersion: v1
kind: ConfigMap
metadata:
 name: search-collector-config
 namespace: <namespace where search-collector add-on is deployed>
data:
 AllowedResources: |-
   - apiGroups:
       - "*"
     resources:
       - services
       - pods
   - apiGroups:
       - admission.k8s.io
       - authentication.k8s.io
     resources:
       - "*"
 DeniedResources: |-
   - apiGroups:
       - "*"
     resources:
       - secrets
   - apiGroups:
       - admission.k8s.io
     resources:
       - policies
       - iampolicies
       - certificatepolicies
```
Steps to create search-collector-config

1. The **name** of the ConfigMap must be `search-collector-config`.

2. **namespace** is the Namespace where the Search-Collector add-on is deployed.

3. Under **data** define `AllowedResources` and `DeniedResources` as key value pairs wrapped in a string block with `|-` to preserve linebreaks.

    - The asterisk `"*"` represents <i>all</i>.

    - For resources that don't have apigroups, you should replace the `apiGroups` value with an empty string `""`.  You can check which resources don't have apigroups with `oc api-resources -o wide`
    - If the same resources are featured in both lists, they will be excluded.
4. Once you save your changes you can apply your changes by running `oc apply -f configMapFile.yaml`

5. Restart the Search-Collector pod.

### Contribution

When you make contributions to this project, fork the project then merge your changes by creating a pull request from your personal fork to the main project:

1. Fork the `search-collector` repository.
2. Clone your fork.
3. Set upstream origin with the following command: `git remote add upstream git@github.com:stolostron/search-collector.git`
4. Make your edits. Test locally with `make test` before creating a pull request to ensure smooth merging :smile:
5. Push your new commits to your personal fork with the following command: `git push origin`
6. Create a pull request from your personal fork again the upstream `search-collector` main branch

Rebuild: 2022-07-05
