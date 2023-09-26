package service

import (
	"github.com/ruhancs/virtual-assistant/internal/infra/grpc/pb"
	chatcompletionstream "github.com/ruhancs/virtual-assistant/internal/usecase/chat_completion_stream"
)

type ChatService struct {
	pb.UnimplementedChatServiceServer
	ChatCompletionStreamUseCase chatcompletionstream.ChatCompletionUseCase
	ChatConfig                  chatcompletionstream.ChatCompletionConfigInputDTO
	StreamChannel               chan chatcompletionstream.ChatCompletionOutputDTO
}

func NewChatService(usecase chatcompletionstream.ChatCompletionUseCase, config chatcompletionstream.ChatCompletionConfigInputDTO, channel chan chatcompletionstream.ChatCompletionOutputDTO) *ChatService {
	return &ChatService{
		ChatCompletionStreamUseCase: usecase,
		ChatConfig:                  config,
		StreamChannel:               channel,
	}
}

func (c *ChatService) ChatStream(req *pb.ChatRequest, stream pb.ChatService_ChatStreamServer) error {
	chatConfig := chatcompletionstream.ChatCompletionConfigInputDTO{
		Model:                c.ChatConfig.Model,
		ModelMaxTokens:       c.ChatConfig.ModelMaxTokens,
		Temperature:          c.ChatConfig.Temperature,
		TopP:                 c.ChatConfig.TopP,
		N:                    c.ChatConfig.N,
		Stop:                 c.ChatConfig.Stop,
		MaxTokens:            c.ChatConfig.MaxTokens,
		InitialSystemMessage: c.ChatConfig.InitialSystemMessage,
	}

	input := chatcompletionstream.ChatCompletionInputDTO{
		UserMessage: req.GetUserMessage(),
		UserID: req.GetUserId(),
		ChatID: req.GetChatId(),
		Config: chatConfig,
	}

	ctx := stream.Context()

	//le o tudo que é recebido no canal, os dados sao enviados pelo usecase (ChatCompletionStreamUseCase.execute(ctx,input))
	go func ()  {
		for msg := range c.StreamChannel {
			stream.Send(&pb.ChatResponse{
				ChatId: msg.ChatID,
				UserId: msg.UserID,
				Content: msg.Content,
			})
		}
	}()

	//envia as respostas do chat gpt para o canal
	_,err := c.ChatCompletionStreamUseCase.Execute(ctx,input)
	if err != nil {
		return err
	}

	return nil
}
