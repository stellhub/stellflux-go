package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otellog "go.opentelemetry.io/otel/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

func (p *Provider) GRPCServerStatsHandler() stats.Handler {
	if p == nil {
		p = New()
	}
	return otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(p.TracerProvider()),
		otelgrpc.WithMeterProvider(p.MeterProvider()),
		otelgrpc.WithPropagators(p.Propagator()),
	)
}

func (p *Provider) GRPCClientStatsHandler() stats.Handler {
	if p == nil {
		p = New()
	}
	return otelgrpc.NewClientHandler(
		otelgrpc.WithTracerProvider(p.TracerProvider()),
		otelgrpc.WithMeterProvider(p.MeterProvider()),
		otelgrpc.WithPropagators(p.Propagator()),
	)
}

func (p *Provider) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		p.emitGRPCLog(ctx, "grpc.server.request", "unary", info.FullMethod, durationSeconds(start), err)
		return resp, err
	}
}

func (p *Provider) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)
		p.emitGRPCLog(stream.Context(), "grpc.server.request", "stream", info.FullMethod, durationSeconds(start), err)
		return err
	}
}

func (p *Provider) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, conn, opts...)
		p.emitGRPCLog(ctx, "grpc.client.request", "unary", method, durationSeconds(start), err)
		return err
	}
}

func (p *Provider) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()
		stream, err := streamer(ctx, desc, conn, method, opts...)
		p.emitGRPCLog(ctx, "grpc.client.request", "stream", method, durationSeconds(start), err)
		return stream, err
	}
}

func (p *Provider) GRPCServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(p.GRPCServerStatsHandler()),
		grpc.ChainUnaryInterceptor(p.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(p.StreamServerInterceptor()),
	}
}

func (p *Provider) GRPCClientOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(p.GRPCClientStatsHandler()),
		grpc.WithChainUnaryInterceptor(p.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(p.StreamClientInterceptor()),
	}
}

func (p *Provider) emitGRPCLog(ctx context.Context, eventName string, rpcType string, method string, duration float64, err error) {
	code := status.Code(err)
	statusValue := int(code)
	record := newRecord(eventName, "gRPC request completed", statusValue, duration, err)
	if err == nil || code == codes.OK {
		record.SetSeverity(otellog.SeverityInfo)
		record.SetSeverityText("INFO")
	}
	record.AddAttributes(
		otellog.String("rpc.system", "grpc"),
		otellog.String("rpc.method", method),
		otellog.String("rpc.type", rpcType),
		otellog.String("rpc.grpc.status_code", code.String()),
	)
	addTraceAttributes(ctx, &record)
	p.Logger().Emit(ctx, record)
}
