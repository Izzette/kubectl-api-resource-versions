package yamlutil_test

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/Izzette/kubectl-api-resource-versions/internal/yamlutil"
)

//go:embed testdata/documents.yaml
var document string

func ExampleYAMLDocumentsToJSON() {
	buf := bytes.NewBufferString(document)
	for result := range yamlutil.YAMLDocumentsToJSON(buf) {
		decoder, err := result.GetDecoder()
		if err != nil {
			panic(err)
		}

		var jsonDoc any
		if err := decoder.Decode(&jsonDoc); err != nil {
			panic(err)
		}

		fmt.Printf("%#v\n", jsonDoc)
	}
	//nolint:lll
	// Output:
	// []interface {}{map[string]interface {}{"key":"value"}, map[string]interface {}{"key2":"value2"}, map[string]interface {}{"key3":"value3"}}
	// map[string]interface {}{"other":"value", "with":map[string]interface {}{"different":"structure"}}
	// <nil>
}
