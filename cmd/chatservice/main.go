package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ruhancs/virtual-assistant/config"
	"github.com/ruhancs/virtual-assistant/internal/infra/repository"
	"github.com/ruhancs/virtual-assistant/internal/infra/web"
	"github.com/ruhancs/virtual-assistant/internal/infra/web/webserver"
	chatcompletion "github.com/ruhancs/virtual-assistant/internal/usecase/chat_completion"
	//chatcompletionstream "github.com/ruhancs/virtual-assistant/internal/usecase/chat_completion_stream"
	"github.com/sashabaranov/go-openai"
)

func main() {
	configs,err := config.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	conn, err := sql.Open(configs.DBDriver, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		configs.DBUser, configs.DBPassword, configs.DBHost, configs.DBPort, configs.DBName))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	repository := repository.NewChatRepositoryMySql(conn)
	client := openai.NewClient(configs.OpenAIApiKey)

	chatConfig := chatcompletion.ChatCompletionConfigInputDTO{
		Model:                configs.Model,
		ModelMaxTokens:       configs.ModelMaxTokens,
		Temperature:          float32(configs.Temperature),
		TopP:                 float32(configs.TopP),
		N:                    configs.N,
		Stop:                 configs.Stop,
		MaxTokens:            configs.MaxTokens,
		InitialSystemMessage: configs.InitialChatMessage,
	}
	
	//chatConfigStream := chatcompletionstream.ChatCompletionConfigInputDTO{
	//	Model:                configs.Model,
	//	ModelMaxTokens:       configs.ModelMaxTokens,
	//	Temperature:          float32(configs.Temperature),
	//	TopP:                 float32(configs.TopP),
	//	N:                    configs.N,
	//	Stop:                 configs.Stop,
	//	MaxTokens:            configs.MaxTokens,
	//	InitialSystemMessage: configs.InitialChatMessage,
	//}

	//use case http
	usecase := chatcompletion.NewChatCompletionUseCase(repository,client)

	//usecase grpc
	//streamChan := make(chan chatcompletionstream.ChatCompletionOutputDTO)
	//streamUseCase := chatcompletionstream.NewChatCompletionUseCase(repository,client,streamChan)

	//config do web server com rota e handle
	webserver := webserver.NewWebServer(":" + configs.WebServerPort)
	webHandler := web.NewWebChatGPTHandler(*usecase,chatConfig,configs.AuthToken)
	webserver.AddHandler("/chat", webHandler.Handle)

	fmt.Println("Server Running on PORT: " + configs.WebServerPort)
	webserver.Start()
}