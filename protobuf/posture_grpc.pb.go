// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: posture.proto

package protobuf

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// PostureServiceClient is the client API for PostureService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PostureServiceClient interface {
	UpdateDefaultPosture(ctx context.Context, in *KubeArmorConfig, opts ...grpc.CallOption) (*Status, error)
}

type postureServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPostureServiceClient(cc grpc.ClientConnInterface) PostureServiceClient {
	return &postureServiceClient{cc}
}

func (c *postureServiceClient) UpdateDefaultPosture(ctx context.Context, in *KubeArmorConfig, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, "/posture.PostureService/updateDefaultPosture", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PostureServiceServer is the server API for PostureService service.
// All implementations should embed UnimplementedPostureServiceServer
// for forward compatibility
type PostureServiceServer interface {
	UpdateDefaultPosture(context.Context, *KubeArmorConfig) (*Status, error)
}

// UnimplementedPostureServiceServer should be embedded to have forward compatible implementations.
type UnimplementedPostureServiceServer struct {
}

func (UnimplementedPostureServiceServer) UpdateDefaultPosture(context.Context, *KubeArmorConfig) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDefaultPosture not implemented")
}

// UnsafePostureServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PostureServiceServer will
// result in compilation errors.
type UnsafePostureServiceServer interface {
	mustEmbedUnimplementedPostureServiceServer()
}

func RegisterPostureServiceServer(s grpc.ServiceRegistrar, srv PostureServiceServer) {
	s.RegisterService(&PostureService_ServiceDesc, srv)
}

func _PostureService_UpdateDefaultPosture_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KubeArmorConfig)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostureServiceServer).UpdateDefaultPosture(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/posture.PostureService/updateDefaultPosture",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostureServiceServer).UpdateDefaultPosture(ctx, req.(*KubeArmorConfig))
	}
	return interceptor(ctx, in, info, handler)
}

// PostureService_ServiceDesc is the grpc.ServiceDesc for PostureService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PostureService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "posture.PostureService",
	HandlerType: (*PostureServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "updateDefaultPosture",
			Handler:    _PostureService_UpdateDefaultPosture_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "posture.proto",
}
