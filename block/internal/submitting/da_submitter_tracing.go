package submitting

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

var _ DASubmitterAPI = (*tracedDASubmitter)(nil)

type tracedDASubmitter struct {
	inner  DASubmitterAPI
	tracer trace.Tracer
}

func WithTracingDASubmitter(inner DASubmitterAPI) DASubmitterAPI {
	return &tracedDASubmitter{
		inner:  inner,
		tracer: otel.Tracer("ev-node/da-submitter"),
	}
}

func (t *tracedDASubmitter) SubmitBlocks(ctx context.Context, headers []*types.SignedHeader, data []*types.Data, cacheMgr cache.Manager, signer signer.Signer, onSubmitError func(error)) error {
	ctx, span := t.tracer.Start(ctx, "DASubmitter.SubmitBlocks",
		trace.WithAttributes(
			attribute.Int("block.count", len(headers)),
		),
	)

	if len(headers) > 0 {
		span.SetAttributes(
			attribute.Int64("block.start_height", int64(headers[0].Height())),
			attribute.Int64("block.end_height", int64(headers[len(headers)-1].Height())),
		)
	}

	var wrappedOnError func(error)
	if onSubmitError != nil {
		wrappedOnError = func(err error) {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
			onSubmitError(err)
		}
	}

	err := t.inner.SubmitBlocks(ctx, headers, data, cacheMgr, signer, wrappedOnError)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return err
	}

	if onSubmitError == nil {
		span.End()
	}

	return nil
}

func (t *tracedDASubmitter) Close() {
	t.inner.Close()
}
