// package entrypoint is the actual entrypoint for the command line application
package entrypoint

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/alecthomas/kong"
	gap "github.com/muesli/go-app-paths"
	"github.com/samber/lo"
	"github.com/wrouesnel/badgeserv/assets"
	"github.com/wrouesnel/badgeserv/pkg/badges"
	"github.com/wrouesnel/badgeserv/pkg/kongutil"
	"github.com/wrouesnel/badgeserv/pkg/server"
	"github.com/wrouesnel/badgeserv/version"
	"go.uber.org/zap"
)

//nolint:gochecknoglobals
var CLI struct {
	Logging struct {
		Level  string `help:"logging level" default:"info"`
		Format string `help:"logging format (${enum})" enum:"console,json" default:"json"`
	} `embed:"" prefix:"logging."`

	Assets assets.Config      `embed:"" prefix:"assets."`
	Badges badges.BadgeConfig `embed:"" prefix:"badges."`

	BadgeConfigDir string `help:"Path to the predefined badge configuration directory" type:"existingdir"`

	Debug struct {
		Assets struct {
			List struct {
			} `cmd:"" help:"list embedded files in the binary"`
			Cat struct {
				Filename string `arg:"" name:"filename" help:"embedded file to emit to stdout"`
			} `cmd:"" help:"output the specifid file to stdout"`
		} `cmd:""`
	} `cmd:""`

	API server.APIServerConfig `cmd:"" help:"Launch the web API"`
}

func configFileName(prefix string, ext string) string {
	return fmt.Sprintf("%s%s.%s", prefix, version.Name, ext)
}

func configDirListGet() ([]string, []string) {
	deferredLogs := []string{}

	// Handle a sensible configuration loader path
	scope := gap.NewScope(gap.User, version.Name)
	baseConfigDirs, err := scope.ConfigDirs()
	if err != nil {
		deferredLogs = append(deferredLogs, err.Error())
	}

	supportedFmts := []string{"json", "yml", "yaml", "toml"}

	normConfigFiles := []string{}
	for _, configDir := range baseConfigDirs {
		normConfigFiles = append(normConfigFiles, lo.Map(supportedFmts, func(ext string, _ int) string {
			return path.Join(configDir, configFileName("", ext))
		})...)
	}

	var curDirConfigFiles []string = lo.Map(supportedFmts, func(ext string, _ int) string {
		return configFileName(".", ext)
	})

	var homeDirConfigFiles []string = lo.Map(curDirConfigFiles, func(configFileName string, _ int) string {
		return path.Join(os.Getenv("HOME"), configFileName)
	})

	configFiles := curDirConfigFiles
	configFiles = append(configFiles, homeDirConfigFiles...)
	configFiles = append(configFiles, normConfigFiles...)

	return configFiles, deferredLogs
}

func Entrypoint(stdOut io.Writer, stdErr io.Writer) int {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	configDirs, deferredLogs := configDirListGet()

	// Command line parsing can now happen
	ctx := kong.Parse(&CLI,
		kong.Description(version.Description),
		kong.Configuration(kongutil.Hybrid, configDirs...))

	// Initialize logging as soon as possible
	logConfig := zap.NewProductionConfig()
	if err := logConfig.Level.UnmarshalText([]byte(CLI.Logging.Level)); err != nil {
		deferredLogs = append(deferredLogs, err.Error())
	}
	logConfig.Encoding = CLI.Logging.Format

	logger, err := logConfig.Build()
	if err != nil {
		// Error unhandled since this is a very early failure
		_, _ = io.WriteString(stdErr, "Failure while building logger")
		return 1
	}

	// Install as the global logger
	zap.ReplaceGlobals(logger)

	// Emit deferred logs
	logger.Info("Using config paths", zap.Strings("configDirs", configDirs))
	for _, line := range deferredLogs {
		logger.Error(line)
	}

	logger.Info("Configuring asset handling", zap.Bool("use-filesystem", CLI.Assets.UseFilesystem))
	assets.UseFilesystem(CLI.Assets.UseFilesystem)

	if err := dispatchCommands(ctx, appCtx, stdOut); err != nil {
		logger.Error("Error from command", zap.Error(err))
	}

	logger.Info("Exiting normally")
	return 0
}
