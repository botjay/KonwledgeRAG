package rag

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

// RAGConfig RAG系统配置
type RAGConfig struct {
	DocsDir        string
	OpenAIKey      string
	OpenAIProxy    string
	OpenAIModel    string
	EmbeddingModel string
	MaxTokens      int
	VectorStore    VectorStoreConfig
}

// RAGService RAG服务
type RAGService struct {
	config        RAGConfig
	docLoader     *DocumentLoader
	vectorStore   *VectorStore
	llmClient     *LLMClient
	docsProcessed bool
}

// NewRAGService 创建一个新的RAG服务
func NewRAGService(config RAGConfig) (*RAGService, error) {
	// 创建文档加载器
	docLoader := NewDocumentLoader(config.DocsDir)

	// 创建OpenAI客户端
	openaiClient, err := openai.New(
		openai.WithToken(config.OpenAIKey),
		openai.WithBaseURL(config.OpenAIProxy),
		openai.WithEmbeddingModel(config.EmbeddingModel),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化OpenAI客户端失败: %w", err)
	}

	// 创建嵌入模型
	embedder, err := embeddings.NewEmbedder(openaiClient)
	if err != nil {
		return nil, fmt.Errorf("初始化嵌入模型失败: %w", err)
	}

	// 创建向量存储
	vectorStore, err := NewVectorStore(embedder, config.VectorStore)
	if err != nil {
		return nil, fmt.Errorf("初始化向量存储失败: %w", err)
	}

	// 创建LLM客户端
	llmClient, err := NewLLMClient(LLMConfig{
		APIKey:    config.OpenAIKey,
		ProxyURL:  config.OpenAIProxy,
		Model:     config.OpenAIModel,
		MaxTokens: config.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化LLM客户端失败: %w", err)
	}

	return &RAGService{
		config:      config,
		docLoader:   docLoader,
		vectorStore: vectorStore,
		llmClient:   llmClient,
	}, nil
}

// LoadAndProcessDocuments 加载并处理文档
func (s *RAGService) LoadAndProcessDocuments(ctx context.Context) error {

	// 加载文档
	docs, err := s.docLoader.LoadDocuments(ctx)
	if err != nil {
		return fmt.Errorf("加载文档失败: %w", err)
	}

	// 分割文档
	splitDocs, err := s.docLoader.SplitDocuments(docs)
	if err != nil {
		return fmt.Errorf("分割文档失败: %w", err)
	}

	// 添加到向量存储
	err = s.vectorStore.AddDocuments(ctx, splitDocs)
	if err != nil {
		return fmt.Errorf("添加文档到向量存储失败: %w", err)
	}

	s.docsProcessed = true
	return nil
}

// Query 查询RAG系统
func (s *RAGService) Query(ctx context.Context, query string) (string, error) {
	if !s.docsProcessed {
		return "", fmt.Errorf("文档尚未处理，请先调用LoadAndProcessDocuments")
	}

	// 执行相似性搜索
	docs, err := s.vectorStore.SimilaritySearch(ctx, query, 5)
	if err != nil {
		return "", fmt.Errorf("相似性搜索失败: %w", err)
	}

	// 构建上下文
	context := ""
	for i, doc := range docs {
		context += fmt.Sprintf("文档 %d:\n%s\n\n", i+1, doc.PageContent)
	}

	// 创建问答链
	chain := s.llmClient.CreateQAChain()

	// 执行问答链
	result, err := chain.Call(ctx, map[string]any{
		"context":  context,
		"question": query,
	})
	if err != nil {
		return "", fmt.Errorf("执行问答链失败: %w", err)
	}

	return result["text"].(string), nil
}

// SetDocsProcessed 设置文档处理状态
func (s *RAGService) SetDocsProcessed(processed bool) {
	s.docsProcessed = processed
}
