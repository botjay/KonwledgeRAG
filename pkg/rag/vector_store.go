package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	QdrantURL      string // Qdrant服务器地址
	CollectionName string // 集合名称
	PersistDir     string // 持久化目录
}

// DefaultVectorStoreConfig 返回默认配置
func DefaultVectorStoreConfig() VectorStoreConfig {
	return VectorStoreConfig{
		QdrantURL:      "http://localhost:6333",
		CollectionName: "default_collection",
		PersistDir:     "./data/qdrant", // 默认持久化目录
	}
}

// VectorStore 负责文档向量化和检索
type VectorStore struct {
	embedder    embeddings.Embedder
	vectorStore vectorstores.VectorStore
	config      VectorStoreConfig
}

// NewVectorStore 创建一个新的向量存储
func NewVectorStore(embedder embeddings.Embedder, config VectorStoreConfig) (*VectorStore, error) {
	// 验证配置
	if embedder == nil {
		return nil, fmt.Errorf("embedder不能为空")
	}

	// 确保有集合名称
	if config.CollectionName == "" {
		config.CollectionName = "default_collection"
	}

	// 确保有Qdrant URL
	if config.QdrantURL == "" {
		config.QdrantURL = "http://localhost:6333"
	}

	log.Printf("开始初始化向量存储，配置: %+v", config)

	// 测试嵌入模型
	log.Printf("正在测试嵌入模型...")
	testEmbed, err := embedder.EmbedQuery(context.Background(), "test")
	if err != nil {
		return nil, fmt.Errorf("嵌入模型测试失败: %w", err)
	}
	log.Printf("嵌入模型测试成功，向量维度: %d", len(testEmbed))

	// 解析URL
	qdrantURL, err := url.Parse(config.QdrantURL)
	if err != nil {
		return nil, fmt.Errorf("解析Qdrant URL失败: %w", err)
	}

	// 直接使用 REST API 创建集合
	err = createQdrantCollection(config.QdrantURL, config.CollectionName, len(testEmbed))
	if err != nil {
		log.Printf("使用REST API创建集合时出错: %v", err)
		// 这里我们不返回错误，因为集合可能已经存在
	}

	// 创建向量存储
	log.Printf("正在创建向量存储")

	// 准备Qdrant选项
	options := []qdrant.Option{
		qdrant.WithEmbedder(embedder),
		qdrant.WithCollectionName(config.CollectionName),
		qdrant.WithURL(*qdrantURL),
	}

	// 创建向量存储
	store, err := qdrant.New(options...)
	if err != nil {
		log.Printf("创建向量存储时出错: %v", err)
		return nil, fmt.Errorf("创建向量存储失败: %w", err)
	}

	log.Printf("成功创建向量存储，集合名称: %s", config.CollectionName)

	return &VectorStore{
		embedder:    embedder,
		vectorStore: store,
		config:      config,
	}, nil
}

// createQdrantCollection 使用 REST API 创建 Qdrant 集合
func createQdrantCollection(baseURL, collectionName string, dimensions int) error {
	// 构建请求URL
	createURL := fmt.Sprintf("%s/collections/%s", baseURL, collectionName)

	// 构建请求体
	requestBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimensions,
			"distance": "Cosine",
		},
		"optimizers_config": map[string]interface{}{
			"default_segment_number": 2,
		},
		"replication_factor":       1,
		"write_consistency_factor": 1,
	}

	// 转换为JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("PUT", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		// 读取响应体
		var responseBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
			return fmt.Errorf("解析响应体失败: %w", err)
		}
		return fmt.Errorf("创建集合失败，状态码: %d，响应: %v", resp.StatusCode, responseBody)
	}

	log.Printf("成功创建或确认集合 %s 存在", collectionName)
	return nil
}

// AddDocuments 添加文档到向量存储
func (vs *VectorStore) AddDocuments(ctx context.Context, docs []schema.Document) error {
	if len(docs) == 0 {
		log.Printf("没有文档需要添加")
		return nil
	}

	log.Printf("正在添加 %d 个文档", len(docs))

	// 直接添加文档
	_, err := vs.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("添加文档失败: %w", err)
	}

	log.Printf("成功添加所有文档")
	return nil
}

// SimilaritySearch 执行相似性搜索
func (vs *VectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]schema.Document, error) {
	log.Printf("正在执行相似性搜索，查询: %s", query)

	// 执行搜索
	docs, err := vs.vectorStore.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, fmt.Errorf("相似性搜索失败: %w", err)
	}

	log.Printf("搜索完成，找到 %d 个相关文档", len(docs))
	return docs, nil
}

// GetVectorStore 获取底层向量存储
func (vs *VectorStore) GetVectorStore() vectorstores.VectorStore {
	return vs.vectorStore
}
