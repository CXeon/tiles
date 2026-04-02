# Tiles

Tiles 是一个基于 Go 语言开发的微服务工具包集合，旨在为微服务项目提供常用的基础设施组件。正如其名称所示，Tiles 像瓦片一样铺设基础，让开发者能在屋顶之下自由构建丰富多彩的业务世界。

## 特性

- **单模块设计**：统一 `github.com/CXeon/tiles` 模块，一次安装即可使用所有组件
- **开箱即用**：提供微服务开发常用的基础设施组件
- **接口抽象**：核心模块均基于抽象接口设计，支持多种后端实现自由切换
- **轻量集成**：按需引用子包，不引入不需要的依赖

## 安装

```bash
go get github.com/CXeon/tiles
```

## 项目结构

```
tiles/
├── cache/                # 缓存模块（接口 + Memory / Redis 实现）
├── config/               # 配置模块（接口 + Viper / Apollo 实现）
├── context/              # 上下文模块（AppContext，微服务链路字段透传）
├── db/
│   └── gormdb/           # 数据库模块（基于 GORM，支持 MySQL / PostgreSQL / SQLite）
├── errors/               # 错误模块（结构化错误码，哨兵错误，错误链）
├── gateway/              # 网关模块（接口 + Traefik 实现）
├── logger/               # 日志模块（接口 + Zap / Logrus / Slog 实现）
└── registry/             # 服务注册中心模块（接口 + Etcd 实现）
```

## 模块说明

### Cache（缓存模块）

位于 `cache/` 目录，提供统一的缓存抽象接口，覆盖 Redis 常用数据类型操作，内存实现可作为 Redis 的开发/测试替代品。

**核心功能**：
- 统一接口覆盖 Redis 常用操作：String、Hash、List、Set、Sorted Set 及原子计数器
- 内存与 Redis 实现均满足同一接口，可通过一行代码无感知切换
- 扩展接口设计（Wrap Interface）：`RedisCache` 在基础接口之上追加 Pipeline、Pub/Sub、Scan 等 Redis 独有能力
- 错误归一：`redis.Nil`、`WRONGTYPE` 等 Redis 错误统一转换为 `cache` 包哨兵错误
- TTL 全面支持：`Set`、`Expire`、`TTL`、`SetNX` 均支持过期控制

**可用实现**：
- **Memory**：基于标准库的内存缓存，无外部依赖，适合开发、测试及 Redis 不可用时的临时替代 - [文档](cache/memory/README.md)
- **Redis**：基于 go-redis/v9，支持 Pipeline、事务 Pipeline、Pub/Sub、Scan 及原始客户端暴露 - [文档](cache/redis/README.md)

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

### Context（上下文模块）

位于 `context/` 目录，提供微服务链路中的标准化请求上下文，用于在服务内部和跨服务调用之间透传通用字段。

**核心功能**：
- 封装 `AppContext`，携带 `TraceID`、`Env`、`Cluster`、`UserID`、`Color` 等链路字段
- 支持从 HTTP Header 自动提取字段（供网关/中间件使用）
- 支持任意扩展字段（`Extra`），满足业务自定义需求
- 兼容标准 `context.Context` 接口，可直接传入所有标准库及第三方函数

### DB（数据库模块）

位于 `db/gormdb/` 目录，基于 [GORM](https://gorm.io) 封装的数据库客户端，统一连接管理与连接池配置。

**核心功能**：
- 支持 MySQL、PostgreSQL、SQLite 三种数据库驱动
- 开箱即用的连接池默认配置（最大 100 连接，30 分钟复用时长）
- 通过 `Option` 函数式选项灵活覆盖连接池、字符集、SSL 等参数
- 支持注入自定义 GORM Logger，与项目日志系统集成
- `GetDB()` 返回底层 `*gorm.DB`，`Pool()` 返回 `*sql.DB`，覆盖所有使用场景

### Errors（错误模块）

位于 `errors/` 目录，提供统一的结构化错误类型与哨兵错误定义，支持错误链追踪。

**核心功能**：
- 8 位结构化错误码：`[严重程度(1位)][服务ID(3位)][序号(4位)]`
- 哨兵错误模式，支持跨层传递和 `errors.Is` 按错误码精确匹配
- `WithErrMsg` 附加内部诊断信息（仅写日志，不暴露给用户）
- `Wrap` 包装底层错误，保留完整错误链，兼容 `errors.As`
- 不可变设计：`WithErrMsg` / `Wrap` 均返回新实例，不修改原哨兵错误

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

**可用实现**：
- **Traefik**：基于 Traefik + etcd 的动态路由实现 - [文档](gateway/traefik/README.md)

### Registry（服务注册中心模块）

位于 `registry/` 目录，提供服务注册与发现的抽象接口和具体实现。

**核心功能**：
- 服务实例的注册、注销和发现
- 实时服务监听（Watch）机制，自动更新本地缓存
- 多种负载均衡策略（RoundRobin、Random、WeightedRandom）
- 多维度流量隔离（环境、集群、公司、项目、染色）
- LoadBalancer 状态保持，确保轮询等策略的连续性

**可用实现**：
- **Etcd**：基于 etcd 的服务注册中心实现 - [文档](registry/etcd/README.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。
