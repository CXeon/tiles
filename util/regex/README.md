# Util - Regex

提供常用正则表达式校验工具函数，内部使用预编译的 `*regexp.Regexp`，调用无额外开销。

## 可用函数

| 函数 | 说明 |
|------|------|
| `IsEmail(s string) bool` | 校验字符串是否为合法邮箱地址 |

## 快速开始

```go
import "github.com/CXeon/tiles/util/regex"

regex.IsEmail("user@example.com")   // true
regex.IsEmail("user.name+tag@sub.domain.org") // true
regex.IsEmail("invalid-email")      // false
regex.IsEmail("@nodomain.com")      // false
```

## 使用场景

在请求参数校验中使用：

```go
func RegisterHandler(c *gin.Context) {
    var req struct {
        Email string `json:"email"`
    }
    c.ShouldBindJSON(&req)

    if !regex.IsEmail(req.Email) {
        c.JSON(400, gin.H{"message": "邮箱格式无效"})
        return
    }
    // ...
}
```

## 运行测试

```bash
go test github.com/CXeon/tiles/util/regex/... -v
```

## 相关链接

- [tiles 项目主页](../../README.md)
