package server

import (
	"net"

	"github.com/ruhancs/virtual-assistant/internal/infra/grpc/pb"
	"github.com/ruhancs/virtual-assistant/internal/infra/grpc/service"
	chatcompletionstream "github.com/ruhancs/virtual-assistant/internal/usecase/chat_completion_stream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"
)

type GRPCServer struct {
	ChatCompletionStreamUseCase chatcompletionstream.ChatCompletionUseCase
	ChatConfig                  chatcompletionstream.ChatCompletionConfigInputDTO
	ChatService                 service.ChatService
	Port                        string
	AuthToken                   string
	StreamChannel               chan chatcompletionstream.ChatCompletionOutputDTO
}


func NewGRPCServer(usecase chatcompletionstream.ChatCompletionUseCase, config chatcompletionstream.ChatCompletionConfigInputDTO, port string,authToken string, channel chan chatcompletionstream.ChatCompletionOutputDTO) *GRPCServer {
	chatService := service.NewChatService(usecase,config,channel)
	return &GRPCServer{
		ChatCompletionStreamUseCase: usecase,
		ChatConfig: config,
		ChatService: *chatService,
		Port: port,
		AuthToken: authToken,
		StreamChannel: channel,
	}
}

func (g *GRPCServer)AuthMiddleware(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	token := md.Get("authorization")
	if len(token) == 0 {
		return status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	if token[0] != g.AuthToken {
		return status.Error(codes.Unauthenticated, "authorization token is invalid")
	}

	return handler(srv, ss)
}

func (gs *GRPCServer) Start() {
	//authenticacao
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(gs.AuthMiddleware),
	}

	//opts... esta a middleware de autheticacao
	grpcServer := grpc.NewServer(opts...)
	//registrar o servidor no grpc
	pb.RegisterChatServiceServer(grpcServer, &gs.ChatService)

	listen,err := net.Listen("tcp",":"+gs.Port)
	if err != nil {
		panic(err)
	}

	if err := grpcServer.Serve(listen); err != nil {
		panic(err)
	}

}
