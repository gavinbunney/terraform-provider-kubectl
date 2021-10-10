package yaml

import (
	"fmt"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

type UnstructuredManifest struct {
	Raw *meta_v1_unstruct.Unstructured
}

func NewFromUnstructured(raw *meta_v1_unstruct.Unstructured) *UnstructuredManifest {
	return &UnstructuredManifest{
		Raw: raw,
	}
}

func (m *UnstructuredManifest) GetAPIVersion() string {
	return m.Raw.GetAPIVersion()
}

func (m *UnstructuredManifest) GetKind() string {
	return m.Raw.GetKind()
}

func (m *UnstructuredManifest) GetName() string {
	return m.Raw.GetName()
}

func (m *UnstructuredManifest) GetNamespace() string {
	return m.Raw.GetNamespace()
}

func (m *UnstructuredManifest) SetNamespace(namespace string) {
	m.Raw.SetNamespace(namespace)
}

func (m *UnstructuredManifest) HasNamespace() bool {
	return m.Raw.GetNamespace() != ""
}

func (m *UnstructuredManifest) GetUID() string {
	return fmt.Sprintf("%v", m.Raw.GetUID())
}

func (m *UnstructuredManifest) String() string {
	if m.HasNamespace() {
		return fmt.Sprintf("%s/%s", m.Raw.GetNamespace(), m.Raw.GetName())
	}

	return m.Raw.GetName()
}

func (m *UnstructuredManifest) GetSelfLink() string {
	selfLink := m.Raw.GetSelfLink()
	if len(selfLink) > 0 {
		return selfLink
	}

	return buildSelfLink(m.GetAPIVersion(), m.GetNamespace(), m.GetKind(), m.GetName())
}

// buildSelfLink creates a selfLink of the form:
//     "/apis/<apiVersion>/namespaces/<namespace>/<kind>s/<name>"
//
// The selfLink attribute is not available in Kubernetes 1.20+ so we need
// to generate a consistent, unique ID for our Terraform resources.
func buildSelfLink(apiVersion string, namespace string, kind string, name string) string {
	var linkBuilder strings.Builder

	// for any v1 api served objects, they used to be served from /api
	// all others are served from /apis
	if apiVersion == "v1" {
		linkBuilder.WriteString("/api")
	} else {
		linkBuilder.WriteString("/apis")
	}

	if len(apiVersion) != 0 {
		_, _ = fmt.Fprintf(&linkBuilder, "/%s", apiVersion)
	}

	if len(namespace) != 0 {
		_, _ = fmt.Fprintf(&linkBuilder, "/namespaces/%s", namespace)
	}

	if len(kind) != 0 {
		var suffix string
		if strings.HasSuffix(kind, "s") {
			suffix = "es"
		} else {
			suffix = "s"
		}
		_, _ = fmt.Fprintf(&linkBuilder, "/%s%s", strings.ToLower(kind), suffix)
	}

	if len(name) != 0 {
		_, _ = fmt.Fprintf(&linkBuilder, "/%s", name)
	}
	return linkBuilder.String()
}
