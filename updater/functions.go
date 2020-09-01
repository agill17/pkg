package updater

import (
	"github.com/agill17/pkg/syaml"
)

// ReplaceContents is a ContentUpdater that replaces the content of file with the
// provided body.
func ReplaceContents(b []byte) ContentUpdater {
	return func([]byte) ([]byte, error) {
		return b, nil
	}
}

// UpdateYAML is a ContentUpdater that updates a YAML file using a key and new
// value, they key can be a dotted path.
//
// UpdateYAML("test.value", []string{"test", "value"})
func UpdateYAML(key string, newValue interface{}) ContentUpdater {
	return func(b []byte) ([]byte, error) {
		data, err := syaml.SetBytes(b, key, newValue)
		return data, err
	}
}
