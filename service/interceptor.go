package service

import (
	"context"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func CheckPeerWithUnary(s *Service) interceptor.UnaryServerMiddleware {
	return func(next interceptor.UnaryHandler) interceptor.UnaryHandler {
		return func(ctx context.Context, req interface{},
			info *common.UnaryServerInfo) (interface{}, error) {

			p, ok := peer.FromContext(ctx)
			if ok && p != nil && p.Addr != nil && p.Addr.String() != "" {
				addr := p.Addr.String()
				switch info.FullMethod {
				case "/diskqueue.Diskqueue/Pull":
					log.Info(addr, "pull")
					c := s.getOrCreateClient(addr)
					ctx = context.WithValue(ctx, "client", c)
				case "/diskqueue.Diskqueue/Ack":
					c, ok := s.getClient(addr)
					if ok {
						ctx = context.WithValue(ctx, "client", c)
					} else {
						return nil, status.Error(codes.FailedPrecondition,
							"not found client in diskqueue")
					}
				}
				return next(ctx, req, info)
			} else {
				return nil, status.New(codes.FailedPrecondition,
					"The client address could not be obtained").Err()
			}
		}
	}
}
