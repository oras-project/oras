package json

import (
	"encoding/json"
	"os"
)

func printJSON(object any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}
