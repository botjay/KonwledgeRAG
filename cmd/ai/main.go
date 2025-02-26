package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"enum_tools/pkg/config"
	"enum_tools/pkg/rag"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	docsDir := flag.String("docs", "./docs", "文档目录路径")
	skipLoad := flag.Bool("skip-load", false, "是否跳过加载文档")
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 命令行参数覆盖配置文件
	if *docsDir != "" {
		cfg.Docs.Dir = *docsDir
	}

	// 创建RAG服务配置
	ragConfig := rag.RAGConfig{
		DocsDir:        cfg.Docs.Dir,
		OpenAIKey:      cfg.API.OpenAIKey,
		OpenAIProxy:    cfg.API.OpenAIProxy,
		OpenAIModel:    cfg.LLM.Model,
		EmbeddingModel: cfg.LLM.EmbeddingModel,
		MaxTokens:      cfg.LLM.MaxTokens,
		VectorStore: rag.VectorStoreConfig{
			QdrantURL:      cfg.VectorStore.URL,
			CollectionName: "default_collection",
			PersistDir:     cfg.VectorStore.PersistDir,
		},
	}

	// 创建RAG服务
	ctx := context.Background()
	service, err := createRAGService(ctx, ragConfig, *skipLoad)
	if err != nil {
		log.Fatalf("创建RAG服务失败: %v", err)
	}

	// 交互式问答循环
	for {
		fmt.Print("请输入查询，exit退出: ")
		query := ""
		fmt.Scanln(&query)
		if query == "exit" {
			break
		}
		answer, err := service.Query(ctx, query)
		if err != nil {
			log.Fatalf("查询失败: %v", err)
		}
		fmt.Println("回答:", answer)
	}
}

// loadConfig 加载配置文件，如果文件不存在则使用默认配置
func loadConfig(configPath string) (*config.Config, error) {
	// 尝试加载配置文件
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// 如果文件不存在，使用默认配置
		if os.IsNotExist(err) {
			log.Printf("配置文件 %s 不存在，使用默认配置", configPath)
			return config.GetDefaultConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// createRAGService 创建RAG服务
func createRAGService(ctx context.Context, config rag.RAGConfig, skipLoad bool) (*rag.RAGService, error) {
	// 创建RAG服务
	service, err := rag.NewRAGService(config)
	if err != nil {
		return nil, fmt.Errorf("创建RAG服务失败: %w", err)
	}

	// 检查是否存在持久化的向量存储
	if skipLoad {
		log.Println("使用已持久化的向量存储")
		// 设置文档已处理标志，表示可以直接使用向量存储进行查询
		service.SetDocsProcessed(true)
	} else {
		log.Println("正在加载和处理文档...")
		err = service.LoadAndProcessDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("加载和处理文档失败: %w", err)
		}
		log.Println("文档加载和处理完成")
	}

	return service, nil
}
