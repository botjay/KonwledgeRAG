# 智能问答系统 (RAG) 与代码文档生成工具

这个项目包含两个主要功能模块：基于检索增强生成（RAG）技术的智能问答系统和自动代码文档生成工具（DocGen）。

## 功能特点

### RAG 智能问答系统

- **文档处理**：自动加载、分割和向量化本地文档
- **语义检索**：使用OpenAI的嵌入模型进行高效的语义搜索
- **向量存储**：支持Qdrant向量数据库，提供高性能的相似度搜索
- **智能问答**：基于语言模型，结合检索到的上下文生成准确回答
- **持久化存储**：支持将向量数据持久化到磁盘，避免重复处理文档
- **灵活配置**：提供基于YAML的配置系统，方便调整各项参数

### DocGen 代码文档生成工具

- **枚举解析**：自动识别并解析Go代码中带有`@ai`标签的枚举定义
- **SQL解析**：解析SQL文件中的表结构定义和字段注释
- **文档生成**：自动生成Markdown格式的枚举和数据库表结构文档
- **智能分类**：根据枚举名称和内容自动推断分类（状态、类型、标志等）
- **标签生成**：自动从枚举名称和描述中提取关键词作为搜索标签
- **多编码支持**：支持处理不同编码格式的源文件

### 数据持久化与加载机制

系统采用智能的数据持久化和加载机制，具体工作方式如下：

1. **持久化存储**：
   - Qdrant向量数据库通过Docker卷挂载实现数据持久化
   - 数据存储在配置的`persist_dir`目录下（默认为`./data/qdrant`）
   - 即使程序或容器重启，数据也会保留在磁盘上

2. **文档加载控制**：
   - 使用`--skip-load`参数可以跳过文档加载过程，直接使用已存在的向量存储
   - 这适用于已经处理过文档并希望直接进行查询的场景
   - 不指定此参数时，系统会加载并处理文档目录中的所有文档

## 安装指南

### 前提条件

- Go 1.18或更高版本
- Docker（用于运行Qdrant向量数据库）
- OpenAI API密钥或兼容的API代理

### 安装步骤


1. 安装依赖：

```bash
go mod download
```

2. 启动Qdrant向量数据库并且开启持久化（仅RAG系统需要）：

```bash
docker run -d --name qdrant --network host -v $(pwd)/data/qdrant:/qdrant/storage qdrant/qdrant
```

## 使用方法

### 配置文件

系统使用YAML格式的配置文件，默认路径为`config.yaml`。配置文件包含以下主要部分：

```yaml
# 语言模型配置
llm:
  model: "DeepSeek-R1"
  embedding_model: "text-embedding-3-small"
  max_tokens: 1000

# 向量数据库配置
vector_store:
  type: "qdrant"
  url: "http://localhost:6333"
  persist_dir: "./data/qdrant"

# 文档配置
docs:
  dir: "./docs"

# API配置
api:
  openai_key: ""  # 默认为空，优先使用环境变量
  openai_proxy: "xxx"
```

您可以根据需要修改配置文件，或者使用命令行参数覆盖配置文件中的设置。

### RAG 智能问答系统

#### 环境变量配置

设置OpenAI API密钥和代理URL（可选）：

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_API_PROXY="https://your-proxy-url"  # 可选
```

#### 运行程序

1. 使用默认配置文件运行：

```bash
go run cmd/ai/main.go
```

2. 指定配置文件路径：

```bash
go run cmd/ai/main.go --config ./custom-config.yaml
```

3. 使用命令行参数覆盖配置：

```bash
go run cmd/ai/main.go --docs ./your-docs-directory --skip-load
```

#### 命令行参数

- `--config`：配置文件路径，默认为`config.yaml`
- `--docs`：文档目录路径，覆盖配置文件中的设置
- `--skip-load`：是否跳过加载文档，默认为`false`

### DocGen 代码文档生成工具

#### 运行程序

```bash
go run cmd/docgen/main.go --localpath /path/to/your/project --output ./docs
```

#### 命令行参数

- `--localpath`：要解析的本地项目路径
- `--output`：文档输出目录，默认为`./docs`

#### 代码标记规范

在Go代码中使用`@ai`标签标记需要生成文档的枚举：

## 示例

### DocGen 文档生成示例

```bash
# 为项目生成枚举和表结构文档
go run cmd/docgen/main.go --localpath ./example --output ./docs

# 生成的文档位于 ./docs/knowledge_your-project.md
```

生成的文档示例：

```markdown
# 枚举类型

## MailStatus 邮件发送状态枚举

**标签：** `completed` · `failed` · `mail` · `pending` · `sending` · `status` · `status 邮件发送状态枚举` · `发送中` · `发送失败` · `已完成` · `待发送` · `邮件发送状态枚举`

| 变量 | 原值 | 描述 |
|---|---|---|
| MailStatusPending | 1 | 待发送 |
| MailStatusSending | 2 | 发送中 |
| MailStatusCompleted | 3 | 已完成 |
| MailStatusFailed | 4 | 发送失败 |

# 数据库表

## order_details

| 字段 | 类型 | 描述 |
|---|---|---|
| id | bigint | 自增ID |
| trade_date | character | 交易日期（2006-01-02） |
| user_id | bigint | 用户id |
| order_id | bigint | 订单ID |
| currency | character | - |
| trade_amount | numeric | 成交金额 |
| trade_quantity | numeric | 成交数量 |
| order_status | character | 订单状态：init-初始化，pending-待处理，processing-处理中，completed-已完成，cancelled-已取消 |
| fee | numeric | - |
```


### RAG 智能问答示例

```bash
# 首次运行，加载文档
go run cmd/ai/main.go --docs ./docs

# 交互式问答
请输入查询，exit退出: 订单状态的枚举有哪些
回答: 资产表是 `ods_account_assets`。该表在多个文档中被反复定义，其字段包含资产相关的核心指标，如 `begin_amount`（期初金额）、`ledger_amount`（账本金额）、`market_value`（市场价值）、`total_asset`（总资产）等，且表名中的 `assets` 直接表明其用途。

# 后续运行，跳过加载文档
go run cmd/ai/main.go --skip-load
```


## 许可证

[MIT License](LICENSE)