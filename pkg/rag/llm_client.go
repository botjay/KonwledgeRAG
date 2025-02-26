package rag

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

// LLMConfig 大语言模型配置
type LLMConfig struct {
	APIKey    string
	ProxyURL  string
	Model     string
	MaxTokens int
}

// LLMClient 大语言模型客户端
type LLMClient struct {
	config LLMConfig
	llm    llms.LLM
}

// NewLLMClient 创建一个新的LLM客户端
func NewLLMClient(config LLMConfig) (*LLMClient, error) {
	opts := []openai.Option{
		openai.WithToken(config.APIKey),
		openai.WithBaseURL(config.ProxyURL),
	}

	if config.Model != "" {
		opts = append(opts, openai.WithModel(config.Model))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("初始化LLM失败: %w", err)
	}

	return &LLMClient{
		config: config,
		llm:    llm,
	}, nil
}

// Call 直接调用LLM
func (c *LLMClient) Call(ctx context.Context, prompt string) (string, error) {
	response, err := c.llm.Call(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("调用LLM失败: %w", err)
	}
	return response, nil
}

// GetLLM 获取底层LLM
func (c *LLMClient) GetLLM() llms.LLM {
	return c.llm
}

// CreateQAChain 创建问答链
func (c *LLMClient) CreateQAChain() chains.Chain {
	// 创建一个简单的提示模板
	template := `使用以下上下文来回答问题。如果你不知道答案，只需说不知道，不要试图编造答案。

上下文:
{{.context}}

问题: {{.question}}

回答:`

	// 创建提示模板
	prompt := prompts.NewPromptTemplate(
		template,
		[]string{"context", "question"},
	)

	// 创建LLM链
	return chains.NewLLMChain(c.llm, prompt)
}
