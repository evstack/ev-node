package submitting

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

var _ DASubmitterAPI = (*tracedDASubmitter)(nil)

// tracedDASubmitter wraps a DASubmitterAPI with OpenTelemetry tracing.
type tracedDASubmitter struct {
	inner  DASubmitterAPI
	tracer trace.Tracer
}

// WithTracingDASubmitter wraps a DASubmitterAPI with OpenTelemetry tracing.
func WithTracingDASubmitter(inner DASubmitterAPI) DASubmitterAPI {
	return &tracedDASubmitter{
		inner:  inner,
		tracer: otel.Tracer("ev-node/da-submitter"),
	}
}

func (t *tracedDASubmitter) SubmitHeaders(ctx context.Context, headers []*types.SignedHeader, marshalledHeaders [][]byte, cache cache.Manager, signer signer.Signer) error {
	ctx, span := t.tracer.Start(ctx, "DASubmitter.SubmitHeaders",
		trace.WithAttributes(
			attribute.Int("header.count", len(headers)),
		),
	)
	defer span.End()

	// calculate total size
	var totalBytes int
	for _, h := range marshalledHeaders {
		totalBytes += len(h)
	}
	span.SetAttributes(attribute.Int("header.total_bytes", totalBytes))

	// add height range if headers present
	if len(headers) > 0 {
		span.SetAttributes(
			attribute.Int64("header.start_height", int64(headers[0].Height())),
			attribute.Int64("header.end_height", int64(headers[len(headers)-1].Height())),
		)
	}

	err := t.inner.SubmitHeaders(ctx, headers, marshalledHeaders, cache, signer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (t *tracedDASubmitter) SubmitData(ctx context.Context, signedDataList []*types.SignedData, marshalledData [][]byte, cache cache.Manager, signer signer.Signer, genesis genesis.Genesis) error {
	ctx, span := t.tracer.Start(ctx, "DASubmitter.SubmitData",
		trace.WithAttributes(
			attribute.Int("data.count", len(signedDataList)),
		),
	)
	defer span.End()

	// calculate total size
	var totalBytes int
	for _, d := range marshalledData {
		totalBytes += len(d)
	}
	span.SetAttributes(attribute.Int("data.total_bytes", totalBytes))

	// add height range if data present
	if len(signedDataList) > 0 {
		span.SetAttributes(
			attribute.Int64("data.start_height", int64(signedDataList[0].Height())),
			attribute.Int64("data.end_height", int64(signedDataList[len(signedDataList)-1].Height())),
		)
	}

	err := t.inner.SubmitData(ctx, signedDataList, marshalledData, cache, signer, genesis)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}
