#IBM Confidential
#OCO Source Materials
#5737-E67
#(C) Copyright IBM Corporation 2019 All Rights Reserved
#The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

# Release Tag
if [ "$TRAVIS_BRANCH" = "master" ]; then
    RELEASE_TAG=latest
else
    RELEASE_TAG="${TRAVIS_BRANCH#release-}-latest"
fi
if [ "$TRAVIS_TAG" != "" ]; then
    RELEASE_TAG="${TRAVIS_TAG#v}"
fi
export RELEASE_TAG="$RELEASE_TAG"

# only push to integration on a merge that is not the development branch
if [ "$TRAVIS_EVENT_TYPE" != "pull_request" ] && [ "$TRAVIS_BRANCH" != "development" ]; then
    DOCKER_REGISTRY=hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com
else
    DOCKER_REGISTRY=hyc-cloud-private-scratch-docker-local.artifactory.swg-devops.com
    GIT_COMMIT="$(git rev-parse --short HEAD)"
    RELEASE_TAG=$GIT_COMMIT
fi
export DOCKER_REGISTRY="$DOCKER_REGISTRY"

# Echo vars
echo TRAVIS_EVENT_TYPE=$TRAVIS_EVENT_TYPE
echo TRAVIS_BRANCH=$TRAVIS_BRANCH
echo TRAVIS_TAG=$TRAVIS_TAG
echo RELEASE_TAG="$RELEASE_TAG"
echo DOCKER_REGISTRY="$DOCKER_REGISTRY"
