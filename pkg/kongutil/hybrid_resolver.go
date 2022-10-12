// package kongutil provides helper functions for working with the kong parser
package kongutil

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	ErrUnsupportedConfigFormat = errors.New("Unsupported config format")
)

// Hybrid returns a Resolver that retrieves values from a supported structured
// datasource. Currently JSON and YAML are supported.
//
// Hyphens in flag names are replaced with underscores.
func Hybrid(r io.Reader) (kong.Resolver, error) {
	values, err := decodeConfig(r)
	if err != nil {
		return nil, errors.Wrap(err, "Hybrid: configuration could not be decoded into any supported format")
	}

	var f kong.ResolverFunc = func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		name := strings.ReplaceAll(flag.Name, "-", "_")
		raw, ok := values[name]
		if ok {
			return raw, nil
		}
		raw = values
		for _, part := range strings.Split(name, ".") {
			if values, ok := raw.(map[string]interface{}); ok {
				raw, ok = values[part]
				if !ok {
					return raw, nil
				}
			} else {
				return raw, nil
			}
		}
		return raw, nil
	}

	return f, nil
}

func decodeConfig(r io.Reader) (map[string]interface{}, error) {
	values := map[string]interface{}{}

	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "decodeConfig: failed to read all config bytes")
	}

	// Attempt JSON decoding first
	err = json.Unmarshal(configBytes, &values)
	if err == nil {
		return values, nil
	}

	// Attempt YAML next
	err = yaml.Unmarshal(configBytes, &values)
	if err == nil {
		return values, nil
	}

	// Attempt TOML
	err = toml.Unmarshal(configBytes, &values)
	if err == nil {
		return values, nil
	}

	return nil, ErrUnsupportedConfigFormat
}
