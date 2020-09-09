echo "=== TEST: Memory and CPU budget ===\n"

CPU_BUDGET=5.00
MEM_BUDGET=14.00
SEARCH_COLLECTOR_IMAGE=$1
# SEARCH_COLLECTOR_IMAGE=quay.io/open-cluster-management/search-collector:2.1.0-SNAPSHOT-2020-09-08-19-53-33
# SEARCH_COLLECTOR_IMAGE=search-collector

create_kind_cluster() { 
    WORKDIR=`pwd`
    if [[ ! -f /usr/local/bin/kind ]]; then
    	echo "Installing kind from (https://kind.sigs.k8s.io/)."
    
    	# uname returns your operating system name
    	# uname -- Print operating system name
    	# -L location, lowercase -o specify output name, uppercase -O Write  output to a local file named like the remote file we get  
    	curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.7.0/kind-$(uname)-amd64"
    	chmod +x ./kind
    	sudo cp ./kind /usr/local/bin/kind
    fi
    # Delete kind cluster collector-test if it exists
    kind delete cluster --name collector-test --quiet || true
    
    echo "Starting kind cluster: collector-test" 
    rm -rf $HOME/.kube/kind-collector-test
    kind create cluster \
        --kubeconfig $HOME/.kube/kind-collector-test \
        --name collector-test \
        --config ${WORKDIR}/tests/e2e/kind/kind-collector-test.config.yaml \
        --quiet
    export KUBECONFIG=$HOME/.kube/kind-collector-test
}

run_container() {
    docker run \
        -e CLUSTER_NAME="local-cluster" \
        -e DEPLOYED_IN_HUB="true" \
        -v $PWD/sslcert:/sslcert  \
        -v ~/.kube/kind-collector-test:/.kube/config \
        --network="host" \
        --name search-collector \
        ${SEARCH_COLLECTOR_IMAGE} &

    echo "Waiting 90s for search-collector container to start."
    sleep 90
    OUTPUT=$(docker stats --no-stream --format "{{.MemUsage}} : {{.CPUPerc}}" search-collector) 
    MEM=$(echo $OUTPUT | awk '{print $1}' | sed 's/[^0-9\.]*//g')
    CPU=$(echo $OUTPUT | awk '{print $5}' | sed 's/[^0-9\.]*//g')

    echo "Stopping and removing search-collector container."
    docker stop search-collector
    docker rm search-collector
    kind delete cluster --name collector-test --quiet
}


verify_mem_cpu() {
    TEST_FAILED=false
    if (( $(echo "$MEM > $MEM_BUDGET" |bc -l) )); then
        echo "MEMORY budget exceeded."
        echo "\tUsed:   $MEM"
        echo "\tBudget: $MEM_BUDGET"
        TEST_FAILED=true
    fi
    if (( $(echo "$CPU > $CPU_BUDGET" |bc -l) )); then
        echo "CPU budget exceeded."
        echo "\tUsed:   $CPU"
        echo "\tBudget: $CPU_BUDGET"
        TEST_FAILED=true
    fi
    if [ $TEST_FAILED == "true" ]; then
        echo "TEST FAILED."
        exit 1
    fi
}

create_kind_cluster
run_container
verify_mem_cpu


echo "\nTEST PASSED."