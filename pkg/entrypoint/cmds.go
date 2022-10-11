package entrypoint

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/wrouesnel/badgeserv/assets"
	"github.com/wrouesnel/badgeserv/pkg/server"
	"go.uber.org/zap"
	"io"
	"io/fs"
)

var (
	ErrCommandNotImplemented = errors.New("Command not implemented")
)

func dispatchCommands(ctx *kong.Context, appCtx context.Context, stdOut io.Writer) error {
	var err error
	logger := zap.L().With(zap.String("command", ctx.Command()))

	switch ctx.Command() {
	case "api":
		err = server.Api(CLI.Api, CLI.Badges)

	case "debug assets list":
		err = fs.WalkDir(assets.Assets, ".", func(path string, d fs.DirEntry, err error) error {
			_, _ = fmt.Fprintf(stdOut, "%s\n", path)
			return nil
		})

	case "debug assets cat":
		var content []byte
		if content, err = assets.Assets.ReadFile(CLI.Debug.Assets.Cat.Filename); err == nil {
			_, _ = stdOut.Write(content)
		} else {
			logger.Error("Error reading embedded file", zap.Error(err))
		}

	default:
		err = ErrCommandNotImplemented
		logger.Error("Command not implemented")
	}

	if err != nil {
		return errors.Wrap(err, ctx.Command())
	}
	return nil
}
