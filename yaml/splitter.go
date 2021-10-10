package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	yamlParser "gopkg.in/yaml.v2"
	"strings"
)

func SplitMultiDocumentYAML(multidoc string) (documents []string, err error) {
	content := []byte(multidoc)

	contentSplit := bytes.Split(content, []byte(yamlSeparator))

	//set the maxCapacity using the size of the largest element
	var maxCapacity = bufio.MaxScanTokenSize
	for _, element := range contentSplit {
		if len(element) >= maxCapacity {
			maxCapacity = len(element) + 100
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))

	//increase the buffer token size if file is over the default token size
	if maxCapacity > bufio.MaxScanTokenSize {
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
	}

	scanner.Split(splitYAMLDocument)

	for scanner.Scan() {
		document := strings.TrimSpace(scanner.Text())

		// attempt to parse the document as yaml
		rawYamlParsed := &map[string]interface{}{}
		err := yamlParser.Unmarshal([]byte(document), rawYamlParsed)
		if err != nil {
			return documents, fmt.Errorf("Error parsing yaml document: %v\n%v", err, document)
		}

		// skip empty yaml documents
		if len(*rawYamlParsed) == 0 {
			continue
		}

		// save to our collection of docs
		documents = append(documents, document)
	}

	return documents, nil
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
