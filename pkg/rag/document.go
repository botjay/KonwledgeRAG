package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// DocumentLoader 负责加载和处理文档
type DocumentLoader struct {
	DocsDir string
}

// NewDocumentLoader 创建一个新的文档加载器
func NewDocumentLoader(docsDir string) *DocumentLoader {
	return &DocumentLoader{
		DocsDir: docsDir,
	}
}

// LoadDocuments 加载目录中的所有 Markdown 文件
func (dl *DocumentLoader) LoadDocuments(ctx context.Context) ([]schema.Document, error) {
	var allDocs []schema.Document

	// 遍历目录下的所有 .md 文件
	err := filepath.Walk(dl.DocsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .md 文件
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			// 使用文件加载器加载文档
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("读取文件 %s 失败: %w", path, err)
			}

			// 创建文档
			doc := schema.Document{
				PageContent: string(content),
				Metadata: map[string]any{
					"source": path,
				},
			}

			allDocs = append(allDocs, doc)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return allDocs, nil
}

// SplitDocuments 将文档分割成更小的块
func (dl *DocumentLoader) SplitDocuments(docs []schema.Document) ([]schema.Document, error) {
	// 创建文本分割器
	splitter := textsplitter.NewTokenSplitter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)

	var splitDocs []schema.Document
	for _, doc := range docs {
		// 分割文本内容
		texts, err := splitter.SplitText(doc.PageContent)
		if err != nil {
			return nil, fmt.Errorf("分割文本失败: %w", err)
		}

		// 为每个分割后的文本创建新文档
		for _, text := range texts {
			splitDoc := schema.Document{
				PageContent: text,
				Metadata:    doc.Metadata,
			}
			splitDocs = append(splitDocs, splitDoc)
		}
	}

	return splitDocs, nil
}
