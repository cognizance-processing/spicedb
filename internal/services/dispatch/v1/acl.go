package dispatch

import (
	"context"
	"errors"
	"time"

	"spicedb/internal/middleware/streamtimeout"

	"spicedb/internal/middleware"

	grpcvalidate "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"spicedb/internal/dispatch"
	"spicedb/internal/graph"
	log "spicedb/internal/logging"
	"spicedb/internal/services/shared"
	dispatchv1 "spicedb/pkg/proto/dispatch/v1"
)

const streamAPITimeout = 45 * time.Second

type dispatchServer struct {
	dispatchv1.UnimplementedDispatchServiceServer
	shared.WithServiceSpecificInterceptors

	localDispatch dispatch.Dispatcher
}

// NewDispatchServer creates a server which can be called for internal dispatch.
func NewDispatchServer(localDispatch dispatch.Dispatcher) dispatchv1.DispatchServiceServer {
	return &dispatchServer{
		localDispatch: localDispatch,
		WithServiceSpecificInterceptors: shared.WithServiceSpecificInterceptors{
			Unary: grpcvalidate.UnaryServerInterceptor(),
			Stream: middleware.ChainStreamServer(
				grpcvalidate.StreamServerInterceptor(),
				streamtimeout.MustStreamServerInterceptor(streamAPITimeout),
			),
		},
	}
}

func (ds *dispatchServer) DispatchCheck(ctx context.Context, req *dispatchv1.DispatchCheckRequest) (*dispatchv1.DispatchCheckResponse, error) {
	resp, err := ds.localDispatch.DispatchCheck(ctx, req)
	return resp, rewriteGraphError(ctx, err)
}

func (ds *dispatchServer) DispatchExpand(ctx context.Context, req *dispatchv1.DispatchExpandRequest) (*dispatchv1.DispatchExpandResponse, error) {
	resp, err := ds.localDispatch.DispatchExpand(ctx, req)
	return resp, rewriteGraphError(ctx, err)
}

func (ds *dispatchServer) DispatchReachableResources(
	req *dispatchv1.DispatchReachableResourcesRequest,
	resp dispatchv1.DispatchService_DispatchReachableResourcesServer,
) error {
	return ds.localDispatch.DispatchReachableResources(req,
		dispatch.WrapGRPCStream[*dispatchv1.DispatchReachableResourcesResponse](resp))
}

func (ds *dispatchServer) DispatchLookupResources(
	req *dispatchv1.DispatchLookupResourcesRequest,
	resp dispatchv1.DispatchService_DispatchLookupResourcesServer,
) error {
	return ds.localDispatch.DispatchLookupResources(req,
		dispatch.WrapGRPCStream[*dispatchv1.DispatchLookupResourcesResponse](resp))
}

func (ds *dispatchServer) DispatchLookupSubjects(
	req *dispatchv1.DispatchLookupSubjectsRequest,
	resp dispatchv1.DispatchService_DispatchLookupSubjectsServer,
) error {
	return ds.localDispatch.DispatchLookupSubjects(req,
		dispatch.WrapGRPCStream[*dispatchv1.DispatchLookupSubjectsResponse](resp))
}

func (ds *dispatchServer) Close() error {
	return nil
}

func rewriteGraphError(ctx context.Context, err error) error {
	// Check if the error can be directly used.
	if st, ok := status.FromError(err); ok {
		return st.Err()
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return status.Errorf(codes.DeadlineExceeded, "%s", err)
	case errors.Is(err, context.Canceled):
		return status.Errorf(codes.Canceled, "%s", err)
	case status.Code(err) == codes.Canceled:
		return err
	case err == nil:
		return nil

	case errors.As(err, &graph.ErrAlwaysFail{}):
		fallthrough
	default:
		log.Ctx(ctx).Err(err).Msg("unexpected dispatch graph error")
		return err
	}
}
