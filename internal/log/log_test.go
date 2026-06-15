package log_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/log"
)

func TestInitAndDebug(t *testing.T) {
	var buf bytes.Buffer

	err := log.Init(log.Options{
		Level:   log.DebugLevel,
		Output:  &buf,
		NoColor: true,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Cleanup(func() { _ = log.Close() })

	log.Debug("hello", "key", "value")

	got := buf.String()
	if !strings.Contains(got, "hello") {
		t.Errorf("output missing message: %q", got)
	}

	if !strings.Contains(got, "key") {
		t.Errorf("output missing key: %q", got)
	}

	if !strings.Contains(got, "value") {
		t.Errorf("output missing value: %q", got)
	}
}

func TestDebugfFormatting(t *testing.T) {
	var buf bytes.Buffer

	err := log.Init(log.Options{
		Level:   log.DebugLevel,
		Output:  &buf,
		NoColor: true,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Cleanup(func() { _ = log.Close() })

	log.Debugf("number=%d", 42)

	if !strings.Contains(buf.String(), "number=42") {
		t.Errorf("Debugf did not format: %q", buf.String())
	}
}

func TestLevelGating(t *testing.T) {
	var buf bytes.Buffer

	err := log.Init(log.Options{
		Level:   log.WarnLevel,
		Output:  &buf,
		NoColor: true,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Cleanup(func() { _ = log.Close() })

	log.Debug("debug-msg")
	log.Info("info-msg")
	log.Warn("warn-msg")
	log.Error("error-msg")

	got := buf.String()
	if strings.Contains(got, "debug-msg") {
		t.Errorf("debug should be filtered at WarnLevel: %q", got)
	}

	if strings.Contains(got, "info-msg") {
		t.Errorf("info should be filtered at WarnLevel: %q", got)
	}

	if !strings.Contains(got, "warn-msg") {
		t.Errorf("warn should pass: %q", got)
	}

	if !strings.Contains(got, "error-msg") {
		t.Errorf("error should pass: %q", got)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    log.Level
		wantErr bool
	}{
		{"debug", "debug", log.DebugLevel, false},
		{"DEBUG uppercase", "DEBUG", log.DebugLevel, false},
		{"trace alias", "trace", log.DebugLevel, false},
		{"info", "info", log.InfoLevel, false},
		{"warn", "warn", log.WarnLevel, false},
		{"warning alias", "warning", log.WarnLevel, false},
		{"empty defaults to warn", "", log.WarnLevel, false},
		{"error", "error", log.ErrorLevel, false},
		{"err alias", "err", log.ErrorLevel, false},
		{"fatal", "fatal", log.FatalLevel, false},
		{"unknown", "potato", log.WarnLevel, true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := log.ParseLevel(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseLevel(%q) err=%v, wantErr=%v", testCase.input, err, testCase.wantErr)
			}

			if got != testCase.want {
				t.Errorf("ParseLevel(%q)=%v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestFilePathTees(t *testing.T) {
	dir := t.TempDir()

	var stderr bytes.Buffer

	filePath := dir + "/nvs.log"

	err := log.Init(log.Options{
		Level:    log.DebugLevel,
		Output:   &stderr,
		FilePath: filePath,
		NoColor:  true,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Cleanup(func() { _ = log.Close() })

	log.Debug("tee-test")

	if !strings.Contains(stderr.String(), "tee-test") {
		t.Errorf("stderr missing message: %q", stderr.String())
	}
}

func TestSetLevelRuntime(t *testing.T) {
	var buf bytes.Buffer

	err := log.Init(log.Options{
		Level:   log.WarnLevel,
		Output:  &buf,
		NoColor: true,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	t.Cleanup(func() { _ = log.Close() })

	log.Debug("first")

	if strings.Contains(buf.String(), "first") {
		t.Errorf("debug should be filtered before SetLevel: %q", buf.String())
	}

	log.SetLevel(log.DebugLevel)
	log.Debug("second")

	if !strings.Contains(buf.String(), "second") {
		t.Errorf("debug should pass after SetLevel: %q", buf.String())
	}

	if log.GetLevel() != log.DebugLevel {
		t.Errorf("GetLevel=%v want Debug", log.GetLevel())
	}
}
