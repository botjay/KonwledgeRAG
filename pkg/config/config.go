package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	LLM         LLMConfig         `yaml:"llm"`
	VectorStore VectorStoreConfig `yaml:"vector_store"`
	Docs        DocsConfig        `yaml:"docs"`
	API         APIConfig         `yaml:"api"`
}

// LLMConfig 语言模型配置
type LLMConfig struct {
	Model          string `yaml:"model"`
	EmbeddingModel string `yaml:"embedding_model"`
	MaxTokens      int    `yaml:"max_tokens"`
}

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	Type       string `yaml:"type"`        // 向量存储类型
	URL        string `yaml:"url"`         // 向量存储服务器地址
	PersistDir string `yaml:"persist_dir"` // 持久化目录
}

// DocsConfig 文档配置
type DocsConfig struct {
	Dir string `yaml:"dir"`
}

// APIConfig API配置
type APIConfig struct {
	OpenAIKey   string `yaml:"openai_key"`
	OpenAIProxy string `yaml:"openai_proxy"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filePath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量获取API密钥（如果设置了）
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.API.OpenAIKey = apiKey
	}

	// 从环境变量获取API代理（如果设置了）
	if apiProxy := os.Getenv("OPENAI_API_PROXY"); apiProxy != "" {
		config.API.OpenAIProxy = apiProxy
	}

	return &config, nil
}

// GetDefaultConfig 返回默认配置
func GetDefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Model:          "DeepSeek-R1",
			EmbeddingModel: "text-embedding-3-small",
			MaxTokens:      1000,
		},
		VectorStore: VectorStoreConfig{
			Type:       "qdrant",
			URL:        "http://localhost:6333",
			PersistDir: "./data/qdrant",
		},
		Docs: DocsConfig{
			Dir: "./docs",
		},
		API: APIConfig{
			OpenAIKey:   "xxx",
			OpenAIProxy: "xxx",
		},
	}
}
