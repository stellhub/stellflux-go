package internal

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const customRouterServiceName = "stellar.examples.grpc.CustomRouter"

type customRouterService struct{}

func customRouterServiceDesc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: customRouterServiceName,
		HandlerType: (*customRouterServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler:    pingHandler,
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "examples/grpc/server/custom-router",
	}
}

type customRouterServer interface {
	Ping(context.Context, *emptypb.Empty) (*structpb.Struct, error)
}

func (s *customRouterService) Ping(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{
		"message": "pong",
		"service": "grpc-custom-router",
	})
}

func pingHandler(
	srv any,
	ctx context.Context,
	dec func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	req := new(emptypb.Empty)
	if err := dec(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(customRouterServer).Ping(ctx, req)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/" + customRouterServiceName + "/Ping",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(customRouterServer).Ping(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, req, info, handler)
}
