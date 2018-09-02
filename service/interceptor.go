package service

import (
	"context"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func CheckPeerWithStream(s *Service) interceptor.StreamServerMiddleware {
	return func(next interceptor.StreamHandler) interceptor.StreamHandler {
		return func(srv interface{}, ss common.ServerStream,
			info *common.StreamServerInfo) error {

			ctx := ss.Context()
			p, ok := peer.FromContext(ctx)
			if ok {
				switch info.FullMethod {
				case "/diskqueue.Diskqueue/KeepAlive":
					addr := p.Addr.String()
					c, ok := s.clients.Load(addr)
					if ok {
						cc := c.(*client)
						if cc.state == stateSubscribed {
							return next(srv, ss, info)
						} else {
							return status.Error(codes.FailedPrecondition,
								"client state is not stateSubscribed")
						}
					} else {
						return status.Error(codes.FailedPrecondition,
							"client has not subscribed")
					}
				default:
					return next(srv, ss, info)
				}
			} else {
				return status.Error(codes.FailedPrecondition,
					"The client address could not be obtained")
			}
		}
	}
}

func CheckPeerWithUnary(s *Service) interceptor.UnaryServerMiddleware {
	return func(next interceptor.UnaryHandler) interceptor.UnaryHandler {
		return func(ctx context.Context, req interface{},
			info *common.UnaryServerInfo) (interface{}, error) {

			p, ok := peer.FromContext(ctx)
			if ok {
				switch info.FullMethod {
				case "/diskqueue.Diskqueue/Pull":
					fallthrough
				case "/diskqueue.Diskqueue/Ack":
					fallthrough
				case "/diskqueue.Diskqueue/Cancel":
					addr := p.Addr.String()
					c, ok := s.clients.Load(addr)
					if ok {
						cc := c.(*client)
						if cc.state == stateSubscribed {
							ctx = context.WithValue(ctx, "client", c)
							return next(ctx, req, info)
						} else {
							return nil, status.New(codes.FailedPrecondition,
								"client state is not stateSubscribed").Err()
						}
					} else {
						return nil, status.New(codes.FailedPrecondition,
							"client has not subscribed").Err()
					}
				default:
					return next(ctx, req, info)
				}
			} else {
				return nil, status.New(codes.FailedPrecondition,
					"The client address could not be obtained").Err()
			}
		}
	}
}
