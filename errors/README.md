# Errors

tiles 框架标准错误模块，提供统一的结构化错误类型、哨兵错误定义及错误链支持。

## 特性

- **结构化错误**：每个错误携带错误码（`Code`）、面向用户的消息（`Message`）和内部诊断信息（`errMsg`）
- **哨兵错误模式**：预定义框架通用错误变量，支持跨层传递和精确匹配
- **错误链支持**：通过 `Wrap` 包装底层错误，兼容标准库 `errors.Is` / `errors.As`
- **不可变设计**：`WithErrMsg` / `Wrap` 均返回新实例，不修改原哨兵错误，并发安全
- **错误码规范**：8 位结构化错误码，区分严重程度、服务来源和序号

## 快速开始

```go
import (
    "errors"
    tileerrors "github.com/CXeon/tiles/errors"
)

// 直接返回哨兵错误
return tileerrors.ErrNotFound

// 附加内部诊断信息（不会暴露给用户）
return tileerrors.ErrNotFound.WithErrMsg(fmt.Sprintf("uid=%s not found in db", uid))

// 包装底层错误，保留错误链
return tileerrors.ErrInternal.Wrap(dbErr)

// 链式使用：同时附加诊断信息和底层错误
return tileerrors.ErrInternal.WithErrMsg("insert failed").Wrap(dbErr)

// 错误判断（按错误码匹配，忽略 WithErrMsg / Wrap 包装）
if errors.Is(err, tileerrors.ErrNotFound) {
    // 处理资源不存在的情况
}

// 获取内部诊断信息（仅用于日志，不写入响应）
if e, ok := err.(*tileerrors.Error); ok {
    log.Error("internal", "err", e.Internal())
}
```

## Error 结构体

```go
type Error struct {
    Code    uint   // 错误码
    Message string // 面向用户的消息，可直接写入 HTTP 响应
    // errMsg：内部诊断信息，通过 Internal() 获取，不对外暴露
    // cause：底层错误，通过 Unwrap() 支持 errors.Is / errors.As
}
```

| 方法 | 说明 |
|---|---|
| `New(code, message)` | 创建新错误 |
| `Error()` | 返回面向用户的消息（实现 `error` 接口） |
| `WithErrMsg(msg)` | 附加内部诊断信息，返回新实例 |
| `Wrap(cause)` | 包装底层错误，返回新实例 |
| `Internal()` | 获取内部诊断信息，仅用于日志 |
| `Is(target)` | 按错误码匹配，支持 `errors.Is` |
| `Unwrap()` | 返回底层错误，支持 `errors.As` |

## 框架通用哨兵错误

服务ID = `000`，错误码段：`20000001` - `20009999`

| 错误码 | 变量名 | Message |
|---|---|---|
| `20000001` | `ErrNotFound` | 资源不存在 |
| `20000002` | `ErrUnauthorized` | 未授权 |
| `20000003` | `ErrForbidden` | 权限不足 |
| `20000004` | `ErrConflict` | 资源已存在 |
| `20000005` | `ErrBadRequest` | 请求参数有误 |
| `20000006` | `ErrInternal` | 服务内部错误 |

## 错误码规范

错误码为 **8 位数字**，由三段组成：

```
[严重程度(1位)] [服务ID(3位)] [序号(4位)]
```

| 字段 | 位数 | 说明 |
|---|---|---|
| 严重程度 | 1 | `1` = 警告（业务可继续），`2` = 错误（业务中断） |
| 服务ID | 3 | 标识错误所属的服务或框架，取值 `000`-`999` |
| 序号 | 4 | 该服务内部的错误编号，取值 `0001`-`9999` |

### 示例

```
2  001  0003
│   │    └── 序号：该服务第 3 个错误
│   └─────── 服务ID：001 = user 服务
└─────────── 严重程度：2 = 错误，业务中断
```

### 成功码

| 错误码 | 含义 |
|---|---|
| `0` | 成功 |

### 严重程度

| 首位 | 含义 | 适用场景 |
|---|---|---|
| `1` | 警告 | 当前操作有提示，但业务流程可以继续 |
| `2` | 错误 | 业务流程中断，需要客户端处理 |

## 在业务服务中定义错误

在各服务的领域层创建 `errors.go`，引用 `tiles/errors` 包定义哨兵错误：

```go
package user

import tileerrors "github.com/CXeon/tiles/errors"

// 服务ID = 001，错误码段：20010001 - 20019999
var (
    ErrUserNotFound   = tileerrors.New(20010001, "用户不存在")
    ErrUserSaveFail   = tileerrors.New(20010002, "保存用户失败")
    ErrUserDeleteFail = tileerrors.New(20010003, "删除用户失败")
    ErrUserConflict   = tileerrors.New(20010004, "用户已存在")
)
```

## 运行测试

```bash
go test github.com/CXeon/tiles/errors/... -v
```

## 相关链接

- [tiles 项目主页](../../README.md)
- [标准库 errors 包文档](https://pkg.go.dev/errors)
