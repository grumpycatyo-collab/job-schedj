package log

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const callersToSkip = 3

type logger struct {
	zl *zerolog.Logger
}

func NewLogger() Logger {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	zl := zerolog.New(os.Stdout).
		Level(zerolog.TraceLevel).
		With().
		Timestamp().
		CallerWithSkipFrameCount(callersToSkip).
		Logger()

	return &logger{
		zl: &zl,
	}
}

func (*logger) handleArgs(event *zerolog.Event, args ...any) {
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, okKey := args[i].(string)
			if okKey {
				event.Interface(key, args[i+1])
			} else {
				event = event.Interface("invalid_key_type", args[i])
			}
		} else {
			event = event.Interface("dangling_arg", args[i])
		}
	}
}

func (l *logger) Info(msg string, args ...any) {
	ev := l.zl.Info()
	l.handleArgs(ev, args...)
	ev.Msg(msg)
}

func (l *logger) Error(msg string, err error, args ...any) {
	ev := l.zl.Error().Stack().Err(err)
	l.handleArgs(ev, args...)
	ev.Msg(msg)
}

func (l *logger) Warning(msg string, args ...any) {
	ev := l.zl.Warn()
	l.handleArgs(ev, args...)
	ev.Msg(msg)
}

func (l *logger) Debug(msg string, args ...any) {
	ev := l.zl.Debug()
	l.handleArgs(ev, args...)
	ev.Msg(msg)
}
