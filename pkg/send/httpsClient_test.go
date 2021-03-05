// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package send

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func Test_getHttpsClient(t *testing.T) {
	client := getHTTPSClient()

	assert.NotNil(t, client, "Should get a valid https client")
}
