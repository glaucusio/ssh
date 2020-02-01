package sshtrace

import (
	"context"

	"github.com/glaucusio/ssh"
	"github.com/glaucusio/ssh/sshfile"
)

type contextKey struct{ string }

var clientTraceKey = contextKey{"client-trace"}

type ClientTrace struct {
	GotFileConfig func(sshfile.Configs)
	GotConfig     func(*ssh.Config)
}

func ContextClientTrace(ctx context.Context) *ClientTrace {
	if ct, ok := ctx.Value(clientTraceKey).(*ClientTrace); ok {
		return ct
	}
	return nil
}

func WithClientTrace(ctx context.Context, ct *ClientTrace) context.Context {
	return context.WithValue(ctx, clientTraceKey, ct)
}
