# Tiles

Tiles 是一个基于 Go 语言开发的微服务工具包集合,旨在为微服务项目提供常用的基础设施组件。正如其名称所示,Tiles 像瓦片一样铺设基础,让开发者能在屋顶之下自由构建丰富多彩的业务世界。

## 特性

- **包结构设计**：统一模块引用，一次安装即可使用所有组件
- **开箱即用**：提供微服务开发常用的基础组件
- **灵活扩展**：基于抽象接口设计，支持多种实现

## 项目结构

```
tiles/
├── config/               # 配置模块
│   └── ...               # 配置抽象接口及具体实现
├── gateway/              # 网关模块
│   └── ...               # 网关抽象接口及具体实现
├── logger/               # 日志模块
│   └── ...               # 日志抽象接口及具体实现
├── context/              # 上下文模块
│   └── ...               # 自定义上下文模块
└── registry/             # 服务注册中心模块
    └── ...               # 注册中心抽象接口及具体实现
```

## 模块说明

### Config（配置模块）

位于 `config/` 目录，提供统一的配置加载与热更新抽象接口和多种实现。

**核心功能**：
- 统一的配置接口，屏蔽本地文件与远程配置中心的差异
- 支持 `GetString`、`GetInt`、`GetBool`、`GetFloat64` 等强类型读取
- 结构体反序列化，支持 `Unmarshal` / `UnmarshalKey`（需 mapstructure tag）
- 配置热更新，通过 `Watch` 注册回调，精确感知新增、修改、删除变更
- 优雅关闭，`Close` 方法释放监听资源

**可用实现**：
- **Viper**：本地文件（YAML/JSON/TOML/ENV）+ 环境变量绑定，基于 fsnotify 热更新 - [文档](config/viper/README.md)
- **Apollo**：远程 Apollo 配置中心，服务端主动推送，支持本地备份容灾 - [文档](config/apollo/README.md)

### Logger（日志模块）

位于 `logger/` 目录，提供统一的日志抽象接口和多种主流日志库实现。

**核心功能**：
- 统一的日志接口，支持多种日志库实现（Zap、Logrus、Slog）
- 结构化日志，支持 JSON 格式输出
- 日志轮转，基于 lumberjack 自动管理日志文件
- 灵活输出，支持文件、stdout 或同时输出
- 错误字段统一，所有实现使用 `"err"` 作为错误字段名

**可用实现**：
- **Zap**：高性能日志库，适合高并发场景 - [文档](logger/zap/README.md)
- **Logrus**：社区流行，生态成熟 - [文档](logger/logrus/README.md)
- **Slog**：Go 1.21+ 官方标准库，零依赖 - [文档](logger/slog/README.md)

### Gateway（网关模块）

位于 `gateway/` 目录，提供服务网关的抽象接口和具体实现。

**核心功能**：
- 服务实例的动态注册、注销和更新
- 统一的网关抽象接口，屏蔽底层实现差异
- 支持多种网关实现和存储后端

**详细文档**：请参考 [gateway/traefik/README.md](gateway/traefik/README.md) 了解具体实现和使用方法

### Registry（服务注册中心模块）

位于 `registry/` 目录，提供服务注册与发现的抽象接口和具体实现。

**核心功能**：
- 服务实例的注册、注销和发现
- 实时服务监听(Watch)机制，自动更新本地缓存
- 多种负载均衡策略（RoundRobin、Random、WeightedRandom）
- 多维度流量隔离（环境、集群、公司、项目、染色）
- LoadBalancer 状态保持，确保轮询等策略的连续性

**可用实现**：
- **Etcd**：基于 etcd 的服务注册中心实现 - [文档](registry/etcd/README.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。
