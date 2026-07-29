# Graph Engineering 对 RunEngram 的启发

> 调研日期：2026-07-29
>
> 范围：官方 GitHub 仓库、官方文档、论文与规范；未采用营销转载或二手解读。
>
> 对象：RunEngram 当前本地 Go + SQLite + React 架构，已有 Task 状态机、
> Agent Run 事件/检查点、可编辑 Learning Note、证据门控的知识晋升。

## 结论

RunEngram 不应变成另一个 LangGraph、Temporal 或 CrewAI，也不应默认把每个
任务拆成多 Agent。

更合适的升级方向是：

> **在现有 Agent Loop 外增加一层轻量、可选、可验证的 Work Graph；把每个
> 节点交给 Codex、Claude Code 或其他 Agent 自己循环，把节点结果沉淀成
> 可恢复的执行收据、可追溯的工程产物和可验证的经验。**

这能保留 Agent 的灵活性，也能解决单 Loop 的四个实际问题：

1. 长任务发生 context rot，恢复时只能依赖一段摘要；
2. 调研、复核、测试等独立工作无法安全并行；
3. “做完了”缺少节点级验收标准；
4. 经验来自聊天或主观总结，难以证明是否真的提高下一次成功率。

推荐最终模型：

```text
Goal / Task
    │
    ▼
Graph Policy ── 简单任务 ──▶ Single Agent Loop
    │
    └── 复杂任务 ───────────▶ Work Graph Run
                                  │
                         ┌────────┴────────┐
                         ▼                 ▼
                    Agent Loop A      Agent Loop B
                         │                 │
                         └──── Evidence ───┘
                                  │
                                  ▼
                       Evaluations / Human Gate
                                  │
                                  ▼
                   Learning Candidate → Verified Memory
                                  │
                                  └────────▶ 下一次任务召回
```

Graph 解决组织与验证问题；Loop 解决节点内部的开放式推理问题。两者是递进
关系，不是替代关系。

## 1. 什么时候应该用 Graph

### 保持单 Loop

满足以下大部分条件时，不创建 Graph：

- 任务能在一次上下文内完成；
- 步骤强耦合，下一步必须依赖刚才的局部推理；
- 没有客观节点验收标准；
- 并行不会显著节省时间；
- 失败后从最近一次任务摘要恢复已经足够。

例如：小范围文案修改、局部代码重命名、单文件问题定位、简短资料整理。

### 升级为 Work Graph

出现下列任意两个信号时，建议创建 Graph：

- 预计跨多个 Agent 会话，存在 context rot；
- 存在两个以上彼此独立的调查、实现或验证分支；
- 需要独立复核，而非执行者自我确认；
- 节点输出可定义明确 Schema 或验收条件；
- 中间结果昂贵，失败后不应重做；
- 存在高风险、不可逆操作，需要人工批准；
- 相同流程会重复出现，值得固化模板。

Graph 的前提不是“任务复杂”，而是**能定义边界与评估标准**。如果不知道
怎样判断节点完成，画出 Graph 只会把含糊的 Loop 变成含糊的流程图。

## 2. 一手项目对比

| 项目 | 可借鉴机制 | 对 RunEngram 的价值 | 不应照搬 |
| --- | --- | --- | --- |
| [LangGraph](https://github.com/langchain-ai/langgraph) | 每个 super-step 保存 checkpoint；并行节点保留 pending writes；`interrupt()` 保存状态并等待输入；可 replay、fork、time travel；子图可选择 per-invocation、per-thread 或 stateless persistence | 节点级检查点、成功分支不重跑、从历史节点分叉、子任务隔离 | Python 运行时、完整 Pregel 引擎、云端 Agent Server。RunEngram 不需要执行模型内部循环 |
| [OpenAI Agents SDK](https://github.com/openai/openai-agents-python) | Agent-as-tool 与 handoff；审批中断可序列化为 `RunState` 后跨进程恢复；trace/span 覆盖 model、tool、handoff、guardrail；session 与 run 分离 | 统一 Codex/Claude/Pi 事件为 trace/span；人工中断必须带结构化请求；Agent 协作既支持“工具调用”也支持“移交所有权” | 不绑定 Python SDK、OpenAI 会话存储或托管 trace；本地记录默认不保存敏感 prompt/tool payload |
| [AutoGen / Magentic-One](https://github.com/microsoft/autogen) | `GraphFlow` 支持顺序、并行、条件与循环；团队可 `save_state` / `load_state`；Magentic-One Orchestrator 负责规划、跟踪进展、失败后重规划 | 区分执行 Agent 与 Orchestrator；保持事实/计划/进展账本；失败后重规划而非盲重试 | AutoGen 已进入 maintenance mode，官方建议新项目使用 Microsoft Agent Framework；不应以其 API 作为新基础 |
| [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) | 2026 年官方后继项目；类型兼容、可达性和边校验；BSP super-step；在边界保存 executor、pending message/request、shared state；pending request 恢复后重新发出；支持本地文件 checkpoint | 最接近 RunEngram 的未来 Work Graph：类型节点、边校验、节点并行、请求持久化；且已有 Go 方向 | 不直接引入其运行时。其 super-step 同步屏障会让快分支等待慢分支，需要按本地开发任务调整 |
| [Google ADK](https://github.com/google/adk-python) | Graph Workflow 混合 LLM、Tool、函数和人工输入节点；Session、State、Artifact、Memory 分开持久化；支持 session rewind 与结果/工具轨迹评估 | 验证 Work Graph 不必等于多 Agent；可借鉴 typed node output、事件日志、会话回退和本地评测数据集 | 不绑定 Gemini、ADK Runner 或 Google Cloud session/memory 服务；rewind 也不能自动撤销外部副作用 |
| [CrewAI](https://github.com/crewAIInc/crewAI) | Crews 负责自治协作，Flows 负责事件驱动、条件路由、状态持久化与 HITL；将自治与确定性流程组合 | 可借鉴“两层模型”：Graph 控制关键路径，节点内 Agent 保持自治 | 不复制角色扮演式 Agent、Python Flow 装饰器或固定职位。角色数量不是效率 |
| [Temporal](https://github.com/temporalio/temporal) | append-only event history + deterministic replay；Workflow 无副作用，Activity 必须幂等或不可重试；signal、timer、heartbeat、retry；不同 worker 可恢复执行 | 节点收据应 append-only；副作用节点需要 idempotency key；代码版本和运行版本必须记录 | Temporal Server、worker queue、确定性 SDK 成本过高，违背 RunEngram 单二进制 + SQLite 边界 |
| [Prefect](https://github.com/PrefectHQ/prefect) | `pause` / `suspend` 可等待 Pydantic 类型输入；恢复表单能显示字段、帮助和默认值；task future、retry、cache、map 与并发 runner | 人工中断不应只是 `blocked` 文本，应有输入 Schema、说明、候选操作与响应记录 | 不引入 Python worker、部署、调度服务；RunEngram 只需要 typed interrupt 协议 |
| [Dagster](https://github.com/dagster-io/dagster) | asset graph 将产物、依赖、物化、检查、版本和 lineage 作为一等对象；失败后可从失败 asset 重执行 | spec、调查报告、补丁、测试报告、经验应成为有 lineage 的工程资产，而非附件列表 | 不复制数据分区、backfill、资源集成和数据平台模型 |
| [DBOS](https://github.com/dbos-inc/dbos-transact-py) | 普通函数拆为 workflow/step；保存输入与 step 输出；重启时返回已完成 step 的结果；支持从指定 step fork | 与本地产品形态接近：轻量 step receipt、恢复和 fork；证明不需要庞大 DAG 服务也能获得 durability | 依赖 PostgreSQL；恢复仍要求 deterministic workflow 与 idempotent step。RunEngram 应继续用 SQLite，不引入第二数据库 |
| [Restate](https://github.com/restatedev/restate) | journal 记录 LLM call、tool step 与结果；恢复时不重复模型调用和已完成副作用；durable promise/timer；OpenTelemetry trace | 节省模型成本，避免重复发消息、重复改文件；节点事件需要区分纯计算和副作用 | Restate Server 是另一套运行平台；RunEngram 当前无需 exactly-once 分布式服务 |
| [OpenEvals](https://github.com/langchain-ai/openevals) | 评估 final output、single step 和完整 trajectory；trajectory 支持 strict、unordered、subset、superset 及 LLM-as-judge | RunEngram 可用现有规范化事件做本地 trajectory eval，比较“召回经验前后”是否减少无效工具和返工 | 不把 LLM judge 当唯一事实；编译、测试、引用覆盖等确定性验证优先 |
| [Letta Code](https://github.com/letta-ai/letta-code) | memory-first Agent；上下文与 memory block 进入 Git；可检查、直接编辑、回滚；后台 dreaming 做异步归纳 | 支持“人能看见、修改、回滚”的经验版本；任务外批量整理可降低主 Loop 负担 | 不允许 Agent 无门槛重写团队规则。RunEngram 继续坚持 candidate → evidence → promotion |
| [Mem0](https://github.com/mem0ai/mem0) | 2026 算法改为 ADD-only；语义、BM25、实体与时间多信号召回；CLI 支持 add/search/update/delete | 经验不覆盖历史；召回可先用 SQLite FTS5 + 标签/实体/时间融合，不急于引入向量库 | 通用对话记忆缺少工程证据门；不能直接替代 Learning Note 与 Capsule |

### 关键来源细节

- LangGraph 的 [Persistence](https://docs.langchain.com/oss/python/langgraph/persistence)
  文档明确：每个 super-step 产生 checkpoint；并行 step 中已经成功的 node
  write 会保留，恢复时不重跑。其
  [Interrupts](https://docs.langchain.com/oss/python/langgraph/interrupts)
  还要求中断前副作用幂等。
- LangGraph 的 [Time travel](https://docs.langchain.com/oss/python/langgraph/use-time-travel)
  支持 replay 与从历史 checkpoint fork；节点之后的 LLM/API/interrupt 会重新执行。
- OpenAI Agents SDK 的
  [Human-in-the-loop](https://openai.github.io/openai-agents-python/human_in_the_loop/)
  将审批、拒绝和 pending tool call 序列化进 `RunState`；其
  [Tracing](https://openai.github.io/openai-agents-python/tracing/)
  明确区分 trace、agent span、generation span、function span、handoff span
  与 guardrail span。
- AutoGen 官方仓库已声明
  [maintenance mode](https://github.com/microsoft/autogen)，新用户应转向
  Microsoft Agent Framework。AutoGen 仍值得研究其
  [GraphFlow](https://microsoft.github.io/autogen/dev/user-guide/agentchat-user-guide/graph-flow.html)
  与 [team state persistence](https://microsoft.github.io/autogen/dev/user-guide/agentchat-user-guide/tutorial/state.html)；
  Magentic-One 论文证明 Orchestrator 可以通过持续规划、跟踪和重规划处理开放任务
  （[论文](https://arxiv.org/abs/2411.04468)）。
- Microsoft Agent Framework 的
  [Workflow](https://learn.microsoft.com/en-us/agent-framework/workflows/workflows)
  以 BSP super-step 执行并在构建时校验 Graph；
  [Checkpoint](https://learn.microsoft.com/en-us/agent-framework/user-guide/workflows/checkpoints)
  保存 executor state、pending message/request 与 shared state；
  [HITL](https://learn.microsoft.com/en-us/agent-framework/workflows/human-in-the-loop)
  恢复时重新发出未完成请求。
- Google ADK 的
  [Graph Workflow](https://adk.dev/graphs/)
  可以在同一图中组合非确定性 Agent 与确定性函数/工具/人工节点；
  [Agent evaluation](https://adk.dev/evaluate/)
  同时评估结果与工具轨迹；其
  [Session rewind](https://adk-labs.github.io/adk-docs/sessions/session/rewind/)
  会保留被回退请求的日志，但明确不恢复外部系统副作用。
- Temporal 官方
  [架构文档](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md)
  把 durable execution 建立在 event sourcing、deterministic workflow 和
  idempotent/non-retryable activity 上。
- Prefect 的
  [Interactive workflow](https://docs.prefect.io/v3/advanced/interactive)
  用类型化 RunInput 驱动暂停、表单、校验与恢复；其
  [Task runners](https://docs.prefect.io/v3/concepts/task-runners)
  将顺序、并发与分布式执行放在统一 future 接口下。
- Dagster 官方仓库强调
  [asset lineage、observability 与 checks](https://github.com/dagster-io/dagster)；
  这类 asset-first 模型适合 RunEngram 的工程产物图，而不是任务编排内核。
- DBOS 的
  [Architecture](https://docs.dbos.dev/architecture)
  说明恢复时重放 workflow，遇到已有 step output 便直接返回；缺失的第一个
  step 才重新执行。其
  [Workflow management](https://docs.dbos.dev/typescript/tutorials/workflow-management)
  支持从选中 step fork。
- Restate 的
  [Durable Agents](https://docs.restate.dev/ai/patterns/durable-agents)
  把 LLM call 与 tool execution 写入 journal，恢复不重复已完成调用。
- OpenEvals 提供
  [trajectory match 与多轮模拟](https://github.com/langchain-ai/openevals)；
  对 RunEngram 最有价值的是“最终结果正确”之外再评估工具路径和中间步骤。
- Letta Code 的
  [Git-backed MemFS](https://github.com/letta-ai/letta-code)
  证明经验可以同时面向 Agent 和人类维护；Mem0 的
  [2026 ADD-only 与多信号召回](https://github.com/mem0ai/mem0)
  说明保留时间序列、避免覆盖历史比生成一份“最新总结”更稳。

## 3. 当前 task dependency graph 不等于 executable Work Graph

RunEngram 已有 `task_deps` DAG。它回答的是宏观排期问题：

```text
任务 B 是否要等任务 A 完成，才成为 runnable？
```

它没有 executable Work Graph 所需语义：

- 不描述任务内部节点；
- 没有 typed input/output；
- 没有条件边、fan-out/fan-in、retry、fallback；
- 没有节点 evaluator、receipt、checkpoint 或 Human Interrupt；
- 不能保留并行分支的部分成功；
- 不知道 spec、补丁、测试报告之间怎样派生和验证。

因此不能把现有依赖图换一种 UI 就称为 Graph Engineering。两层应同时保留：

```text
Task Dependency Graph：跨任务、宏观依赖、排期与 runnable
Executable Work Graph：单任务内部、节点执行、验证、恢复与回退
```

## 4. RunEngram 应形成 Work、Org、Evidence 三张图

推荐产品模型：

1. **Work Graph**：工作怎样执行；
2. **Org Graph**：谁具备能力、权限与工作区来执行/复核；
3. **Evidence Graph**：产物为何可信，经验如何由证据晋升。

Learning 不是第四套独立调度图，而是 Evidence Graph 上由执行证据派生、验证、
版本化和召回的子图。

### 4.1 Work Graph：工作怎样完成

Work Graph 是任务内部运行路径。节点不是一句 Prompt，而是一个有契约的
工作单元：

```json
{
  "id": "reproduce_bug",
  "capability": "code-investigation",
  "inputs": ["bug_report", "project_context"],
  "outputs": ["reproduction.md", "failing_test"],
  "completion": [
    "artifact:failing_test",
    "evaluation:test_fails_before_fix"
  ],
  "retry": {
    "max_attempts": 2,
    "idempotency_key": "task+node+input_fingerprint"
  },
  "human_interrupt": null,
  "budget": {
    "max_minutes": 30
  }
}
```

最小节点字段：

- 输入引用及 fingerprint；
- 输出 Artifact Schema；
- 执行 capability，而非固定 Agent 名称；
- 可判定完成条件；
- evaluator；
- retry 与 idempotency 规则；
- budget 与 stop condition；
- 是否需要人工中断。

边也不能只表示“下一步”。至少支持：

- success / failure / needs-human；
- evaluator score 或确定性 assertion；
- fan-out / fan-in；
- retry / fallback；
- loop limit。

### 4.2 Artifact & Evidence Graph：结果为何可信

当前 Markdown、链接、测试报告等是任务附件。下一步应把它们升级为有类型、
版本和 lineage 的 Artifact：

```text
requirement.md
   ├──derived-by──▶ scope.md
   ├──derived-by──▶ design.md
   └──verified-by─▶ acceptance-check

bug-report
   └──reproduced-by▶ failing-test
                         ├──fixed-by────▶ patch
                         └──passes-after▶ regression-result
```

建议 Artifact 字段：

- `type`: requirement / investigation / design / patch / test-report /
  review / research-evidence / decision；
- `uri` 或本地 storage path；
- content hash、Git commit、代码 fingerprint；
- producer run/node/agent；
- derived-from / verifies / contradicts / supersedes 边；
- evaluator result；
- freshness / invalidation 条件。

这样，“完成”不再由状态字段决定，而由 Artifact + Evaluation 满足节点契约
决定。任务状态仍供人查看，证据图负责说明为什么可信。

#### Learning 子图：下一次怎样更准确

当前 `Learning Note → Evidence → Exploration Capsule` 方向正确。Graph
Engineering 应补三部分：

1. **版本链**：修改经验生成新版本，旧版本保留 `superseded_by`，不覆盖历史；
2. **关系边**：supports / contradicts / applies-to / invalidated-by；
3. **效果边**：本次召回作用于哪个 node，减少了哪些失败、澄清或重复查找。

经验条目建议补充：

- `valid_from` / `valid_until`；
- `source_artifact_ids` 与 `source_node_attempt_id`；
- `confidence` 与 evidence count；
- `contradicts_memory_ids`；
- `applicability_predicate`；
- `last_revalidated_at`；
- `helpful_count` / `rejected_count` / `stale_count`。

召回先保持本地：

```text
SQLite FTS5(BM25)
  + labels/module/entity exact match
  + code/dependency fingerprint
  + recency/freshness
  + evidence/confidence
  → top-k verified memories
```

不必马上引入向量数据库。代码符号、模块、命令、版本号等工程信息常需要精确
匹配；纯 embedding 会削弱这些信号。

## 5. Org Graph 与 Work Graph 必须分开

图片中的“两层图”值得保留，但当前个人/小团队试点应先做 Work Graph。

### Work Graph

- 描述运行时步骤；
- 节点绑定 capability；
- 每次任务生成实例；
- 直接提高开发、文档、修 Bug、调研体验。

### Org Graph

- 描述人和 Agent 能力、可用工具、权限、管理/复核关系；
- 决定谁可以领取哪个节点；
- 后续团队并行与跨电脑运行才需要。

不要把 `researcher → writer → reviewer` 固化成组织结构。更稳的方式是：

```text
node.requires = ["primary-source-search", "document-writing"]
agent.provides = ["primary-source-search", "browser"]
human.provides = ["product-decision", "release-approval"]
```

调度按 capability、可用性、工作区和权限匹配。一个人可以拥有多个能力，
一个 Agent 也可在不同任务承担不同角色。

## 6. 四类日常任务模板

模板不是硬编码 SOP。它提供默认节点、契约与 gate；Agent 可在节点内部自由
循环，用户可增删节点。

### 6.1 需求开发

```text
需求输入
  → [范围/验收条件]
  → fan-out {
       代码影响面调查
       相似实现/历史经验召回
       风险与测试策略
    }
  → [技术方案]
  → Human Gate: 范围与方案
  → [实现]
  → fan-out { 单测/构建, 回归范围, 独立代码审查 }
  → [交付证据]
  → [经验候选]
```

体感提升：

- 新会话不用重新解释架构和范围；
- 调查与测试设计可并行；
- 未通过验收不会误进入完成；
- 以后类似需求直接召回已验证入口、约束和命令。

### 6.2 Bug 修复

```text
问题报告
  → [稳定复现]
  → fan-out { 假设 A, 假设 B, 环境排除 }
  → [最小证伪实验]
  → [根因]
  → [修复]
  → fan-out { 原用例通过, 邻近回归, 变更审查 }
  → [防复发资产]
  → [经验候选]
```

硬 gate：

- 修复前必须有可观察失败，或明确写出“无法复现”的证据；
- 修复后同一证据变绿；
- 根因与 patch 路径一致；
- 防复发至少是测试、lint、监控、Skill 或明确不适用理由之一。

### 6.3 文档写作

```text
目标与读者
  → [材料收集]
  → fan-out { 事实核查, 结构提纲, 术语/规范检查 }
  → [证据包]
  → Human Gate: 观点与范围
  → [初稿]
  → fan-out { 引用检查, 独立编辑, 可执行性检查 }
  → [定稿]
  → [可复用模板/写作经验]
```

文档节点输出不是聊天文本，而是 Markdown Artifact。每条外部事实关联来源，
每次修改保留前后版本和评审意见。

### 6.4 技术调研

```text
问题与决策标准
  → [问题拆分]
  → fan-out { 官方文档, 官方源码, 论文/规范, 反例与限制 }
  → [证据包]
  → [矛盾检查]
  → Human Gate: 缺失信息/偏好
  → [综合结论]
  → [Decision Record]
  → [可复用调研经验]
```

硬 gate：

- claim-to-source 覆盖率；
- 一手来源比例；
- 来源日期/版本；
- 支持证据与反证分开；
- 结论明确标记事实、推断、建议。

## 7. Checkpoint 需要从“摘要”升级为“双账本 + 收据”

当前 `summary + next_step` 适合恢复聊天，但不够支撑 Graph replay。

先吸收 Magentic-One 的双账本：

- **Task Ledger**：目标、范围、约束、已确认事实、验收条件、当前计划；
- **Progress Ledger**：本轮动作、新增证据、仍缺信息、阻塞、下一节点、是否停滞。

连续多轮没有新增 Evidence 时，不继续空转；触发 replan、fallback 或 Human
Interrupt。账本面向协调和恢复，不能包含 hidden reasoning。

建议每个 Node Attempt 写入结构化 Receipt：

```json
{
  "node_id": "code_impact",
  "attempt": 1,
  "input_fingerprints": ["git:abc123", "doc:sha256:..."],
  "output_artifact_ids": ["artifact_123"],
  "assertions": [
    {
      "key": "affected-modules",
      "value": ["app", "webview"],
      "evidence_ids": ["event_41", "artifact_123"]
    }
  ],
  "tool_effects": [
    {
      "kind": "file-write",
      "target": "path",
      "idempotency_key": "..."
    }
  ],
  "evaluation_ids": ["eval_9"],
  "status": "completed",
  "next_nodes": ["design"]
}
```

恢复规则：

1. Receipt 完成且输入 fingerprint 未变：复用结果；
2. Receipt 完成但输入变更：标记 stale，重跑本节点及受影响下游；
3. 并行分支部分成功：复用成功分支，只重跑失败分支；
4. 节点有外部副作用：依据 idempotency key 判定是否可以重试；
5. checkpoint schema 或 graph definition 版本不兼容：阻塞并请求人工迁移，
   不猜测恢复。

## 8. Human Interrupt 应成为一等对象

`run.blocked` 只记录文字，不足以让另一个人或下一会话准确恢复。新增：

```text
Interrupt
  id
  graph_run_id
  node_attempt_id
  kind             approval | question | choice | credentials | conflict
  prompt_markdown
  response_schema
  options[]
  default
  risk
  requested_by
  status           pending | answered | rejected | expired
  response
  responded_by
  created_at / resolved_at
```

UI 直接根据 `response_schema` 生成表单。恢复时重新展示 pending interrupt；
历史回答保留。敏感凭据只提示用户在本机配置，不进入 checkpoint 或事件正文。

## 9. 评估先于自动进化

自动学习系统不能只问“是否保存了经验”，需要回答“经验是否改善了后续运行”。

### 三层评估

| 层级 | 评估对象 | 示例 |
| --- | --- | --- |
| Artifact | 最终结果 | 编译通过、测试通过、引用有效、文档结构完整 |
| Node | 单个决策 | 是否找到正确入口、是否选择正确 Skill、是否生成稳定复现 |
| Trajectory | 完整路径 | 无效工具数、重复搜索数、人工纠正数、返工节点数、恢复次数 |

### 确定性优先

优先级：

1. 编译器、测试、lint、Schema、hash、链接状态；
2. 规则 evaluator；
3. 人工批准；
4. LLM-as-judge。

LLM judge 输出只作为 Evaluation 之一，不能直接晋升团队规则。

### Memory Lift

为每次召回生成 `memory_usage`：

```text
memory_id
node_attempt_id
reason_recalled
applied | ignored | rejected | stale
verification_result
avoided_tool_calls
avoided_failed_attempts
human_corrections_after_recall
```

推荐试点指标：

- 首次有效产物耗时；
- 从失败到恢复耗时；
- 重复搜索和失败命令数量；
- 人工澄清/纠正次数；
- review 后 reopen 率；
- 已验证经验召回后成功率；
- recall 后 stale / rejected 比例；
- 相同任务模板的 P50/P90 节点耗时。

这些是观察相关性，不应直接声称因果节省时间。P1 再用回放语料做
“有经验 vs 无经验”对照。

## 10. 建议的数据模型演进

保留现有 `tasks`、`task_deps`、`agent_runs`、`task_events` 与
`learning_notes`。新增资源：

```text
graph_definitions
  id, project_id, name, work_kind, version, definition_json, active

graph_runs
  id, task_id, graph_definition_id, definition_version,
  status, input_fingerprint, started_at, completed_at

graph_node_instances
  id, graph_run_id, node_key, state, input_json, output_json,
  lease_owner, lease_expires_at

node_attempts
  id, node_instance_id, agent_run_id, attempt_no,
  status, receipt_json, started_at, completed_at

interrupts
  id, graph_run_id, node_attempt_id, kind, request_json,
  status, response_json, requested_by, responded_by, timestamps

artifacts
  id, project_id, task_id, graph_run_id, node_attempt_id,
  type, uri, content_hash, fingerprint, metadata_json, created_at

artifact_edges
  from_artifact_id, to_artifact_id,
  relation ∈ {derived-from, verifies, contradicts, supersedes}

evaluations
  id, target_type, target_id, evaluator,
  deterministic, score, passed, details_json, created_at

memory_versions
  id, learning_note_id/capsule_id, version, content,
  valid_from, valid_until, supersedes_id, created_at
```

重要约束：

- 一个 task 同时最多一个 active `graph_run`；
- 一个 graph node 可有多个 attempt，但一次只有一个 live lease；
- 不同 runnable node 可并行领取；
- Receipt、Event、Evaluation append-only；
- 编辑 promoted memory 创建新 version，不修改旧版本；
- graph definition version 固定在 run 启动时，运行中升级必须显式迁移。

## 11. P0 / P1 / P2 路线图

### P0：让 Graph 对个人日常任务产生体感

目标：不实现通用 Graph 引擎，先证明节点契约、恢复和验收能减少重复工作。

1. **Work kind 与模板选择**
   - `feature | bug | docs | research`；
   - 创建任务时推荐 `single-loop` 或模板；
   - 用户可一键接受、精简、切回单 Loop。
2. **Node Contract + Receipt**
   - 先支持顺序节点、条件边和人工 gate；
   - checkpoint 保存结构化 receipt，而非只有摘要；
   - 输入 fingerprint 未变化时可恢复。
3. **Typed Interrupt**
   - Web/CLI 展示问题、选项、风险、响应 Schema；
   - 恢复时重新发出 pending interrupt。
4. **Artifact + Evaluation**
   - spec、调查报告、测试报告等登记为 typed artifact；
   - completion 由 evaluator 判定；
   - UI 显示“节点为何通过”。
5. **Trajectory UI**
   - 默认仍是当前个人任务看板；
   - 进入任务后显示节点时间线、产物、验证、阻塞与经验回执；
   - 不把复杂 Graph 直接暴露给首次用户。
6. **四个模板试点**
   - 各完成至少 3 个真实任务；
   - 采集首次有效产物耗时、人工纠正、返工与恢复数据。

P0 不做：

- 自动生成任意 Graph；
- 固定多 Agent 组织；
- 分布式调度；
- 向量数据库；
- 自动修改 Skill 或团队规则。

### P1：并行、复核和可测学习

1. **fan-out / fan-in 与 partial success**
   - 同一 task 一个 graph run；
   - 多个 runnable node 可被不同 Agent/电脑并行领取；
   - 成功分支 receipt 保留，失败分支单独重跑。
2. **独立 Reviewer**
   - 执行者与 reviewer 可配置 separation-of-duty；
   - review 节点只能消费冻结 Artifact，不共享隐藏推理。
3. **Artifact Lineage 与增量失效**
   - 输入 fingerprint 变化只失效受影响节点和下游；
   - 图上显示 stale 原因。
4. **Fork / Replay**
   - 从 checkpoint 创建实验分支；
   - 比较不同方案 trajectory、成本与验证结果；
   - 原运行不可变。
5. **Memory Curator**
   - 任务结束后异步扫描 Learning Note；
   - 仅提出去重、冲突、合并、失效建议；
   - 人确认后生成新 memory version。
6. **本地评估语料**
   - 将真实失败、人工纠正、成功恢复转为 regression case；
   - 对同一模板比较 recall on/off；
   - 只有重复证明有效的经验才建议升级为 Skill/test/rule。

### P2：团队 Graph 与自适应策略

1. **Org Graph**
   - 人/Agent capability、工具、工作区、权限、可用性；
   - capability-based routing；
   - 管理、执行、复核关系可配置。
2. **Adaptive Graph Composer**
   - 根据 task、历史模板和风险推荐 Graph；
   - 先展示差异，由人批准；
   - 运行中只能通过版本化 mutation 修改。
3. **策略晋升**
   - 多次验证后，Learning 可晋升为 Skill、测试、lint、模板或 workflow gate；
   - 每次晋升有 rollback 和影响范围。
4. **可选团队同步**
   - 同步 Graph、Artifact metadata、Learning 和 Evaluation；
   - 源码与私有文档仍保留本地；
   - 不破坏个人模式。
5. **可选远程 Runner**
   - 仅在团队并发需求出现后增加；
   - 本地 CLI/Agent adapter 仍使用同一事件协议。

## 12. 最重要的产品边界

### 应做

- 节点内允许 Agent 自主 Loop；
- Graph 只约束交付契约、依赖、验证和人工 gate；
- 中间产物、证据、经验全可读、可编辑、可回滚；
- 简单任务默认保持单 Loop；
- 通过实际指标证明 Graph 和经验召回有价值。

### 不应做

- 为“多 Agent”而多 Agent；
- 把角色名字当作能力模型；
- 保存 hidden reasoning 或完整私密 transcript；
- 用一个 LLM judge 决定任务完成；
- 让单次成功经验自动改写团队 Skill；
- 为获得 durable execution 引入 Temporal、PostgreSQL、Redis 或云服务；
- 把 UI 做成复杂流程设计器，迫使普通开发者先学习 Graph。

## 13. 一句话定位

如果按此路线演进，RunEngram 不只是任务看板，也不是通用 Agent 框架：

> **RunEngram 是本地优先的 Agent Engineering control plane：让开放式 Agent
> Loop 在必要时组成可恢复、可复核的 Work Graph，并把验证过的执行经验变成
> 下一次任务可用的工程记忆。**

其创新点不在“画一张 Agent 图”，而在把四件通常分散的事闭环：

```text
可恢复执行
  + 可追溯工程产物
  + 节点/轨迹评估
  + 证据门控的持续学习
```

这四者共同回答真正的效率问题：后续任务是否少走弯路、少问一次、少重做
一段，并且仍然能说明结果为什么可信。
