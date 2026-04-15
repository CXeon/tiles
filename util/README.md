# Util（工具模块）

提供微服务开发中常用的轻量工具函数集合，无业务逻辑，各子包独立引用，按需使用。

## 子包概览

| 子包 | 包路径 | 功能 | 文档 |
|------|--------|------|------|
| **gormlog** | `github.com/CXeon/tiles/util/gormlog` | 将 tiles `logger.Logger` 适配为 GORM `logger.Interface` | [文档](gormlog/README.md) |
| **ip** | `github.com/CXeon/tiles/util/ip` | 获取本机第一个有效 IPv4 地址 | [文档](ip/README.md) |
| **regex** | `github.com/CXeon/tiles/util/regex` | 常用正则校验（邮箱等） | [文档](regex/README.md) |

## 相关链接

- [gormlog - GORM 日志适配器](gormlog/README.md)
- [ip - 本机 IP 获取](ip/README.md)
- [regex - 正则校验](regex/README.md)
- [tiles 项目主页](../README.md)
