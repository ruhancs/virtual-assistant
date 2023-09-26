package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ruhancs/virtual-assistant/internal/domain/entity"
	"github.com/ruhancs/virtual-assistant/internal/infra/db"
)

type ChatRepository struct {
	DB      *sql.DB
	Queries *db.Queries
}

func NewChatRepositoryMySql(database *sql.DB) *ChatRepository {
	return &ChatRepository{
		DB:      database,
		Queries: db.New(database),
	}
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat *entity.Chat) error {
	err := r.Queries.CreateChat(
		ctx,
		db.CreateChatParams{
			ID:               chat.ID,
			UserID:           chat.UserID,
			InitialMessageID: chat.InitialSystemMessage.Content,
			Status:           chat.Status,
			TokenUsage:       int32(chat.TokenUsage),
			Model:            chat.Config.Model.Name,
			ModelMaxTokens:   int32(chat.Config.Model.MaxTokens),
			Temperature:      float64(chat.Config.Temperature),
			TopP:             float64(chat.Config.TopP),
			N:                int32(chat.Config.N),
			Stop:             chat.Config.Stop[0],
			MaxTokens:        int32(chat.Config.MaxTokens),
			PresencePenalty:  float64(chat.Config.PresencePenalty),
			FrequencyPenalty: float64(chat.Config.FrequencyPenalty),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	)
	if err != nil {
		return err
	}

	//cria msg inicial para iniciar o chat
	err = r.Queries.AddMessage(
		ctx,
		db.AddMessageParams{
			ID:        chat.InitialSystemMessage.ID,
			ChatID:    chat.ID,
			Content:   chat.InitialSystemMessage.Content,
			Role:      chat.InitialSystemMessage.Role,
			Tokens:    int32(chat.InitialSystemMessage.Tokens),
			CreatedAt: chat.InitialSystemMessage.CreatedAt,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *ChatRepository) FindChatByID(ctx context.Context, chatID string) (*entity.Chat,error) {
	chat := &entity.Chat{}
	
	res,err := r.Queries.FindChatByID(ctx,chatID)
	if err != nil {
		return nil, errors.New("chat not found")
	}

	//passar o chat model para chat entity
	chat.ID = res.ID
	chat.UserID = res.UserID
	chat.Status = res.Status
	chat.TokenUsage = int(res.TokenUsage)
	chat.Config = &entity.ChatConfig{
		Model: &entity.Model{
			Name: res.Model,
			MaxTokens: int(res.ModelMaxTokens),
		},
		Temperature: float32(res.Temperature),
		TopP: float32(res.TopP),
		N: int(res.N),
		Stop: []string{res.Stop},
		MaxTokens: int(res.MaxTokens),
		PresencePenalty: float32(res.PresencePenalty),
		FrequencyPenalty: float32(res.FrequencyPenalty),
	}

	//pegar as messages do chat pelo id
	messages,err := r.Queries.FindMessagesByChatID(ctx, chatID)
	if err != nil {
		return nil,err
	}

	//adicionar as messages do chat model no chat entity, menssagens ativas do chat
	for _,msg := range messages {
		chat.Messages = append(chat.Messages, &entity.Message{
			ID: msg.ID,
			Content: msg.Content,
			Role: msg.Role,
			Tokens: int(msg.Tokens),
			Model: &entity.Model{Name: msg.Model,},
			CreatedAt: msg.CreatedAt,
		})
	}

	//menssagens apagadas do chat
	errasedMessages,err := r.Queries.FindErasedMessagesByChatID(ctx,chatID)
	if err != nil {
		return nil,err
	}

	//adicionar as messages do chat model no chat entity, menssagens ativas do chat
	for _,msg := range errasedMessages {
		chat.ErasedMessages = append(chat.Messages, &entity.Message{
			ID: msg.ID,
			Content: msg.Content,
			Role: msg.Role,
			Tokens: int(msg.Tokens),
			Model: &entity.Model{Name: msg.Model,},
			CreatedAt: msg.CreatedAt,
		})
	}
	return chat, nil
}

func (r *ChatRepository) SaveChat(ctx context.Context, chat *entity.Chat) error {
	params := db.SaveChatParams{
		ID:               chat.ID,
		UserID:           chat.UserID,
		Status:           chat.Status,
		TokenUsage:       int32(chat.TokenUsage),
		Model:            chat.Config.Model.Name,
		ModelMaxTokens:   int32(chat.Config.Model.MaxTokens),
		Temperature:      float64(chat.Config.Temperature),
		TopP:             float64(chat.Config.TopP),
		N:                int32(chat.Config.N),
		Stop:             chat.Config.Stop[0],
		MaxTokens:        int32(chat.Config.MaxTokens),
		PresencePenalty:  float64(chat.Config.PresencePenalty),
		FrequencyPenalty: float64(chat.Config.FrequencyPenalty),
		UpdatedAt:        time.Now(),
	}

	err := r.Queries.SaveChat(
		ctx,
		params,
	)
	if err != nil {
		return err
	}
	// delete messages
	err = r.Queries.DeleteChatMessages(ctx, chat.ID)
	if err != nil {
		return err
	}
	// delete erased messages
	err = r.Queries.DeleteErasedChatMessages(ctx, chat.ID)
	if err != nil {
		return err
	}
	// save messages
	i := 0
	for _, message := range chat.Messages {
		err = r.Queries.AddMessage(
			ctx,
			db.AddMessageParams{
				ID:        message.ID,
				ChatID:    chat.ID,
				Content:   message.Content,
				Role:      message.Role,
				Tokens:    int32(message.Tokens),
				Model:     chat.Config.Model.Name,
				CreatedAt: message.CreatedAt,
				OrderMsg:  int32(i),
				Erased:    false,
			},
		)
		if err != nil {
			return err
		}
		i++
	}
	// save erased messages
	i = 0
	for _, message := range chat.ErasedMessages {
		err = r.Queries.AddMessage(
			ctx,
			db.AddMessageParams{
				ID:        message.ID,
				ChatID:    chat.ID,
				Content:   message.Content,
				Role:      message.Role,
				Tokens:    int32(message.Tokens),
				Model:     chat.Config.Model.Name,
				CreatedAt: message.CreatedAt,
				OrderMsg:  int32(i),
				Erased:    true,
			},
		)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}
