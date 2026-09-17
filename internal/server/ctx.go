package server

import "context"

type ctxKey struct{}

func contextWithReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func reqID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
