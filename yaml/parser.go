package yaml

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"

	"github.com/icza/dyno"
	yamlParser "gopkg.in/yaml.v2"
	metav1unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseYAML parses a yaml string into an Manifest.
//
// To make things play nice we need the JSON representation of the object as the `RawObj`
// 1. UnMarshal YAML into map
// 2. Marshal map into JSON
// 3. UnMarshal JSON into the Unstructured type so we get some K8s checking
func ParseYAML(yaml string) (*Manifest, error) {
	var manifests []*Manifest

	dec := yamlParser.NewDecoder(strings.NewReader(yaml))

	for {
		rawYamlParsed := &map[string]interface{}{}
		err := dec.Decode(rawYamlParsed)

		if rawYamlParsed == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		rawJSON, err := json.Marshal(dyno.ConvertMapI2MapS(*rawYamlParsed))
		if err != nil {
			return nil, err
		}

		unstruct := metav1unstruct.Unstructured{}
		err = unstruct.UnmarshalJSON(rawJSON)
		if err != nil {
			return nil, err
		}

		manifest := &Manifest{
			Raw: &unstruct,
		}

		manifests = append(manifests, manifest)

		log.Printf("[DEBUG] %s Unstructed YAML: %+v\n", manifest, manifest.Raw.UnstructuredContent())
	}

	if len(manifests) == 0 {
		return nil, errors.New("no documents found in YAML")
	}

	if len(manifests) > 1 {
		return nil, errors.New("multiple documents found in YAML, split them up with the `kubectl_file_documents` data source")
	}

	return manifests[0], nil
}
