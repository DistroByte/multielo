package domain

import "context"

type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

type NoopLogger struct{}

func (NoopLogger) Info(msg string, fields map[string]interface{})             {}
func (NoopLogger) Error(msg string, err error, fields map[string]interface{}) {}
func (NoopLogger) Debug(msg string, fields map[string]interface{})            {}

type MetricsRecorder interface {
	RecordMatchAdded(playerCount int)
	RecordPlayerAdded()
	RecordELOChange(playerName string, delta int)
	RecordError(operation string, errType ErrorType)
}

type NoopMetrics struct{}

func (NoopMetrics) RecordMatchAdded(playerCount int)                {}
func (NoopMetrics) RecordPlayerAdded()                              {}
func (NoopMetrics) RecordELOChange(playerName string, delta int)    {}
func (NoopMetrics) RecordError(operation string, errType ErrorType) {}

type ArchiveCallback func(ctx context.Context, matches []*Match) error

var NoopArchiveCallback ArchiveCallback = func(ctx context.Context, matches []*Match) error {
	return nil
}
