# ADR-0007: 仓库布局与工具链

- **状态**：已采纳(2026-06-24)；**更新(2026-06-24)**：`go.work` 本轮移除——实测 server 与 cli 无跨模块 import，workspace 的统一版本选择反而引入 `charmbracelet/x` 传递依赖冲突；各模块独立 `go.mod` 构建均通过（含交叉编译）。待真正需要跨模块共享类型时再引入。

## 背景
新增后端(Go)、CLI(Go)、前端(React)三块，要与现有 Android 工程（仓库根的 `app/` + Gradle）共存，且各代理并行开发不能互相踩。

## 决策
单仓库（monorepo），顶层并列：

```
hybrid-app/
├── app/            现有 Android 工程（不动其构建机制，仅改 WebViewActivity 接容灾）
├── channels/       现有渠道 CSV（由 CLI 渲染，唯一数据源在后台）
├── server/         Go 后端（cmd/server + internal/...）
├── cli/            Go 打包 CLI（cmd/hybrid-pack）
├── web/            React 18 前端（Vite）
├── go.work         串联 server + cli，共享 internal 类型
├── docs/           方案(admin) + 决策(adr)
├── CLAUDE.md       会话必读
└── .claude/agents/ 多代理定义
```

各代理**只动自己的顶层目录**，互不冲突；`go.work`、根级配置由编排者维护。

## 理由
- 顶层目录路径互斥 → 多代理可并行开发、零冲突，无需 git worktree 隔离。
- `go.work` 让 server 与 cli 共享 manifest/域名结构体，定义不漂移。
- Android 与新栈解耦，互不影响构建。

## 后果
- ✅ 并行友好、职责清晰。
- ➖ 多语言仓库，CI 需分别构建（go / pnpm / gradle）；`.gitignore` 要覆盖 `server/bin`、`web/node_modules`、`web/dist`、`cli/bin` 等。

## 备选
- **多仓库**：跨仓协作与版本对齐成本高，初期不划算，被否。
- **git worktree 隔离各代理**：路径已互斥，无需引入，徒增合并复杂度，被否。
