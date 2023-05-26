# Copyright Contributors to the Open Cluster Management project

FROM registry.ci.openshift.org/stolostron/builder:go1.20-linux AS builder

WORKDIR /go/src/github.com/stolostron/search-collector
COPY . .
RUN CGO_ENABLED=0 GOGC=25 go build -trimpath -o main main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

ARG VCS_REF
ARG VCS_URL
ARG IMAGE_NAME
ARG IMAGE_DESCRIPTION
ARG IMAGE_DISPLAY_NAME
ARG IMAGE_NAME_ARCH
ARG IMAGE_MAINTAINER
ARG IMAGE_VENDOR
ARG IMAGE_VERSION
ARG IMAGE_RELEASE
ARG IMAGE_SUMMARY
ARG IMAGE_OPENSHIFT_TAGS

LABEL org.label-schema.vendor="Red Hat" \
      org.label-schema.name="$IMAGE_NAME_ARCH" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Red Hat Advanced Cluster Management for Kubernetes EULA" \
      org.label-schema.schema-version="1.0" \
      name="$IMAGE_NAME" \
      maintainer="$IMAGE_MAINTAINER" \
      vendor="$IMAGE_VENDOR" \
      version="$IMAGE_VERSION" \
      release="$IMAGE_RELEASE" \
      description="$IMAGE_DESCRIPTION" \
      summary="$IMAGE_SUMMARY" \
      io.k8s.display-name="$IMAGE_DISPLAY_NAME" \
      io.k8s.description="$IMAGE_DESCRIPTION" \
      io.openshift.tags="$IMAGE_OPENSHIFT_TAGS"

RUN microdnf update &&\
    microdnf install ca-certificates vi --nodocs &&\
    mkdir /licenses &&\
    microdnf clean all

COPY --from=builder /go/src/github.com/stolostron/search-collector/main /bin/main

ENV VCS_REF="$VCS_REF" \
    USER_UID=1001 \
    GOGC=25

USER ${USER_UID}
ENTRYPOINT ["/bin/main"]
