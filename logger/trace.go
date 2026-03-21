package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// traceIDKey is the context key for TraceID
type traceIDKey struct{}

// TraceIDExtractor defines how to extract TraceID from context
type TraceIDExtractor func(ctx context.Context) string

var (
	// defaultExtractor reads from the specific traceIDKey
	extractor TraceIDExtractor = func(ctx context.Context) string {
		if ctx == nil {
			return ""
		}
		if id, ok := ctx.Value(traceIDKey{}).(string); ok {
			return id
		}
		return ""
	}
	extractorMu sync.RWMutex
)

// SetTraceIDExtractor allows users to customize how to extract TraceID from context.
// For example, if you are using OpenTelemetry, you can set an extractor that reads from span context.
func SetTraceIDExtractor(f TraceIDExtractor) {
	extractorMu.Lock()
	defer extractorMu.Unlock()
	extractor = f
}

// GetTraceID extracts TraceID from context using the registered extractor.
func GetTraceID(ctx context.Context) string {
	extractorMu.RLock()
	defer extractorMu.RUnlock()
	return extractor(ctx)
}

// WithTraceID injects a TraceID into the context using the default key.
// This is useful for the default extractor.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// NewTraceID generates a random 32-character hex string (resembling a UUID without dashes).
func NewTraceID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback if random read fails (unlikely)
		return fmt.Sprintf("%d", 0)
	}
	return hex.EncodeToString(b)
}
