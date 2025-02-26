package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"enum_tools/pkg/docgen"
)

var defaultGitPath string

func main() {
	outputPath := "./docs"
	flag.StringVar(&outputPath, "output", "docs", "输出文档目录")
	flag.StringVar(&defaultGitPath, "localpath", "", "本地项目路径")
	flag.Parse()

	if err := run(outputPath); err != nil {
		log.Fatal(err)
	}
}

func run(outputPath string) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 获取项目名称
	projectName := filepath.Base(defaultGitPath)

	parser := docgen.NewParser()

	// 解析枚举
	_, err := parser.ParseEnums(defaultGitPath)
	if err != nil {
		return fmt.Errorf("解析枚举失败: %w", err)
	}

	// 解析数据库注释
	_, err = parser.ParseDBComments(defaultGitPath)
	if err != nil {
		return fmt.Errorf("解析数据库注释失败: %w", err)
	}

	// 生成 Markdown 文档
	mdContent := parser.ToMarkdown()

	// 使用项目名称作为文件名
	mdFileName := fmt.Sprintf("%s/knowledge_%s.md", outputPath, projectName)
	err = os.WriteFile(mdFileName, []byte(mdContent), 0644)
	if err != nil {
		log.Fatalf("写入文档失败: %v", err)
	}

	return nil
}
