package chatcompletionstream

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/ruhancs/virtual-assistant/internal/domain/entity"
	"github.com/ruhancs/virtual-assistant/internal/domain/gateway"
	openai "github.com/sashabaranov/go-openai" //comunicacao com chat gpt
)

// configuracao para enviar ao execute, para configurar a api do chat gpt
type ChatCompletionConfigInputDTO struct {
	Model                string
	ModelMaxTokens       int
	Temperature          float32
	TopP                 float32
	N                    int
	Stop                 []string
	MaxTokens            int
	PresencePenalty      float32
	FrequencyPenalty     float32
	InitialSystemMessage string
}

// dados que o usuario envia para o chat gpt
type ChatCompletionInputDTO struct {
	ChatID      string
	UserID      string
	UserMessage string
	Config      ChatCompletionConfigInputDTO
}

type ChatCompletionOutputDTO struct {
	ChatID  string
	UserID  string
	Content string //resposta do chat gpt
}

type ChatCompletionUseCase struct {
	Gateway      gateway.ChatGateway
	OpenAIClient *openai.Client //comunicacao com api do chat gpt
	Stream       chan ChatCompletionOutputDTO
}

func NewChatCompletionUseCase(gateway gateway.ChatGateway, openAIChatClient *openai.Client, stream chan ChatCompletionOutputDTO) *ChatCompletionUseCase {
	return &ChatCompletionUseCase{
		Gateway:      gateway,
		OpenAIClient: openAIChatClient,
		Stream:       stream,
	}
}

func (usecase *ChatCompletionUseCase) Execute(ctx context.Context, userInput ChatCompletionInputDTO) (*ChatCompletionOutputDTO, error) {
	//checar se o chat existe
	chat, err := usecase.Gateway.FindChatByID(ctx, userInput.ChatID)
	if err != nil {
		if err.Error() == "chat not found" {
			//criar novo chat (entity)
			chat, err = createNewChat(userInput)
			if err != nil {
				return nil, errors.New("error to create the chat: " + err.Error())
			}
			//inserir o novo chat no db
			err = usecase.Gateway.CreateChat(ctx, chat)
			if err != nil {
				return nil, errors.New("error to save the chat on db: " + err.Error())
			}
		} else {
			return nil, errors.New("error fetching existing chat: " + err.Error())
		}
	}

	//criacao da message para enviar ao chat
	userMessage, err := entity.NewMessage("user", userInput.UserMessage, chat.Config.Model)
	if err != nil {
		return nil, errors.New("error creating user msg: " + err.Error())
	}

	err = chat.AddMessage(userMessage)
	if err != nil {
		return nil, errors.New("error to add new user msg: " + err.Error())
	}

	//adicionar todas messages do chat em messages, no formato da api do openai
	messages := []openai.ChatCompletionMessage{}
	for _, msg := range chat.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	//enviar o contexto de messages ao chat para ele retornar a resposta
	respStream, err := usecase.OpenAIClient.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model:            chat.Config.Model.Name,
			Messages:         messages,
			MaxTokens:        chat.Config.MaxTokens,
			Temperature:      chat.Config.Temperature,
			TopP:             chat.Config.TopP,
			PresencePenalty:  chat.Config.PresencePenalty,
			FrequencyPenalty: chat.Config.FrequencyPenalty,
			Stop:             chat.Config.Stop,
			Stream:           true, //conforme vai gerando a msg ja vai enviando, nao espera a msg estar totalmente pronta
		},
	)
	if err != nil {
		return nil, errors.New("error creating chat completion: " + err.Error())
	}

	//observar a msg de resposta do chat gpt conforme ele envia
	var fullResponse strings.Builder//strings.builder() permiter adicionar mais dados a string
	for {
		response, err := respStream.Recv()
		// erro que indica que a msg acabou
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("error streming response: " + err.Error())
		}
		//inserir conforme chega a resposta do chat gpt em fullResponse
		fullResponse.WriteString(response.Choices[0].Delta.Content)

		//montar o output do chat
		r := ChatCompletionOutputDTO{
			ChatID:  chat.ID,
			UserID:  userInput.UserID,
			Content: fullResponse.String(),
		}
		//inserir a saida no canal, para ser enviado por outra thread, que sera utilizado com grpc para saida
		usecase.Stream <- r
	}

	//criar msgs igual ao contexto de msgs enviadas ao chat para ser salva no db
	assistant, err := entity.NewMessage("assistant", fullResponse.String(), chat.Config.Model)
	if err != nil {
		return nil, errors.New("error to create new message: " + err.Error())
	}
	err = chat.AddMessage(assistant)
	if err != nil {
		return nil, errors.New("error to add message: " + err.Error())
	}
	//salvar dados do chat
	err = usecase.Gateway.SaveChat(ctx, chat)
	if err != nil {
		return nil, errors.New("error save chat on db: " + err.Error())
	}

	return &ChatCompletionOutputDTO{
		ChatID:  chat.ID,
		UserID:  userInput.UserID,
		Content: fullResponse.String(),
	}, nil
}

func createNewChat(input ChatCompletionInputDTO) (*entity.Chat, error) {
	model := entity.NewModel(input.Config.Model, input.Config.ModelMaxTokens)
	chatConfig := &entity.ChatConfig{
		Temperature:      input.Config.Temperature,
		TopP:             input.Config.TopP,
		N:                input.Config.N,
		Stop:             input.Config.Stop,
		MaxTokens:        input.Config.ModelMaxTokens,
		PresencePenalty:  input.Config.PresencePenalty,
		FrequencyPenalty: input.Config.FrequencyPenalty,
		Model:            model,
	}
	initialMessage, err := entity.NewMessage("system", input.Config.InitialSystemMessage, model)
	if err != nil {
		return nil, errors.New("error to create initial message: " + err.Error())
	}

	chat, err := entity.NewChat(input.UserID, initialMessage, chatConfig)
	if err != nil {
		return nil, errors.New("error to create new chat: " + err.Error())
	}

	return chat, nil
}
