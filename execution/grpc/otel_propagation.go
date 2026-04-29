package grpc

import (
	"context"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func inboundPropagationInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			prop := otel.GetTextMapPropagator()
			ctx = prop.Extract(ctx, propagation.HeaderCarrier(req.Header()))
			return next(ctx, req)
		}
	})
}

func outboundPropagationInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			prop := otel.GetTextMapPropagator()
			prop.Inject(ctx, propagation.HeaderCarrier(req.Header()))
			return next(ctx, req)
		}
	})
}
