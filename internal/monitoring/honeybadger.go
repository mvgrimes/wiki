package monitoring

import (
	"context"

	"github.com/honeybadger-io/honeybadger-go"
)

func ConfigureHoneybadger(apiKey, environment string) {
	honeybadger.Configure(honeybadger.Configuration{
		APIKey: apiKey,
		Env:    environment,
	})
}

func NotifyError(_ context.Context, err error, fields map[string]any) {
	if err == nil {
		return
	}
	honeybadger.Notify(err, fields)
}
