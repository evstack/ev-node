package da

import (
	"context"
	"encoding/hex"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// tracedClient decorates a FullClient with OpenTelemetry spans.
type tracedClient struct {
	inner  FullClient
	tracer trace.Tracer
}

// WithTracingClient decorates the provided client with tracing spans.
func WithTracingClient(inner FullClient) FullClient {
	return &tracedClient{inner: inner, tracer: otel.Tracer("ev-node/da")}
}

func (t *tracedClient) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	total := 0
	for _, b := range data {
		total += len(b)
	}
	ctx, span := t.tracer.Start(ctx, "DA.Submit",
		trace.WithAttributes(
			attribute.Int("blob.count", len(data)),
			attribute.Int("blob.total_size_bytes", total),
			attribute.String("da.namespace", hex.EncodeToString(namespace)),
		),
	)
	defer span.End()

	res := t.inner.Submit(ctx, data, gasPrice, namespace, options)
	if res.Code != datypes.StatusSuccess {
		span.RecordError(&submitError{msg: res.Message})
		span.SetStatus(codes.Error, res.Message)
	} else {
		span.SetAttributes(attribute.Int64("da.height", int64(res.Height)))
	}
	return res
}

func (t *tracedClient) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	ctx, span := t.tracer.Start(ctx, "DA.Retrieve",
		trace.WithAttributes(
			attribute.Int("ns.length", len(namespace)),
			attribute.String("da.namespace", hex.EncodeToString(namespace)),
		),
	)
	defer span.End()

	res := t.inner.Retrieve(ctx, height, namespace)

	if res.Code != datypes.StatusSuccess && res.Code != datypes.StatusNotFound {
		span.RecordError(&submitError{msg: res.Message})
		span.SetStatus(codes.Error, res.Message)
	} else {
		span.SetAttributes(attribute.Int("blob.count", len(res.Data)))
	}
	return res
}

func (t *tracedClient) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	ctx, span := t.tracer.Start(ctx, "DA.Get",
		trace.WithAttributes(
			attribute.Int("id.count", len(ids)),
			attribute.String("da.namespace", hex.EncodeToString(namespace)),
		),
	)
	defer span.End()

	blobs, err := t.inner.Get(ctx, ids, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("blob.count", len(blobs)))
	return blobs, nil
}

func (t *tracedClient) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	ctx, span := t.tracer.Start(ctx, "DA.GetProofs",
		trace.WithAttributes(
			attribute.Int("id.count", len(ids)),
			attribute.String("da.namespace", hex.EncodeToString(namespace)),
		),
	)
	defer span.End()

	proofs, err := t.inner.GetProofs(ctx, ids, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("proof.count", len(proofs)))
	return proofs, nil
}

func (t *tracedClient) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	ctx, span := t.tracer.Start(ctx, "DA.Validate",
		trace.WithAttributes(
			attribute.Int("id.count", len(ids)),
			attribute.String("da.namespace", hex.EncodeToString(namespace)),
		),
	)
	defer span.End()
	res, err := t.inner.Validate(ctx, ids, proofs, namespace)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("result.count", len(res)))
	return res, nil
}

func (t *tracedClient) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	ctx, span := t.tracer.Start(ctx, "DA.GetLatestDAHeight")
	defer span.End()

	height, err := t.inner.GetLatestDAHeight(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	span.SetAttributes(attribute.Int64("da.latest_height", int64(height)))
	return height, nil
}

func (t *tracedClient) GetHeaderNamespace() []byte { return t.inner.GetHeaderNamespace() }
func (t *tracedClient) GetDataNamespace() []byte   { return t.inner.GetDataNamespace() }
func (t *tracedClient) GetForcedInclusionNamespace() []byte {
	return t.inner.GetForcedInclusionNamespace()
}
func (t *tracedClient) HasForcedInclusionNamespace() bool {
	return t.inner.HasForcedInclusionNamespace()
}

type submitError struct{ msg string }

func (e *submitError) Error() string { return e.msg }
