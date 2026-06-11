package observability

import (
	"testing"

	"github.com/stellhub/stellar/config"
)

func TestNormalizeLogOutput(t *testing.T) {
	tests := map[string]struct {
		cfg  config.OpenTelemetryLogConfig
		want string
	}{
		"default local console": {
			cfg:  config.OpenTelemetryLogConfig{},
			want: outputConsole,
		},
		"local file": {
			cfg:  config.OpenTelemetryLogConfig{Output: outputFile},
			want: outputFile,
		},
		"enabled otlp": {
			cfg:  config.OpenTelemetryLogConfig{Enabled: true},
			want: outputOTLP,
		},
		"disabled ignores otlp output": {
			cfg:  config.OpenTelemetryLogConfig{Output: outputOTLP},
			want: outputConsole,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := normalizeLogOutput(tt.cfg); got != tt.want {
				t.Fatalf("normalizeLogOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}
