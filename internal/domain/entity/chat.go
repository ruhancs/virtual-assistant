package entity

import "errors"

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

func (c *Chat) AddMessage(m *Message)error {
	if c.Status == "ended" {
		return errors.New("chat is ended, m=no more messages allowed")
	}

	//percorrer as msgs para verificar a quatidade de tokens, se nao excedeu o limite do modelo do chatgpt
	for {
		//verificar se tem tokens disponiveis no modelo, verificando a qnt de tokens armazenados no chat somando com a nova msg
		if c.Config.Model.GetMaxToken() >= m.GetQTDTokens() + c.TokenUsage {
			c.Messages = append(c.Messages, m)
			c.refreshTokenUsage()
			break
		}

		//se nao tiver espaco para msg, apaga a mais antiga, e inseri na lista de menssagens apagadas
		c.ErasedMessages = append(c.ErasedMessages, c.Messages[0])
		c.Messages = c.Messages[1:]//apaga a msg 0
		c.refreshTokenUsage()
	}
	return nil
}

func (c *Chat) refreshTokenUsage() {
	c.TokenUsage = 0
	for m := range c.Messages {
		//percorrer todas menssagens para somar a quantidade de tokens
		c.TokenUsage = c.Messages[m].GetQTDTokens()
	}
}