package docgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"bytes"

	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

const defaultGitPath = "." // 默认为当前目录

// EnumGroup 表示一个枚举分组
type EnumGroup struct {
	Name        string     `json:"name"`        // 枚举组名称
	Description string     `json:"description"` // 枚举组描述
	Package     string     `json:"package"`     // 包路径
	File        string     `json:"file"`        // 文件路径
	Type        string     `json:"type"`        // 类型（const/var/type）
	Items       []EnumItem `json:"items"`       // 枚举项
	Tags        []string   `json:"tags"`        // 相关标签，用于搜索
	Category    string     `json:"category"`    // 分类（如：状态、类型、标志等）
}

// EnumItem 表示具体的枚举项
type EnumItem struct {
	Name        string      `json:"name"`        // 枚举名称
	Value       interface{} `json:"value"`       // 枚举值
	Comment     string      `json:"comment"`     // 注释说明
	Description string      `json:"description"` // 详细描述
	Example     string      `json:"example"`     // 使用示例
}

type TableComment struct {
	TableName string
	Comment   string
	Fields    []FieldComment
}

type FieldComment struct {
	FieldName string
	FieldType string // 添加字段类型
	Comment   string
}

type Parser struct {
	enums      map[string]*EnumGroup
	dbComments map[string]TableComment
}

func NewParser() *Parser {
	return &Parser{
		enums:      make(map[string]*EnumGroup),
		dbComments: make(map[string]TableComment),
	}
}

func (p *Parser) ParseEnums(rootPath string) (map[string]*EnumGroup, error) {
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := p.parseFile(path); err != nil {
				return err
			}
		}
		return nil
	})

	return p.enums, err
}

func (p *Parser) ParseDBComments(rootPath string) (map[string]TableComment, error) {
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".sql") {
			if err := p.parseSQLFile(path); err != nil {
				return err
			}
		}
		return nil
	})

	return p.dbComments, err
}

func (p *Parser) parseFile(filename string) error {
	if !strings.HasSuffix(filename, ".go") {
		return nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// 获取相对路径
	relPath, err := filepath.Rel(defaultGitPath, filename)
	if err != nil {
		relPath = filename
	}

	// 遍历所有声明
	for _, decl := range node.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			switch gen.Tok {
			case token.CONST, token.VAR:
				// 检查前面的注释是否包含 @ai 标签
				if gen.Doc != nil {
					for _, comment := range gen.Doc.List {
						if strings.Contains(comment.Text, "@ai") {
							// 提取注释内容（去掉 @ai 标记）
							content := strings.TrimSpace(strings.Replace(comment.Text, "// @ai", "", 1))

							// 解析声明组
							group := p.parseEnumGroup(gen, content, node.Name.Name, relPath)
							if group != nil {
								// 生成标签
								group.Tags = p.generateTags(group)
								// 推断分类
								group.Category = p.inferCategory(group)

								// 合并或添加到现有组
								if existing, ok := p.enums[group.Name]; ok {
									p.mergeEnumGroup(existing, group)
								} else {
									p.enums[group.Name] = group
								}
							}
							break // 找到 @ai 标签后就跳出
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *Parser) parseEnumGroup(gen *ast.GenDecl, docComment, pkgName, filePath string) *EnumGroup {
	group := &EnumGroup{
		Package:     pkgName,
		File:        filePath,
		Type:        gen.Tok.String(),
		Description: docComment,
		Items:       make([]EnumItem, 0),
	}

	// 设置组名
	group.Name = fmt.Sprintf("%s %s", p.getEnumGroupName(gen), docComment)

	// 解析枚举项
	items := p.parseEnumItems(gen, docComment)
	if len(items) > 0 {
		group.Items = items
		return group
	}

	return nil
}

// 生成搜索标签
func (p *Parser) generateTags(group *EnumGroup) []string {
	tags := make(map[string]bool)

	// 从名称中提取关键词
	words := splitCamelCase(group.Name)
	for _, word := range words {
		tags[strings.ToLower(word)] = true
	}

	// 从描述中提取关键词
	words = extractKeywords(group.Description)
	for _, word := range words {
		tags[strings.ToLower(word)] = true
	}

	// 从枚举项中提取关键词
	for _, item := range group.Items {
		words = splitCamelCase(item.Name)
		for _, word := range words {
			tags[strings.ToLower(word)] = true
		}
		words = extractKeywords(item.Comment)
		for _, word := range words {
			tags[strings.ToLower(word)] = true
		}
	}

	// 转换为切片
	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// 推断枚举分类
func (p *Parser) inferCategory(group *EnumGroup) string {
	name := strings.ToLower(group.Name)
	switch {
	case strings.Contains(name, "status") || strings.Contains(name, "state"):
		return "状态"
	case strings.Contains(name, "type"):
		return "类型"
	case strings.Contains(name, "flag"):
		return "标志"
	case strings.Contains(name, "mode"):
		return "模式"
	case strings.Contains(name, "level"):
		return "级别"
	default:
		return "其他"
	}
}

// 辅助函数：驼峰命名分词
func splitCamelCase(s string) []string {
	var words []string
	var current string

	for _, r := range s {
		if unicode.IsUpper(r) {
			if current != "" {
				words = append(words, current)
			}
			current = string(r)
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// 辅助函数：从文本中提取关键词
func extractKeywords(text string) []string {
	// 移除常见的无意义词
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
	}

	words := strings.Fields(text)
	var keywords []string

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ",.()[]{}\"'"))
		if !stopWords[word] && len(word) > 2 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// 获取枚举组名称
func (p *Parser) getEnumGroupName(gen *ast.GenDecl) string {
	// 如果只有一个规范且有类型，使用类型作为组名
	if len(gen.Specs) == 1 {
		if spec, ok := gen.Specs[0].(*ast.ValueSpec); ok && spec.Type != nil {
			if ident, ok := spec.Type.(*ast.Ident); ok {
				return ident.Name
			}
		}
	}

	// 否则尝试从第一个规范获取前缀
	if len(gen.Specs) > 0 {
		if spec, ok := gen.Specs[0].(*ast.ValueSpec); ok && len(spec.Names) > 0 {
			name := spec.Names[0].Name
			// 尝试提取通用前缀
			for i := len(name) - 1; i >= 0; i-- {
				if name[i] >= 'A' && name[i] <= 'Z' {
					return name[:i]
				}
			}
			return name
		}
	}

	return "Unknown"
}

// 解析枚举项
func (p *Parser) parseEnumItems(gen *ast.GenDecl, groupComment string) []EnumItem {
	var items []EnumItem

	for _, spec := range gen.Specs {
		if vspec, ok := spec.(*ast.ValueSpec); ok {
			for i, name := range vspec.Names {
				item := EnumItem{
					Name: name.Name,
				}

				// 获取值
				if i < len(vspec.Values) {
					switch v := vspec.Values[i].(type) {
					case *ast.BasicLit:
						item.Value = v.Value
					case *ast.Ident:
						item.Value = v.Name
					case *ast.SelectorExpr:
						if x, ok := v.X.(*ast.Ident); ok {
							item.Value = fmt.Sprintf("%s.%s", x.Name, v.Sel.Name)
						}
					}
				}

				// 获取注释
				if vspec.Comment != nil {
					item.Comment = strings.TrimSpace(vspec.Comment.Text())
				} else if vspec.Doc != nil {
					item.Comment = strings.TrimSpace(vspec.Doc.Text())
				} else if groupComment != "" {
					item.Comment = groupComment
				}

				items = append(items, item)
			}
		}
	}

	return items
}

func (p *Parser) parseSQLFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// 预处理 SQL 内容
	sqlContent := string(content)
	sqlContent = strings.ReplaceAll(sqlContent, "\r\n", "\n")
	sqlContent = strings.TrimPrefix(sqlContent, "\xef\xbb\xbf")

	// 分割成单独的语句
	statements := splitSQLStatements(sqlContent)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// 尝试提取建表语句
		if strings.HasPrefix(strings.ToUpper(stmt), "CREATE TABLE") {
			if err := p.parseCreateTableByString(stmt); err != nil {
				fmt.Printf("警告: 解析建表语句出错 (文件: %s): %v\n语句内容: %s\n",
					filename, err, stmt)
			}
			continue
		}

		// 尝试提取注释语句
		if strings.HasPrefix(strings.ToUpper(stmt), "COMMENT ON") {
			if err := p.parseCommentByString(stmt); err != nil {
				fmt.Printf("警告: 解析注释语句出错 (文件: %s): %v\n语句内容: %s\n",
					filename, err, stmt)
			}
			continue
		}
	}

	return nil
}

func splitSQLStatements(sql string) []string {
	var statements []string
	var currentStmt strings.Builder

	// 按行处理，保留原始换行
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 跳过空行
		if trimmedLine == "" {
			continue
		}

		// 处理单行注释
		if strings.HasPrefix(trimmedLine, "--") {
			continue
		}

		// 处理多行注释
		if strings.Contains(trimmedLine, "/*") && !strings.Contains(trimmedLine, "*/") {
			continue
		}

		// 移除行内注释
		if idx := strings.Index(trimmedLine, "--"); idx >= 0 {
			trimmedLine = strings.TrimSpace(trimmedLine[:idx])
		}

		// 跳过空行（移除注释后）
		if trimmedLine == "" {
			continue
		}

		// 添加到当前语句
		if currentStmt.Len() > 0 {
			currentStmt.WriteString(" ")
		}
		currentStmt.WriteString(trimmedLine)

		// 检查语句是否结束
		if strings.HasSuffix(trimmedLine, ";") {
			stmt := currentStmt.String()
			if isRelevantStatement(stmt) {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
		}
	}

	// 处理最后一个语句（如果没有分号）
	lastStmt := strings.TrimSpace(currentStmt.String())
	if lastStmt != "" {
		if !strings.HasSuffix(lastStmt, ";") {
			lastStmt += ";"
		}
		if isRelevantStatement(lastStmt) {
			statements = append(statements, lastStmt)
		}
	}

	return statements
}

func isRelevantStatement(stmt string) bool {
	upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
	// 处理 CREATE TABLE 和 COMMENT ON 语句
	return strings.HasPrefix(upperStmt, "CREATE TABLE") ||
		strings.HasPrefix(upperStmt, "COMMENT ON") ||
		strings.HasPrefix(upperStmt, "ALTER TABLE")
}

func (p *Parser) parseCreateTableByString(stmt string) error {
	// 提取表名，支持 public. 前缀和双引号
	tableNameRegex := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"?public"?\.)?"?([^"\s(]+)"?`)
	matches := tableNameRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("无法提取表名")
	}

	// 处理表名
	tableName := strings.Trim(matches[1], `"`)
	tableName = strings.TrimPrefix(tableName, "public.")

	// 提取字段定义
	fieldsRegex := regexp.MustCompile(`\((.*)\)`)
	matches = fieldsRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("无法提取字段定义")
	}

	// 分割字段定义，但保持括号内的内容完整
	fieldDefs := splitFields(matches[1])
	var fields []FieldComment
	for _, fieldDef := range fieldDefs {
		fieldDef = strings.TrimSpace(fieldDef)
		if fieldDef == "" || strings.HasPrefix(strings.ToUpper(fieldDef), "PRIMARY KEY") ||
			strings.HasPrefix(strings.ToUpper(fieldDef), "CONSTRAINT") {
			continue
		}

		// 提取字段名和类型
		field := parseFieldDef(fieldDef)
		if field != nil {
			fields = append(fields, *field)
		}
	}

	// 存储表信息，使用不带 public. 前缀的表名
	p.dbComments[tableName] = TableComment{
		TableName: tableName,
		Fields:    fields,
	}

	return nil
}

// 辅助函数：分割字段定义，保持括号内的内容完整
func splitFields(fields string) []string {
	var result []string
	var current strings.Builder
	var parenCount int

	for i := 0; i < len(fields); i++ {
		char := fields[i]
		switch char {
		case '(':
			parenCount++
			current.WriteByte(char)
		case ')':
			parenCount--
			current.WriteByte(char)
		case ',':
			if parenCount == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteByte(char)
			}
		default:
			current.WriteByte(char)
		}
	}

	// 添加最后一个字段
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// 辅助函数：解析单个字段定义
func parseFieldDef(fieldDef string) *FieldComment {
	// 移除多余的空格
	fieldDef = strings.TrimSpace(fieldDef)

	// 分割字段定义为部分
	parts := strings.Fields(fieldDef)
	if len(parts) < 2 {
		return nil
	}

	fieldName := strings.Trim(parts[0], `"`)

	// 提取类型（可能包含括号和参数）
	typeStart := strings.Index(fieldDef, parts[1])
	restDef := fieldDef[typeStart:]
	fieldType := extractType(restDef)

	return &FieldComment{
		FieldName: fieldName,
		FieldType: fieldType,
	}
}

// 辅助函数：提取完整的类型定义
func extractType(s string) string {
	var result strings.Builder
	var parenCount int
	var inType bool = true

	for i := 0; i < len(s) && inType; i++ {
		char := s[i]
		switch {
		case char == '(':
			parenCount++
			result.WriteByte(char)
		case char == ')':
			parenCount--
			result.WriteByte(char)
		case char == ' ' && parenCount == 0:
			inType = false
		default:
			if inType {
				result.WriteByte(char)
			}
		}
	}

	return strings.TrimSpace(result.String())
}

func (p *Parser) parseCommentByString(stmt string) error {
	// 提取注释内容，支持大小写不敏感的 COMMENT 和 IS 关键字，以及 TABLE 关键字
	// 修改正则表达式以更好地处理 public. 前缀
	commentRegex := regexp.MustCompile(`(?i)COMMENT\s+ON\s+(?:TABLE\s+|COLUMN\s+)?(?:public\.)?([^\s]+)\s+IS\s+'([^']*)'`)
	matches := commentRegex.FindStringSubmatch(stmt)
	if len(matches) < 3 {
		// 如果没有匹配到，尝试使用双引号的版本
		commentRegex = regexp.MustCompile(`(?i)COMMENT\s+ON\s+(?:TABLE\s+|COLUMN\s+)?(?:public\.)?([^\s]+)\s+IS\s+"([^"]*)"`)
		matches = commentRegex.FindStringSubmatch(stmt)
		if len(matches) < 3 {
			return fmt.Errorf("无法提取注释内容")
		}
	}

	target := strings.Trim(matches[1], `"`)
	comment := decodeComment(matches[2])

	// 处理表注释
	if !strings.Contains(target, ".") {
		// 移除可能的 public. 前缀
		target = strings.TrimPrefix(target, "public.")
		if table, ok := p.dbComments[target]; ok {
			table.Comment = comment
			p.dbComments[target] = table
		} else {
			p.dbComments[target] = TableComment{
				TableName: target,
				Comment:   comment,
				Fields:    []FieldComment{},
			}
		}
		return nil
	}

	// 处理字段注释
	parts := strings.Split(target, ".")
	if len(parts) < 2 {
		return fmt.Errorf("无效的字段引用")
	}

	// 移除可能的 public. 前缀
	tableName := strings.TrimPrefix(parts[len(parts)-2], "public.")
	tableName = strings.Trim(tableName, `"`)
	fieldName := strings.Trim(parts[len(parts)-1], `"`)

	if table, ok := p.dbComments[tableName]; ok {
		for i := range table.Fields {
			if table.Fields[i].FieldName == fieldName {
				table.Fields[i].Comment = comment
				break
			}
		}
		p.dbComments[tableName] = table
	}

	return nil
}

func decodeComment(s string) string {
	// 如果已经是有效的 UTF-8 字符串，直接返回
	if utf8.ValidString(s) {
		return s
	}

	// 尝试 GBK 解码
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewDecoder())
	if d, err := ioutil.ReadAll(reader); err == nil && utf8.Valid(d) {
		return string(d)
	}

	// 尝试 Big5 解码
	reader = transform.NewReader(bytes.NewReader([]byte(s)), traditionalchinese.Big5.NewDecoder())
	if d, err := ioutil.ReadAll(reader); err == nil && utf8.Valid(d) {
		return string(d)
	}

	// 尝试 GB18030 解码
	reader = transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GB18030.NewDecoder())
	if d, err := ioutil.ReadAll(reader); err == nil && utf8.Valid(d) {
		return string(d)
	}

	// 如果所有解码尝试都失败了，返回原始字符串
	return s
}

func (p *Parser) mergeEnumGroup(existing, new *EnumGroup) {
	// 合并描述
	if new.Description != "" && existing.Description != new.Description {
		existing.Description += "\n" + new.Description
	}

	// 合并枚举项
	existingItems := make(map[string]bool)
	for _, item := range existing.Items {
		existingItems[item.Name] = true
	}

	for _, item := range new.Items {
		if !existingItems[item.Name] {
			existing.Items = append(existing.Items, item)
		}
	}

	// 合并标签
	existingTags := make(map[string]bool)
	for _, tag := range existing.Tags {
		existingTags[tag] = true
	}

	for _, tag := range new.Tags {
		if !existingTags[tag] {
			existing.Tags = append(existing.Tags, tag)
		}
	}

	// 保持标签排序
	sort.Strings(existing.Tags)
}

// 添加新的结构体方法来生成 Markdown
func (p *Parser) ToMarkdown() string {
	var md strings.Builder

	// 生成枚举文档
	if len(p.enums) > 0 {
		md.WriteString("# 枚举类型\n\n")
		for _, enum := range p.enums {
			md.WriteString(fmt.Sprintf("## %s\n\n", enum.Name))
			// 使用更友好的标签格式
			if len(enum.Tags) > 0 {
				md.WriteString("**标签：** ")
				for i, tag := range enum.Tags {
					if i > 0 {
						md.WriteString(" · ")
					}
					md.WriteString(fmt.Sprintf("`%s`", tag))
				}
				md.WriteString("\n\n")
			}
			md.WriteString("| 变量 | 原值 | 描述 |\n|---|---|---|\n")
			for _, value := range enum.Items {
				valueStr := ""
				if value.Value != nil {
					valueStr = fmt.Sprintf("%v", value.Value)
				}
				md.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
					value.Name, valueStr, value.Comment))
			}
			md.WriteString("\n")
		}
	}

	// 生成数据库表文档
	if len(p.dbComments) > 0 {
		md.WriteString("# 数据库表\n\n")
		// 先对表名进行排序，保证输出顺序一致
		var tableNames []string
		for tableName := range p.dbComments {
			tableNames = append(tableNames, tableName)
		}
		sort.Strings(tableNames)

		for _, tableName := range tableNames {
			table := p.dbComments[tableName]
			// 如果有表注释，将其添加到表名后面
			if table.Comment != "" {
				md.WriteString(fmt.Sprintf("## %s（%s）\n\n", tableName, table.Comment))
			} else {
				md.WriteString(fmt.Sprintf("## %s\n\n", tableName))
			}

			md.WriteString("| 字段 | 类型 | 描述 |\n|---|---|---|\n")
			for _, field := range table.Fields {
				comment := field.Comment
				if comment == "" {
					comment = "-" // 如果没有注释，显示一个占位符
				}
				md.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
					field.FieldName, field.FieldType, comment))
			}
			md.WriteString("\n")
		}
	}

	return md.String()
}

func removeDollarQuotes(sql string) string {
	// 匹配 $$ 或 $tag$ 之间的内容
	dollarRegex := regexp.MustCompile(`\$[^$]*\$.*?\$[^$]*\$`)
	return dollarRegex.ReplaceAllString(sql, "''")
}

func removeMultilineComments(sql string) string {
	commentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	return commentRegex.ReplaceAllString(sql, "")
}

// 辅助函数：标准化空白字符
func normalizeWhitespace(s string) string {
	// 将多个空白字符替换为单个空格
	return strings.Join(strings.Fields(s), " ")
}
