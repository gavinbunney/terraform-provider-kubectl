package yaml

import (
	"encoding/json"
	"github.com/icza/dyno"
	yamlParser "gopkg.in/yaml.v2"
	meta_v1_unstruct "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
)

// ParseYAML parses a yaml string into an Manifest.
//
// To make things play nice we need the JSON representation of the object as the `RawObj`
// 1. UnMarshal YAML into map
// 2. Marshal map into JSON
// 3. UnMarshal JSON into the Unstructured type so we get some K8s checking
func ParseYAML(yaml string) (*Manifest, error) {
	rawYamlParsed := &map[string]interface{}{}
	err := yamlParser.Unmarshal([]byte(yaml), rawYamlParsed)
	if err != nil {
		return nil, err
	}

	rawJSON, err := json.Marshal(dyno.ConvertMapI2MapS(*rawYamlParsed))
	if err != nil {
		return nil, err
	}

	unstruct := meta_v1_unstruct.Unstructured{}
	err = unstruct.UnmarshalJSON(rawJSON)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Raw: &unstruct,
	}

	log.Printf("[DEBUG] %s Unstructed YAML: %+v\n", manifest, manifest.Raw.UnstructuredContent())
	return manifest, nil
}
