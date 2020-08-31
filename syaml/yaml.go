package syaml

import (
	"fmt"
	"github.com/tidwall/sjson"
	"sigs.k8s.io/yaml"
)

// SetBytes accepts a YAML body, a path and a new value, and updates the
// specific key in the YAML body using the path.
//
// e.g. SetBytes([]byte("name: testing\n"), "name", "new name") would would
// return "name: newname\n"
func SetBytes(y []byte, path string, value interface{}) ([]byte, error) {
	j, err := yaml.YAMLToJSON(y)
	fmt.Println("in SetBytes, after yamlToJson")
	fmt.Println(string(j))
	if err != nil {
		return nil, err
	}
	updated, err := sjson.SetBytes(j, path, value)
	if err != nil {
		return nil, err
	}
	fmt.Println("in SetBytes, after SetBytes")
	fmt.Println(string(updated))
	
	return yaml.JSONToYAML(updated)
}
