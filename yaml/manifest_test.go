package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSelfLink(t *testing.T) {
	// general case
	link := buildSelfLink("v1", "ns", "kind", "name")
	assert.Equal(t, link, "/api/v1/namespaces/ns/kinds/name")
	// no-namespace case
	link = buildSelfLink("v1", "", "kind", "name")
	assert.Equal(t, link, "/api/v1/kinds/name")
	// plural kind adds 'es'
	link = buildSelfLink("v1", "ns", "kinds", "name")
	assert.Equal(t, link, "/api/v1/namespaces/ns/kindses/name")
	link = buildSelfLink("apps/v1", "ns", "Deployment", "name")
	assert.Equal(t, link, "/apis/apps/v1/namespaces/ns/deployments/name")
}
