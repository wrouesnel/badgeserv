package badgeconfig

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"path/filepath"
)

type BadgeDesc struct {
	Label   string `mapstructure:"label" help:"Label template"`
	Message string `mapstructure:"message" help:"Message template"`
	Color   string `mapstructure:"color" help:"Color template"`
}

type BadgeDefinition struct {
	BadgeDesc   `mapstructure:",squash"`
	Target      string            `mapstructure:"target" help:"target URL to resolve badge data from"`
	Parameters  map[string]string `mapstructure:"parameters" help:"Accepted parameters for the interface"`
	Example     map[string]string `mapstructure:"example" help:"Prefilled parameters to display an example badge"`
	Description string            `mapstructure:"description"`
}

type Config struct {
	PredefinedBadges map[string]BadgeDefinition `mapstructure:"predefined_badges"`
}

// Decoder returns the decoder for config maps.
//nolint:exhaustruct
func Decoder(target interface{}, allowUnused bool) (*mapstructure.Decoder, error) {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: !allowUnused,
		DecodeHook:  mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc()),
		Result:      target,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Load: BUG - decoder configuration rejected")
	}
	return decoder, nil
}

// configMapMerge merges config maps right-to-left. Maps and nested maps
// are merged key-by-key, but lists will be replaced.
func configMapMerge(left, right map[string]interface{}) {
	for k, leftValue := range left {
		// left key does not exist in right map
		rightValue, ok := right[k]
		if !ok {
			right[k] = leftValue
			continue
		}
		// does exist - check if this is a map type on the right
		switch v := rightValue.(type) {
		case map[string]interface{}:
			// check if map on the left
			leftValueMap, ok := leftValue.(map[string]interface{})
			if !ok {
				// Not a value map on left.
				break
			}
			// map on both sides - descend and merge.
			configMapMerge(leftValueMap, v)
		default:
			// leave non-maps alone on the right.
			continue
		}
	}
}

// loadConfigMap unmarshals config bytes into the map for mapstructure.
func loadConfigMap(configBytes []byte) (map[string]interface{}, error) {
	// Load the default config to setup the defaults
	configMap := make(map[string]interface{})
	err := yaml.Unmarshal(configBytes, configMap)
	if err != nil {
		return configMap, errors.Wrapf(err, "loadConfigMap: yaml unmarshalling failed")
	}

	return configMap, nil
}

// Load loads a configuration file from the supplied bytes.
//nolint:forcetypeassert,funlen,cyclop
func Load(configData []byte) (*Config, error) {
	//defaultMap := loadDefaultConfigMap()
	configMap, err := loadConfigMap(configData)
	if err != nil {
		return nil, errors.Wrap(err, "Load: failed")
	}

	// Merge default configuration into the configMap
	//configMapMerge(defaultMap, configMap)

	// Do an initial decode to detect any unused key errors
	cfg := new(Config)
	decoder, err := Decoder(cfg, false)
	if err != nil {
		return nil, errors.Wrapf(err, "Load: config map decoder failed to initialize")
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, errors.Wrap(err, "Load: config map decoding failed")
	}

	// Do the decode after inheritance and allow unused key errors.
	cfg = new(Config)
	decoder, err = Decoder(cfg, true)
	if err != nil {
		return nil, errors.Wrapf(err, "Load: second-pass config map decoder failed to initialize")
	}

	if err := decoder.Decode(configMap); err != nil {
		return nil, errors.Wrap(err, "Load: second-pass config map decoding failed")
	}
	return cfg, nil
}

// LoadDir loads a directory of predefined badge configuration files.
func LoadDir(dirPath string) (*Config, error) {
	// Note: ick.
	logger := zap.L()

	matches := lo.FlatMap([]string{"yml", "yaml"}, func(ext string, _ int) []string {
		extMatches, _ := filepath.Glob(fmt.Sprintf("%s/*.yml", dirPath))
		return extMatches
	})

	finalConfig := Config{PredefinedBadges: map[string]BadgeDefinition{}}

	for _, configPath := range matches {
		logger.Debug("Loading predefined badges from config file", zap.String("config_path", configPath))
		configBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			logger.Warn("Could not read config file", zap.String("config_path", configPath), zap.Error(err))
			continue
		}
		config, err := Load(configBytes)
		if err != nil {
			logger.Warn("Config parsing error", zap.String("config_path", configPath), zap.Error(err))
			continue
		}
		finalConfig.PredefinedBadges = lo.Assign(finalConfig.PredefinedBadges, config.PredefinedBadges)
	}

	return &finalConfig, nil
}
