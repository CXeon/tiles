# Tiles

Tiles 是一个基于 Go 语言开发的微服务工具包集合，旨在为微服务项目提供常用的基础设施组件。正如其名称所示，Tiles 希望能像积木一样，为搭建应用贡献一份力量。

## 特性

- **模块化设计**：每个工具可以单独引用，无需导入整个项目
- **开箱即用**：提供微服务开发常用的基础组件
- **灵活扩展**：基于抽象接口设计，支持多种实现

## 项目结构

```
tiles/
└── gateway/              # 网关模块
    └── ...               # 网关抽象接口及具体实现
```

*注：未来计划支持更多模块，如日志（log）、上下文（context）、配置中心（config）、服务注册中心（service registry）等。*

## 模块说明

### Gateway（网关模块）

位于 `gateway/` 目录，提供服务网关的抽象接口和具体实现。

**核心功能**：
- 服务实例的动态注册、注销和更新
- 统一的网关抽象接口，屏蔽底层实现差异
- 支持多种网关实现和存储后端

**详细文档**：请参考 [gateway/traefik/README.md](gateway/traefik/README.md) 了解具体实现和使用方法

## 快速开始

### 安装

```bash
# 安装网关模块
go get github.com/CXeon/tiles/gateway
```

### 基本使用

各模块的详细使用方法请参考对应目录下的 README 文档：

- [Gateway 模块使用指南](gateway/traefik/README.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。
