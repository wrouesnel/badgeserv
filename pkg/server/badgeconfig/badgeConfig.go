package badgeconfig

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	ErrConfigLoading = errors.New("error while loading configuration")
)

type BadgeDesc struct {
	Label   string `mapstructure:"label" help:"Label template"`
	Message string `mapstructure:"message" help:"Message template"`
	Color   string `mapstructure:"color" help:"Color template"`
}

// BadgeExample defines an example of a predefined badge. It can be used to
// present common badges on the main UI. Multiple examples can be defined.
type BadgeExample struct {
	Description string `mapstructure:"description" help:"description of the badge example"`
	Parameters  map[string]string
}

type BadgeDefinition struct {
	BadgeDesc   `mapstructure:",squash"`
	Target      string            `mapstructure:"target" help:"target URL to resolve badge data from"`
	Parameters  map[string]string `mapstructure:"parameters" help:"Accepted parameters for the interface"`
	Examples    []BadgeExample    `mapstructure:"examples" help:"List of example badges to include"`
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

	errors := []error{}
	for _, configPath := range matches {
		logger.Debug("Loading predefined badges from config file", zap.String("config_path", configPath))
		configBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			logger.Warn("Could not read config file", zap.String("config_path", configPath), zap.Error(err))
			errors = append(errors, err)
			continue
		}
		config, err := Load(configBytes)
		if err != nil {
			logger.Warn("Config parsing error", zap.String("config_path", configPath), zap.Error(err))
			errors = append(errors, err)
			continue
		}
		finalConfig.PredefinedBadges = lo.Assign(finalConfig.PredefinedBadges, config.PredefinedBadges)
	}

	if len(errors) > 0 {
		return &finalConfig, ErrConfigLoading
	}

	return &finalConfig, nil
}
