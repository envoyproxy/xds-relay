package stringify

import "encoding/json"

// Converts an interface to a human-readable string for pretty-printing via marshalling to JSON.
func InterfaceToString(v interface{}) (string, error) {
	prettyJSON, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(prettyJSON), nil
}
