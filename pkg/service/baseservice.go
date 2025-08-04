package service

import (
	"context"

	"github.com/rs/zerolog"
)

/*
type FooService struct {
	BaseService
	// extra fields for FooService
}
func NewFooService(logger log.Logger) *FooService {
	fs := &FooService{}
	fs.BaseService = *NewBaseService(logger, "FooService", fs)
	return fs
}
func (fs *FooService) OnStart(ctx context.Context) error {
	// initialize fields, start goroutines, etc.
	go func() {
		<-ctx.Done()
		// cleanup when context is cancelled
	}()
	return nil
}
func (fs *FooService) OnStop(ctx context.Context) {
	// stop routines, cleanup resources, etc.
}
func (fs *FooService) OnReset(ctx context.Context) error {
	// implement reset if desired
	return nil
}
*/

// Service exposes a Run method that blocks until the service ends or the context is canceled.
type Service interface {
	// Run starts the service and blocks until it is shut down via context cancellation,
	// an error occurs, or all work is done.
	Run(ctx context.Context) error
}

// BaseService provides a basic implementation of the Service interface.
type BaseService struct {
	Logger zerolog.Logger
	name   string
	impl   Service // Implementation that can override Run behavior
}

// NewBaseService creates a new BaseService.
// The provided implementation (impl) should be the "subclass" that implements Run.
func NewBaseService(logger zerolog.Logger, name string, impl Service) *BaseService {
	return &BaseService{
		Logger: logger,
		name:   name,
		impl:   impl,
	}
}

// SetLogger sets the logger.
func (bs *BaseService) SetLogger(l zerolog.Logger) {
	bs.Logger = l
}

// Run implements the Service interface. It logs the start of the service,
// then defers to the implementation's Run method to do the actual work.
// If impl is nil or the same as bs, it uses the default implementation.
func (bs *BaseService) Run(ctx context.Context) error {
	bs.Logger.Info().Str("Starting service", bs.name).Str("impl", bs.name).Msg("service start")

	// If the implementation is nil or is the BaseService itself,
	// use the default implementation which just waits for context cancellation
	if bs.impl == nil || bs.impl == bs {
		<-ctx.Done()
		bs.Logger.Info().Str("Stopping service", bs.name).Str("impl", bs.name).Msg("service stop")
		return ctx.Err()
	}

	// Otherwise, call the implementation's Run method
	return bs.impl.Run(ctx)
}

// String returns the service name.
func (bs *BaseService) String() string {
	return bs.name
}
