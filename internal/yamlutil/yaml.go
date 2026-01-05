package yamlutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	"gopkg.in/yaml.v3"
)

// YAMLToJSON provides either the transcoded document, or an error if an error occurred during the transcoding process.
type YAMLToJSON interface {
	// GetDecoder returns a JSON decoder for the document, or an error if the document could not be transcoded to JSON.
	GetDecoder() (*json.Decoder, error)
}

type yamlToJSONErr struct {
	err error
}

func (y *yamlToJSONErr) GetDecoder() (*json.Decoder, error) {
	return nil, y.err
}

type yamlToJSON struct {
	data []byte
}

func (y *yamlToJSON) GetDecoder() (*json.Decoder, error) {
	return json.NewDecoder(bytes.NewReader(y.data)), nil
}

// YAMLDocumentsToJSON converts a stream of YAML documents into a sequence of JSON documents.
func YAMLDocumentsToJSON(yamlStream io.Reader) iter.Seq[YAMLToJSON] {
	return func(yield func(YAMLToJSON) bool) {
		decoder := yaml.NewDecoder(yamlStream)

		for {
			var doc any

			err := decoder.Decode(&doc)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // End of documents
				}

				err = fmt.Errorf("failed to decode YAML document: %w", err)
				yield(&yamlToJSONErr{err: err})

				break // We can't continue if we can't decode the document, as we won't necessarily be able to find the next one.
			}

			jsonBytes, err := json.Marshal(doc)
			if err != nil {
				err = fmt.Errorf("failed to marshal YAML document to JSON: %w", err)
				if !yield(&yamlToJSONErr{err: err}) {
					break // Stop iteration if yield returns false
				}

				continue // Continue to the next document even if we can't marshal this one to JSON
			}

			if !yield(&yamlToJSON{data: jsonBytes}) {
				break // Stop iteration if yield returns false
			}
		}
	}
}
