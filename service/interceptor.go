package service

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryIntercept(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	_, ok := peer.FromContext(ctx)
	if ok {
		return handler(ctx, req)
	} else {
		return nil, status.New(codes.FailedPrecondition,
			"The client address could not be obtained").Err()
	}
}
