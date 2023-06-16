package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"os"
)

var (
	OpenAIToken = os.Getenv("OPENAI_TOKEN")
	messages    = make([]openai.ChatCompletionMessage, 0)
	client      = openai.NewClient(OpenAIToken)
)

func main() {
	ctx := context.Background()

	// 添加系统 Prompt
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "Don't make assumptions about what values to plug into functions. Ask for clarification if a user request is ambiguous.",
	})
	for {
		var userInput string
		fmt.Printf("%s: ", openai.ChatMessageRoleUser)
		fmt.Scanln(&userInput)
		msg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		}
		respMsg := handleRequest(ctx, msg)
		printMsg(respMsg)
		messages = append(messages, respMsg)
	}
}

func handleRequest(ctx context.Context, msg openai.ChatCompletionMessage) openai.ChatCompletionMessage {
	messages = append(messages, msg)
	funcs := functions()
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo0613,
		Functions: funcs,
		Messages:  messages,
	}
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		panic(err)
	}
	return handleResponse(ctx, &resp)
}

func handleResponse(ctx context.Context, resp *openai.ChatCompletionResponse) openai.ChatCompletionMessage {
	choice := resp.Choices[0]
	msg := choice.Message
	if msg.FunctionCall == nil {
		return msg
	}
	printMsg(msg)
	// 处理函数
	callInfo := msg.FunctionCall
	var funcResp string
	switch callInfo.Name {
	case "getBotInfo":
		funcResp = getBotInfo()
	case "getWalletBalance":
		funcResp = getWalletBalance(callInfo.Arguments)
	}
	funcMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleFunction,
		Content: funcResp,
		Name:    callInfo.Name,
	}
	printMsg(funcMsg)
	return handleRequest(ctx, funcMsg)
}

func getBotInfo() string {
	body := map[string]string{
		"name":     "空空",
		"features": "聊天,助手,查询钱包余额",
	}
	b, _ := json.Marshal(body)
	return string(b)
}

func getWalletBalance(body string) string {
	args := make(map[string]string)
	_ = json.Unmarshal([]byte(body), &args)
	userId := args["userId"]
	var balance int
	var name string
	switch userId {
	case "lisi":
		balance = 10000
		name = "李四"
	case "zhangsan":
		balance = 20000
		name = "张三"
	}
	resp := map[string]interface{}{
		"balance": balance,
		"user":    name,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func printMsg(msg openai.ChatCompletionMessage) {
	role := msg.Role
	content := msg.Content
	if content == "" {
		contentBytes, _ := json.Marshal(msg)
		content = string(contentBytes)
	}
	fmt.Printf("%s: %s\n", role, content)
}

func functions() []*openai.FunctionDefine {
	return []*openai.FunctionDefine{
		{
			Name:        "getBotInfo",
			Description: "获取机器人信息，在打招呼或自我介绍时可使用此函数获取名称以及功能",
			Parameters: &openai.FunctionParams{
				Type: "object",
				// 因为Properties不传接口会报错，传空map序列化后会忽略，所以这里传了一个无所谓的参数占位，解决序列化问题后可不传参数
				Properties: map[string]*openai.JSONSchemaDefine{
					"id": {
						Type:        "string",
						Description: "gpt模型自动生成的id",
					},
				},
			},
		},
		{
			Name:        "getWalletBalance",
			Description: "查询用户钱包余额",
			Parameters: &openai.FunctionParams{
				Type: "object",
				Properties: map[string]*openai.JSONSchemaDefine{
					"userId": {
						Type:        "string",
						Description: "用户id",
					},
				},
				Required: []string{"userId"},
			},
		},
	}
}
