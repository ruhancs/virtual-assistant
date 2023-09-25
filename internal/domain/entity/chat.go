package entity

import (
	"errors"

	"github.com/google/uuid"
)

type ChatConfig struct {
	Model            *Model
	Temperature      float32  // 0.0 to 1.0 precisao da resposta 0 mais presiso
	TopP             float32  // 0.0 to 1.0 - conservador nas respostas
	N                int      // number of messages to generate
	Stop             []string // list of tokens to stop on
	MaxTokens        int      // maximo de tokens de uma conversa
	PresencePenalty  float32  // -2.0 to 2.0 - Number between -2.0 and 2.0. penalizacao por palavras repetidas
	FrequencyPenalty float32  // -2.0 to 2.0 - Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing frequency in the text so far, increasing the model's likelihood to talk about new topics.
}

type Chat struct {
	ID                   string
	UserID               string
	InitialSystemMessage *Message
	Messages             []*Message
	ErasedMessages       []*Message // mensagens retiradas do contexto de envio ao chat gpt
	Status               string
	TokenUsage           int //qnts token ja foram utilizados
	Config               *ChatConfig
}

func NewChat(userID string, initialSystemMessage *Message, chatConfig *ChatConfig) (*Chat, error) {
	chat := &Chat{
		ID:                   uuid.New().String(),
		UserID:               userID,
		InitialSystemMessage: initialSystemMessage,
		Status:               "active",
		Config:               chatConfig,
		TokenUsage:           0,
	}
	chat.AddMessage(initialSystemMessage)
	
	if err := chat.Validate(); err != nil {
		return nil,err
	}
	return chat,nil
}

func (c *Chat) Validate() error {
	if c.UserID == "" {
		return errors.New("user id is empty")
	}

	if c.Status != "active" && c.Status != "ended" {
		return errors.New("invalid status")
	}

	if c.Config.Temperature < 0 || c.Config.Temperature > 2 {
		return errors.New("invalid temperature must be (0 - 2)")
	}

	return nil
}

func (c *Chat) AddMessage(m *Message) error {
	if c.Status == "ended" {
		return errors.New("chat is ended, m=no more messages allowed")
	}

	//percorrer as msgs para verificar a quatidade de tokens, se nao excedeu o limite do modelo do chatgpt
	for {
		//verificar se tem tokens disponiveis no modelo, verificando a qnt de tokens armazenados no chat somando com a nova msg
		if c.Config.Model.GetMaxToken() >= m.GetQTDTokens()+c.TokenUsage {
			c.Messages = append(c.Messages, m)
			c.RefreshTokenUsage()
			break
		}

		//se nao tiver espaco para msg, apaga a mais antiga, e inseri na lista de menssagens apagadas
		c.ErasedMessages = append(c.ErasedMessages, c.Messages[0])
		c.Messages = c.Messages[1:] //apaga a msg 0
		c.RefreshTokenUsage()
	}
	return nil
}

func (c *Chat) GetMessages() []*Message {
	return c.Messages
}

func (c *Chat) CountMessages() int {
	return len(c.Messages)
}

func (c *Chat) EndChat() {
	c.Status = "ended"
}

func (c *Chat) RefreshTokenUsage() {
	c.TokenUsage = 0
	for m := range c.Messages {
		//percorrer todas menssagens para somar a quantidade de tokens que cada msg tem
		c.TokenUsage += c.Messages[m].GetQTDTokens()
	}
}
