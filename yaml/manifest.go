package yaml

import (
	"fmt"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlWriter "sigs.k8s.io/yaml"
	"strings"
)

type Manifest struct {
	Raw *meta_v1_unstruct.Unstructured
}

func NewFromUnstructured(raw *meta_v1_unstruct.Unstructured) *Manifest {
	return &Manifest{
		Raw: raw,
	}
}

func (m *Manifest) GetAPIVersion() string {
	return m.Raw.GetAPIVersion()
}

func (m *Manifest) GetKind() string {
	return m.Raw.GetKind()
}

func (m *Manifest) GetName() string {
	return m.Raw.GetName()
}

func (m *Manifest) GetNamespace() string {
	return m.Raw.GetNamespace()
}

func (m *Manifest) SetNamespace(namespace string) {
	m.Raw.SetNamespace(namespace)
}

func (m *Manifest) HasNamespace() bool {
	return m.Raw.GetNamespace() != ""
}

func (m *Manifest) GetUID() string {
	return fmt.Sprintf("%v", m.Raw.GetUID())
}

func (m *Manifest) GetSelfLink() string {
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

func (m *Manifest) String() string {
	if m.HasNamespace() {
		return fmt.Sprintf("%s/%s", m.Raw.GetNamespace(), m.Raw.GetName())
	}

	return m.Raw.GetName()
}

// AsYAML will produce a yaml representation of the manifest.
// We do this by serializing to json and back again to ensure values and comments are cleansed
func (m *Manifest) AsYAML() (string, error) {
	yamlJson, err := m.Raw.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to convert object to json: %+v", err)
	}

	yamlParsed, err := yamlWriter.JSONToYAML(yamlJson)
	if err != nil {
		return "", fmt.Errorf("failed to convert json to yaml: %+v", err)
	}

	return string(yamlParsed), nil
}
