package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func initLogger() {
	Log = logrus.New()
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
		FullTimestamp:             true,
	})
	Log.SetLevel(logrus.DebugLevel)
}

type logrusSLogProxy struct{}

func (p *logrusSLogProxy) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

func (proxy *logrusSLogProxy) Handle(ctx context.Context, record slog.Record) error {
	record.Message = "[ACMEzClient] " + record.Message + " "
	attrsMap := make(map[string]any)
	record.Attrs(func(a slog.Attr) bool {
		attrsMap[a.Key] = a.Value.Any()
		return true
	})
	if len(attrsMap) > 0 {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(attrsMap); err != nil {
			record.Message += err.Error()
		} else {
			record.Message += buf.String()
		}
	}
	Log.Debugln(strings.TrimSpace(record.Message))
	return nil
}

func (p *logrusSLogProxy) WithAttrs(attrs []slog.Attr) slog.Handler {
	return p
}

func (p *logrusSLogProxy) WithGroup(name string) slog.Handler {
	return p
}

func getLogrusSLogProxy() *slog.Logger {
	return slog.New(&logrusSLogProxy{})
}
