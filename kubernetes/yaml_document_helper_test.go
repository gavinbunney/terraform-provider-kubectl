package kubernetes

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLDocumentHelper(t *testing.T) {
	testCases := []struct {
		description  string
		yaml         string
		expectedDocs []string
	}{
		{
			description:  "Test single document",
			yaml:         buildTestData(1),
			expectedDocs: []string{"kind: Service1"},
		},
		{
			description:  "Test multi document",
			yaml:         buildTestData(2),
			expectedDocs: []string{"kind: Service1", "kind: Service2"},
		},
		{
			description: "Test multi document with empty document at end",
			yaml: buildTestData(2) + `
---
# just a comment
---
`,
			expectedDocs: []string{"kind: Service1", "kind: Service2"},
		},
		{
			description: "Test multi document with empty document at start",
			yaml: `
---
# just a comment
---
` + buildTestData(2),
			expectedDocs: []string{"kind: Service1", "kind: Service2"},
		},
		{
			description: "Test multi document with only empty documents",
			yaml: `
---
# just a comment
---
# more empty docs
---
`,
			expectedDocs: nil,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			result, err := splitMultiDocumentYAML(tcase.yaml)
			assert.NoError(t, err, "Expect to succeed")
			assert.Equal(t, len(tcase.expectedDocs), len(result), "Expect docs count to match")
			assert.Equal(t, tcase.expectedDocs, result, "Expect docs to match")
		})
	}
}

func TestYAMLDocumentHelperReadLargeFile(t *testing.T) {
	testCases := []struct {
		description  string
		yaml         string
		expectedDocs string
	}{
		{
			description:  "Test processing large file",
			yaml:         readTestFile(),
			expectedDocs: "storage: true",
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			result, err := splitMultiDocumentYAML(tcase.yaml)
			assert.NoError(t, err, "Expect to succeed")
			assert.Equal(t, 6, len(result), "Expect docs count to match")
			assert.Contains(t, result[5], tcase.expectedDocs, "Expect docs to contain")
		})
	}
}

func buildTestData(count int) (content string) {
	for i := 1; i <= count; i++ {
		content += fmt.Sprintf("\nkind: Service%v\n---", i)
	}

	return content
}

func readTestFile() (content string) {

	path := "../_examples/cert-manager/01-cert-manager-crds.yaml"
	file, err := ioutil.ReadFile(path)
	check(err)

	return string(file)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
