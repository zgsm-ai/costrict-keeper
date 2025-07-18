# 编码规范

## 编程语言

使用GO语言进行开发,要求支持go 1.21版本。

## 框架

采用如下框架：

| 框架名 | 说明 |
|---|---|
| gin | 轻量级web框架 |
| gorm | 轻量级ORM框架 |
| github.com/glebarez/sqlite| sqlite3数据库 |
| github.com/go-redis/redis/v8 | 键值数据库,可用作缓存及消息队列 |
| logrus |  日志 |
| spf13/viper | 配置文件 |
| spf13/cobra | 命令行工具 |
| text/template | 文本模板 |
| github.com/swaggo/gin-swagger | swagger文档生成 |

## 结构

采用分层设计方式，从上到下分为四层：

- 交互层: 与用户进行交互的逻辑
- 业务层：处理用户的业务逻辑，也可以称为策略层。
- 机制层：业务逻辑依赖的基础机制，公共机制。
- 数据层/IO层：负责处理数据，调用数据库的方法，返回结果给机制层。

各层逻辑之间分离到不同的代码文件中，必要时保存在不同目录。

其中RESTful API接口实现部分放在controllers目录下；
业务层放在services目录下；
机制层放在internal目录下；
数据IO层放在dao目录下；

## 注释

所有注释内容，要求使用英文进行描述。

RESTful API实现函数，采用swagger注释标准进行注释，保证能生成所有API的swagger文档。

范例：

```go
// RenderPrompt render prompt template
// @Summary Render prompt template
// @Description Get prompt template by ID and render with input args
// @Tags Render
// @Accept json
// @Produce json 
// @Param prompt_id path string true "Prompt template ID"
// @Param args body string false "Template args" SchemaExample({"args":{"text":"Singleton pattern implementation"}})
// @Success 200 {object} map[string]interface{} "Rendered result"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 404 {object} map[string]interface{} "Template not found"
// @Failure 500 {object} map[string]interface{} "Render failed"
// @Router /v1/prompts/{prompt_id}/render [post]
func (pc *PromptController) RenderPrompt(c *gin.Context) {
}
```

其它函数，采用jsdoc风格给每个函数进行注释，说明函数的功能，参数，返回值，用法，注意事项等。

范例:

```go
/**
 * Upload file to server
 * @param {string} serverPath - Target storage path on server
 * @param {*resource.AI_File} file - File object to upload, contains metadata like Size
 * @returns {error} Returns error object, nil on success
 * @description
 * - Auto handles file size: small files upload directly, large files(>DEF_PART_SIZE) call PostHugeFile
 * - Sets HTTP headers: Content-Type, Cookie and Accept
 * - Processes server response and updates progress bar
 * @throws
 * - File stream error (createFileBuffer)
 * - POST request creation failure (http.NewRequest)
 * - HTTP request error (client.Do)
 * - Non-200 status code (statusToError)
 * @example
 * err := session.PostFile("/upload/path", file)
 * if err != nil {
 *     log.Fatal(err)
 * }
 */
func (ss *AI_Session) PostFile(serverPath string, file *resource.AI_File) error {
...
}
```

## 单元测试

每个函数都需要编写单元测试，单元测试需覆盖函数主体逻辑。

## 支持swagger文档

需要支持swagger文档。通过/swagger接口可以访问该应用的swagger文档展示页面。

## 指标监控

需要支持prometheus监控，向prometheus报告必要的指标数据。

## RESTful API错误处理

RESTful API发生错误，统一返回如下格式信息：

```json
{
    "code": "group.errortag",
    "message": "detail information"
}
```

其中code表示错误类型，用于程序分别错误类别，message提供详细信息供人类阅读。

## 业务就绪探针

作为服务，需要实现业务就绪探针，供K8s POD/docker探测服务是否已经做好准备。

探针接口使用`/healthz`作为接口地址。
