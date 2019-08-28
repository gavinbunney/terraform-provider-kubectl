package kubernetes

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	yamlParser "gopkg.in/yaml.v2"
	"strings"
)

func dataSourceKubectlFileDocuments() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubectlFileDocumentsRead,
		Schema: map[string]*schema.Schema{
			"content": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"documents": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceKubectlFileDocumentsRead(d *schema.ResourceData, m interface{}) error {
	content := []byte(d.Get("content").(string))

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Split(splitYAMLDocument)

	var documents []string
	for scanner.Scan() {
		document := strings.TrimSpace(scanner.Text())

		// attempt to parse the document as yaml
		rawYamlParsed := &map[string]interface{}{}
		err := yamlParser.Unmarshal([]byte(document), rawYamlParsed)
		if err != nil {
			return fmt.Errorf("Error parsing yaml document: %v\n%v", err, document)
		}

		// skip empty yaml documents
		if len(*rawYamlParsed) == 0 {
			continue
		}

		// save to our collection of docs
		documents = append(documents, document)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256(content)))
	d.Set("documents", documents)
	return nil
}

const yamlSeparator = "\n---"

// splitYAMLDocument is a bufio.SplitFunc for splitting YAML streams into individual documents.
func splitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
