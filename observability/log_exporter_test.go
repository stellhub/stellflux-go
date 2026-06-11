package observability

import (
	"bytes"
	"context"
	"strings"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestLocalLogExporterSplitsStdoutAndStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exporter := newLocalLogExporter(&stdout, &stderr, nil, logFormatJSON)

	info := sdklog.Record{}
	info.SetEventName("http.server.request")
	info.SetBody(otellog.StringValue("ok"))
	info.SetSeverity(otellog.SeverityInfo)
	info.SetSeverityText("INFO")

	failed := sdklog.Record{}
	failed.SetEventName("http.server.request")
	failed.SetBody(otellog.StringValue("failed"))
	failed.SetSeverity(otellog.SeverityError)
	failed.SetSeverityText("ERROR")

	if err := exporter.Export(context.Background(), []sdklog.Record{info, failed}); err != nil {
		t.Fatalf("export logs: %v", err)
	}

	if !strings.Contains(stdout.String(), `"body":"ok"`) {
		t.Fatalf("expected info log in stdout, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), `"body":"failed"`) {
		t.Fatalf("did not expect error log in stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `"body":"failed"`) {
		t.Fatalf("expected error log in stderr, got %q", stderr.String())
	}
}
