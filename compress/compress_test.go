package compress

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"unicode/utf8"
)

func TestPackString(t *testing.T) {
	result, err := PackString("this is a string")
	assert.Nil(t, err)
	assert.Equal(t, "H4sIAAAAAAAE/wAQAO//dGhpcyBpcyBhIHN0cmluZwEAAP//QhhJHxAAAAA=", result)
}

func TestUnpackString(t *testing.T) {
	result, err := UnpackString("H4sIAAAAAAAE/wAQAO//dGhpcyBpcyBhIHN0cmluZwEAAP//QhhJHxAAAAA=")
	assert.Nil(t, err)
	assert.Equal(t, "this is a string", result)
}

func TestPackAndUnpackString(t *testing.T) {
	test := "this is a string"
	packed, packErr := PackString(test)
	assert.Nil(t, packErr)
	result, unpackErr := UnpackString(packed)
	assert.Nil(t, unpackErr)
	assert.Equal(t, test, result)
}

func TestPackAndUnpackManifests(t *testing.T) {
	testCases := []struct {
		description string
		manifest    string
	}{
		{
			description: "Check service account is packed and unpacked",
			manifest: `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: name-here
  namespace: default
`,
		},
		{
			description: "Check multiple yaml doc is packed and unpacked",
			manifest: `---
apiVersion: "stable.example.com/v1"
kind: CronTab
metadata:
  name: name-here-crd
spec:
  cronSpec: "* * * * /5"
  image: my-awesome-cron-image
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: name-here-crontabs.stable.example.com
spec:
  group: stable.example.com
  conversion:
    strategy: None
  scope: Namespaced
  names:
    plural: name-here-crontabs
    singular: crontab
    kind: CronTab
    shortNames:
      - ct
  version: v1
  versions:
    - name: v1
      served: true
      storage: true
---
`,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			packed, packErr := PackString(tcase.manifest)
			assert.NotEqual(t, len(packed), len(tcase.manifest))
			assert.Nil(t, packErr)
			assert.True(t, utf8.ValidString(packed), "Validate packed string is utf-8")

			unpacked, unpackErr := UnpackString(packed)
			assert.Nil(t, unpackErr)
			assert.Equal(t, tcase.manifest, unpacked, "Check manifest is restored exactly")
			assert.True(t, utf8.ValidString(unpacked), "Validate unpacked string is utf-8")
		})
	}
}