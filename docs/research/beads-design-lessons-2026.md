# Beads 对 RunEngram 工程记忆系统的启发

> 调研时间：2026-07-29  
> 调研范围：`gastownhall/beads` 官方 README、文档、源码目录、CHANGELOG 与 v1.1.x Release。  
> 结论先行：RunEngram 应吸收 Beads 的“结构化图、原子领取、上下文再注入、冷热数据分层、可恢复压缩”理念，但不应复制 Beads 的 Issue Tracker、Dolt 存储和完整工作流语言。

## 1. Beads 当前是什么

Beads 把自己定义为面向 AI Agent 的分布式图式 Issue Tracker，也是 Agent 的持久化结构化记忆。核心闭环是：

```text
create → dependency graph → ready → atomic claim → close → unblock
                                      ↓
                              Dolt push / pull
```

任务不是平铺列表。Issue、依赖、发现来源、验证关系、版本替代关系和讨论线程共同构成可查询图。Agent 每次会话通过 `bd prime` 获得简短工作协议和持久记忆，再用 JSON 友好的 CLI 查询、领取和更新任务。

来源：

- [README：产品定位、核心命令和存储模式](https://github.com/gastownhall/beads/blob/main/README.md)
- [Issue 数据模型](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/issues.md)
- [Agent 工作说明](https://github.com/gastownhall/beads/blob/main/AGENT_INSTRUCTIONS.md)
- [v1.1.2 Release](https://github.com/gastownhall/beads/releases/tag/v1.1.2)

## 2. 值得吸收的设计

### 2.1 从“记忆列表”升级为“有类型的工程记忆图”

Beads 把边分成两类：

- 阻塞边：`blocks`、`parent-child`、`conditional-blocks`、`waits-for`，参与 `ready` 计算；
- 语义边：`related`、`tracks`、`discovered-from`、`caused-by`、`validates`、`supersedes`、`replies-to`，只表达来源和关系。

它还支持 `duplicates` 和 `supersedes` 链，避免旧知识和新知识同时被当成事实。写入阻塞依赖时执行环检测；同层节点可并行。

这对 RunEngram 最有价值。当前 Capsule 已保存来源任务、证据、触发条件、置信度和复用反馈，但 Capsule 之间缺少结构化关系。后续可增加最小关系集：

| 关系 | 用途 |
| --- | --- |
| `derived-from` | 经验由哪个任务、修复或人工纠正产生 |
| `validated-by` | 哪个测试、交付物或后续任务证明经验有效 |
| `applies-to` | 适用模块、版本、平台或任务类型 |
| `supersedes` | 新规则替代旧规则 |
| `conflicts-with` | 两条经验不能同时应用，需要人工裁决 |
| `caused-by` | 症状经验连接到根因经验 |

这样召回不再只有“文本相似度”，还能回答：

- 这条经验为何可信；
- 哪些任务验证过；
- 是否已被新规则替代；
- 当前版本和模块是否适用；
- 失败时应该回退到哪个来源证据。

来源：

- [依赖、Ready Queue、环检测和 Gate](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/dependencies.md)
- [非阻塞图关系](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/graph-links.md)

### 2.2 `prime`：把长期记忆转成短小、可重复注入的上下文

Beads 不要求 Agent 每次读取整个数据库。`bd prime` 输出约 1–2k token 的工作协议和持久记忆；Agent 集成在新会话开始时注入，发生上下文压缩后标记上下文失效，并在下一轮重新注入一次。

关键不是多存，而是形成稳定的“重新装载协议”：

```text
长期事实源 → 任务相关检索 → 有预算的 Context Pack
                         ↑
              新会话 / 压缩后重新生成
```

RunEngram 已有固定 Context Snapshot、项目规则和相关场景经验。下一步应补齐：

1. 每个运行保存 `context_revision` 和召回原因；
2. 新会话、恢复任务、Codex/Claude Code 压缩后重新生成 Context Pack；
3. 只追加新增或失效内容，避免整包重复污染上下文；
4. Agent 必须能报告“本次使用了哪些记忆、为何使用”。

这比把所有经验塞进系统提示词更稳，也解决长期项目中的 context rot。

来源：

- [安装与 Agent 集成：CLI、MCP、`bd prime`、压缩后刷新](https://github.com/gastownhall/beads/blob/main/docs/getting-started/installation.md)
- [CHANGELOG：SessionStart、PreCompact、PostToolUse 再注入](https://github.com/gastownhall/beads/blob/main/CHANGELOG.md)

### 2.3 原子领取、幂等恢复和明确交接

Beads 的 `--claim` 原子设置负责人和 `in_progress`，并保证同一 Agent 重复领取幂等。多 Agent 模式还提供：

- `ready --claim`：从无阻塞队列直接原子领取；
- Comments：交接说明；
- Fan-out / Fan-in：拆分并行工作后汇合；
- Merge Slot：串行化高冲突操作；
- Assignee 查询：从管理视角观察当前工作。

RunEngram 已有注册 Agent、租约、心跳、恢复和依赖 DAG，基础比 Beads 的“assignee 是普通字符串”更严格。可继续吸收：

- 标准化 Handoff Packet：结论、改动、未决问题、证据、下一步；
- 对合并、发布、修改共享规则等高冲突操作增加互斥资源；
- 并行子任务必须有 Fan-in 节点和独立验证证据；
- 从 Agent 运行记录生成交接，不依赖聊天摘要。

来源：

- [多 Agent 协调、原子领取、Fan-out/Fan-in 和 Merge Slot](https://github.com/gastownhall/beads/blob/main/docs/multi-agent/coordination.md)

### 2.4 持久任务、临时运行和“提炼后删除”

Beads 的 Molecule 是持久 Work Graph；Wisp 是临时 Work Graph。Wisp 正常参与领取和执行，但默认不进入共享同步，完成后可以：

- `squash`：提炼为持久摘要；
- `burn`：无价值时删除；
- `gc/purge`：批量清理。

更重要的理念：不是所有执行痕迹都值得成为记忆。

RunEngram 可映射为：

```text
原始运行事件/候选经验（临时）
        ↓ 人工验证或复用验证
工程记忆 Capsule（持久）
        ↓ 被替代或过期
归档 Capsule（可恢复，不召回）
```

建议保留原始证据，不直接用摘要覆盖原内容。压缩产物应是派生层，记录来源 ID、压缩版本和生成时间。删除前检查是否被活跃任务、规则或证据引用。

来源：

- [Molecules：依赖驱动的持久 Work Graph](https://github.com/gastownhall/beads/blob/main/docs/workflows/molecules.md)
- [Wisps：临时执行、Squash、Burn 和 GC](https://github.com/gastownhall/beads/blob/main/docs/workflows/wisps.md)
- [Dolt 文档：引用感知的 Prune](https://github.com/gastownhall/beads/blob/main/docs/architecture/dolt.md)
- [v1.1.0 Release：压缩前归档、支持恢复](https://github.com/gastownhall/beads/releases/tag/v1.1.0)

### 2.5 模板是可选层，不绑死 Agent 内部 Loop

Beads 用 Formula 描述可复用步骤、变量、依赖和 Gate；实例化后成为 Molecule。普通任务不必使用 Formula，简单 Epic 和依赖已经够用。

这支持 RunEngram 当前方向：适配任意 Flow，但不再造强制流程引擎。

可借鉴两点：

- Adapter 只描述节点、能力、依赖、输入输出和 Gate；
- 简单任务继续单 Agent Loop，只有跨会话、并行、人工门禁或高重建成本任务才展开 Work Graph。

不应照搬 Formula 的完整 DSL、Proto/Cook/Pour/Bond 命令体系。RunEngram 面向研发记忆闭环，复杂术语会增加团队采用成本。

来源：

- [Formulas：声明式模板与变量](https://github.com/gastownhall/beads/blob/main/docs/workflows/formulas.md)
- [Molecules：模板可选、Epic 与依赖足够覆盖多数任务](https://github.com/gastownhall/beads/blob/main/docs/workflows/molecules.md)

### 2.6 事实源、导出和备份必须分开

Beads 早期使用 SQLite + JSONL + Git 同步。当前主线已迁移为 Dolt：

- Dolt 数据库是唯一事实源；
- `.beads/issues.jsonl` 只用于查看、交换、迁移和审计导出；
- JSONL Import 是 Upsert，无法表达“缺失记录是删除还是未导出”；
- 完整恢复使用 Dolt Backup，而非 JSONL；
- Schema 新于 Binary 时旧客户端失败并给出升级路径；
- 远端迁移指定单一迁移者，其他 Clone Bootstrap，避免多版本并发迁移。

RunEngram 应吸收“唯一事实源 + 明确导出语义”，但不需要 Dolt：

- SQLite 继续作为本地/小团队事实源；
- JSON/Markdown 导出只做可读快照与备份交换，不做双向实时同步；
- 数据库备份包含版本和校验信息；
- Server 启动执行 Schema Compatibility Guard；
- 多客户端写入增加记录版本或 ETag/CAS，防止平台编辑覆盖 Agent 新写入；
- Migration 由中心服务单点执行。

来源：

- [Sync Concepts：Dolt 事实源与 JSONL 边界](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/sync-concepts.md)
- [Dolt Backend：Embedded/Server、迁移、备份和远端](https://github.com/gastownhall/beads/blob/main/docs/architecture/dolt.md)
- [v1.1.0 Release：迁移哈希、漂移检测、事务内 Stale Guard](https://github.com/gastownhall/beads/releases/tag/v1.1.0)

### 2.7 CLI 优先，MCP 只做薄适配

Beads 官方建议有 Shell 的 Agent 优先 CLI：上下文约 1–2k token，低于 MCP 工具 Schema 的 10–50k token；MCP 主要服务无 Shell 环境。当前版本也已移除旧的常驻 Daemon/JSONL 自动同步架构，转为直接 CLI + Dolt。

RunEngram 可继续保留：

- CLI：Agent 的稳定协议；
- Skill/Plugin：把正确调用方式注入 Codex、Claude Code；
- HTTP Server：Web UI、团队共享和中心事实源；
- MCP：未来薄封装现有 API，不复制业务规则。

RunEngram 不能照搬 Beads 的“无 Server”形态。Web 看板、团队共享指标和跨机器使用需要中心服务。

来源：

- [安装文档：CLI 与 MCP 的适用边界](https://github.com/gastownhall/beads/blob/main/docs/getting-started/installation.md)
- [CHANGELOG：Dolt-in-Git 替代 JSONL Sync，移除旧 Daemon 引用](https://github.com/gastownhall/beads/blob/main/CHANGELOG.md)

## 3. RunEngram 当前差距

| 维度 | RunEngram 当前能力 | 主要差距 | 建议 |
| --- | --- | --- | --- |
| 记忆模型 | 项目规则 + 场景经验；证据门禁；置信度；复用反馈 | Capsule 间无显式关系 | 增加最小 Typed Memory Graph |
| 召回 | 启动快照 + 动态相关召回 | 压缩后刷新、召回增量和原因可视化不足 | 增加 Prime/Refresh 回执 |
| 有效性 | helpful/rejected/stale，人工验证后启用 | 缺少跨版本、跨模块的冲突和替代关系 | 增加 supersedes/conflicts-with/applies-to |
| 多 Agent | 注册、领取、租约、心跳、恢复 | 交接内容仍容易散落；高冲突动作无互斥资源 | Handoff Packet + Resource Lock |
| 图执行 | Task DAG、Run Graph、节点证据 | 任务关系和知识关系分离不够清楚 | Work Graph 与 Memory Graph 分层 |
| 生命周期 | 候选、验证、可信、有争议、过期 | 临时运行痕迹和长期记忆的清理策略不完整 | 临时层 → 提炼 → 归档/清理 |
| 持久化 | 单 SQLite + HTTP Server | 导出、完整备份、Schema Skew 和并发覆盖保护不足 | Backup Manifest + CAS/ETag + Schema Guard |
| Agent 集成 | Codex 默认，Claude Code Skill；CLI JSON | 压缩后自动刷新协议不足 | SessionStart/PreCompact/PostCompact Hooks |

## 4. 不应照搬

### 4.1 不采用 Dolt

原因：

- RunEngram 当前目标是本机和小团队试点；
- 单二进制 + SQLite 已满足部署和备份；
- Dolt Embedded/Server、Remote、Migration、Bootstrap 会显著增加运维面；
- RunEngram 已有中心 HTTP 服务，不需要把数据库历史塞进 Git Remote。

未来出现跨地域离线多主写入需求，再评估同步后端。当前优先做 SQLite 上的乐观并发和可靠备份。

### 4.2 不复制通用 Issue Tracker

Beads 核心是 Issue Tracker；RunEngram 核心是“执行 → 证据 → 经验 → 下次召回 → 复用验证”。不应复制：

- 完整 Issue 类型、优先级和 Tracker Integration；
- Formula/Proto/Cook/Pour/Bond 全套 DSL；
- Federation、Cross-rig Routing；
- Message Issue 和邮件线程；
- Dolt 分支与 Remote 管理。

已有 GitHub/Jira/Linear 应继续管理组织级需求。RunEngram 只保存 Agent 执行闭环需要的最小任务镜像和工程记忆。

### 4.3 不采用“写入即全局记忆”

`bd remember` 的 K/V 内容可由 `bd prime` 自动注入，适合个人显式记事，但缺少 RunEngram 所需的证据门禁。RunEngram 应保留更严格规则：

- 未验证内容只能是候选；
- 项目规则必须人工确认；
- 场景经验必须有适用边界；
- 后续任务记录真实复用结果；
- 被拒绝、冲突或过期内容停止召回。

这是 RunEngram 相比 Beads 更应该强化的差异点。

### 4.4 不直接压缩或删除原始证据

LLM 摘要可能丢失否定条件、版本范围和失败细节。不能用“压缩后的经验”覆盖：

- 原始任务输入；
- 测试输出；
- 人工确认；
- Capsule 来源和版本链。

压缩只生成可重建的派生摘要；原始证据先归档，再按引用保护策略清理。

### 4.5 不弱化 Agent 身份

Beads Assignee 是普通字符串，没有 Agent Registry。RunEngram 需要身份、租约、运行归属和团队指标，不应退回字符串负责人模型。

## 5. 推荐路线

### P0：让记忆可解释、可刷新、可纠错

目标：用户能明确感受到“这次任务因哪些历史经验做得更快、更准”。

1. 增加 Typed Memory Links：`validated-by`、`applies-to`、`supersedes`、`conflicts-with`。
2. Context Snapshot 保存召回原因、记忆版本和关系路径。
3. 新会话、恢复和上下文压缩后自动刷新 Context Pack。
4. 平台展示“本次采用/跳过哪些经验、原因、结果”。
5. Web 编辑和 Agent 写入增加版本号/CAS，冲突时提示重新加载。
6. 旧经验被替代后保留可追溯链，立即停止自动召回。

验收：

- Agent 能解释每条记忆为何进入上下文；
- 同一规则的新旧版本不会同时生效；
- 压缩后 Agent 可恢复任务、规则和待决问题；
- 并发编辑不静默覆盖。

### P1：让多任务经验能提炼，不无限膨胀

目标：任务越多，记忆质量提高，而非上下文越来越长。

1. 增加临时候选层和引用感知归档。
2. 多条同类经验聚合成“规则建议”，人工确认后成为项目规则。
3. 摘要只作为派生 Capsule；保留来源任务、证据和恢复入口。
4. 增加标准 Handoff Packet 与 Fan-in 验证节点。
5. 增加 JSON/Markdown 只读导出、完整 SQLite Backup Manifest 和恢复演练。
6. 统计召回命中、采用、拒绝、过期、验证次数；不虚构节省时长。

验收：

- 大量相似候选不会重复进入 Context Pack；
- 每个聚合规则能追溯到所有来源证据；
- 删除或归档前能检测活跃引用；
- 新 Agent 只看 Handoff Packet 即可恢复执行。

### P2：可选团队协作和协议扩展

目标：保持本地简单，同时支持团队试点。

1. 中心服务继续作为唯一写入事实源；客户端保留本地只读缓存。
2. 增加 Schema Compatibility Guard、单点 Migration 和可验证 Backup。
3. MCP 作为现有 HTTP/CLI 的薄适配，只暴露高频命令。
4. 增加高冲突资源锁：合并、发布、项目规则修改。
5. 评估跨项目 Memory Graph 和受控共享；默认项目隔离。
6. 只有出现真实离线多主需求时，才研究同步引擎；不预先引入 Dolt/Federation。

验收：

- Codex、Claude Code 和 MCP 客户端使用同一领域规则；
- 混合插件版本能被服务端明确拒绝或降级；
- 团队协作不产生静默覆盖、重复领取和错误记忆扩散。

## 6. 最终判断

可以吸收 Beads，但重点不是“把 RunEngram 也做成 Beads”。

Beads 最值得学习的不是 UI 或命令名称，而是四个底层原则：

1. **事实和工作必须结构化成图，而非散落在 Markdown 与聊天中。**
2. **Agent 上下文必须能在新会话和压缩后确定性重建。**
3. **临时执行痕迹与长期记忆必须分层，持久化前需要提炼。**
4. **并发协作依赖原子领取、显式依赖、可追溯交接和冲突保护。**

RunEngram 应保留自己的创新核心：**经验不是记录后立即相信，而是经过证据审核、任务召回、实际复用反馈，再成长为可信项目知识。** Beads 提供图与协作骨架；RunEngram 应把“验证过的工程记忆如何让下一次 Agent 执行更准”做深。

## 官方来源清单

- [gastownhall/beads](https://github.com/gastownhall/beads)
- [README](https://github.com/gastownhall/beads/blob/main/README.md)
- [Issue Model](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/issues.md)
- [Dependencies and Gates](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/dependencies.md)
- [Graph Links](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/graph-links.md)
- [Metadata](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/metadata.md)
- [Sync Concepts](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/sync-concepts.md)
- [Dolt Backend](https://github.com/gastownhall/beads/blob/main/docs/architecture/dolt.md)
- [Agent Coordination](https://github.com/gastownhall/beads/blob/main/docs/multi-agent/coordination.md)
- [Formulas](https://github.com/gastownhall/beads/blob/main/docs/workflows/formulas.md)
- [Molecules](https://github.com/gastownhall/beads/blob/main/docs/workflows/molecules.md)
- [Wisps](https://github.com/gastownhall/beads/blob/main/docs/workflows/wisps.md)
- [CHANGELOG](https://github.com/gastownhall/beads/blob/main/CHANGELOG.md)
- [v1.1.0 Release](https://github.com/gastownhall/beads/releases/tag/v1.1.0)
- [v1.1.2 Release](https://github.com/gastownhall/beads/releases/tag/v1.1.2)
