# 10 · 账号管理（Admin-only User Management）

> RBAC 收敛为两档角色（admin / user）见 `server/internal/auth/auth.go`、`server/internal/model/model.go` 顶部注释与迁移 `000010_role_collapse`。本文档只讲在此之上新增的「账号管理」能力：谁能管理账号、账号能被怎样操作、以及几条不可绕过的安全护栏。

## 1. 角色

只有两档，历史上的 `operator`/`viewer` 已在 `000010_role_collapse` 迁移中一次性归一为 `user`，本模块的新建/修改接口也只认这两个值：

| 角色 | 权限 |
| --- | --- |
| `admin` | 全部权限：系统设置（商店管理）、渠道归档/删除、**账号管理**，以及全部日常业务操作 |
| `user` | 除系统设置、渠道归档/删除、账号管理外的全部日常操作：渠道、域名、打包、推送等 |

## 2. 账号管理能力（仅 admin 可见 / 可调用）

- **列表**：`GET /api/users`，返回 `id / username / role / enabled / createdAt / updatedAt`，**绝不下发密码哈希**（`model.AdminUser.PasswordHash` 的 json 标签是 `"-"`，天然做不到误下发）。
- **新建账号**：`POST /api/users`，`{ username, password, role, enabled? }`；`enabled` 缺省 `true`。
  - `username` 去空白、非空、大小写不敏感全局唯一（应用层 `LOWER()` 比较，见 `repo.ExistsUsernameCI`，不依赖 sqlite/MySQL 各自不同的 collation 默认行为）。
  - `role` 只允许 `admin`/`user`，拒绝 `operator`/`viewer` 等历史值。
  - 密码用与登录同一套 bcrypt（`auth.HashPassword`）哈希；唯一的「既有密码规则」是 bcrypt 本身 72 字节上限，超出提前拒绝为 400（而不是让哈希在库内部报错）。
- **修改角色 / 启停用**：`PUT /api/users/:id`，`{ role?, enabled? }`，见 §3 安全护栏。
- **重置密码**：`POST /api/users/:id/reset-password`，`{ password }`；同一套 bcrypt 哈希，旧密码哈希被整行覆盖，此后旧密码必然登录失败。
- **不提供硬删除**：见 §4。

## 3. 安全护栏

服务端强制（`server/internal/service/user.go` + `server/internal/repo/user.go`），前端只是把这些规则做成更友好的 UX（禁用按钮/开关），**不是权限的最终来源**：

- 不能修改自己的角色、不能禁用自己的账号（`actorID == targetID` 时直接拒绝，409）。
- 不能让系统里「启用中的 admin」归零：`repo.UpdateUserRoleEnabled` 在**同一个事务**里用 `SELECT ... FOR UPDATE`（MySQL 生效）锁住目标行，重新统计「其余启用中的 admin」数，只有 >0 才放行禁用/降级——避免两个并发请求同时把最后一个 admin 弄没了。
  - 这条护栏甚至能防住"过期 JWT 角色声明"的边界情况：一个 token 里 `role=admin` 但其账号在此之后已被别人降级为 `user` 的操作者，仍会在这条护栏前被拒绝去禁用/降级真正唯一剩下的启用 admin（已用真实 HTTP 请求验证，见交付报告「人工验证」一节）。
- 角色 / `enabled` 只允许 `admin`/`user` 与布尔值，非法输入一律 400。

## 4. 删除 vs 停用

**不提供硬删除接口。** 决策依据：`audit_log.user_id`、`channel.created_by`、`listing_app.created_by` 均以整数列引用 `admin_user.id`，且都**没有外键级联**——硬删一个账号不会报错，但会让这些字段变成指向一个不存在账号的悬空 ID，讲不清"这条渠道/上架包/审计记录到底是谁建的"。这类破坏审计与归属追溯的代价，在一个自托管、单租户的后台里没有必要为了"能删除"这一个功能而承担。

因此：**禁用（`enabled=false`）是唯一支持的"移除"手段**。禁用不会删除任何历史归属数据，只是让该账号无法再登录/操作。

## 5. 禁用账号的生效时机

JWT access token 一旦签发，在自然过期前仅凭签名校验即可通过——单纯禁用数据库里的账号，并不会让这个账号已经拿到手的 token 失效（access token 有效期见 `config.AccessTokenTTL`，默认较长）。

为了让"禁用"立即生效而不是等 token 自然过期，`handler.RequireEnabled` 中间件挂在鉴权中间件之后、所有 `/api/*` 受保护路由之前，对每个请求按 `claims.UserID` 重新查一次库确认账号当前仍启用；查不到或 `enabled=false` 一律 401（见 `server/internal/handler/routes.go` 的 `api.Use(h.authMgr.Middleware(), h.RequireEnabled)`）。

构建机静态令牌（`RUNNER_TOKEN`）注入的机器身份（`claims.UserID == 0`）不对应 `admin_user` 记录，跳过此项检查——机器身份的可用性不受账号启停用影响。

**已知的相关限制（未在本轮修复，超出"禁用"这条明确需求）**：本方案只做了"启停用"的会话即时撤销，**没有**做"角色变更"的会话即时撤销——一个账号被降级后，其手里未过期的旧 token 仍带着旧的 `role=admin` 声明，直到该 token 自然过期或刷新前，仍可能通过 `RequireRole` 检查（但会在 §3 的"最后一个启用 admin"护栏前被拦下，如涉及该护栏范围内的操作）。如需堵上这个口子，需要把 `RequireEnabled` 类似的"按请求重新查库"逻辑也套用到角色上，或改造 token 撤销机制（如 token 版本号/黑名单），本轮未做，留作后续风险项。

## 6. 前端

`Settings（系统设置）→ 账号管理` 区块（`web/src/pages/SettingsPage.tsx` 内 `UsersManager` 组件），复用 `StoresManager` 的既有交互约定（新增表单 / 行内编辑 / 行级错误提示）：

- 列表：用户名、角色（下拉可改）、状态胶囊 + 启停用开关、创建时间、"重置密码"（行内展开的新密码/确认密码表单）。
- 新增账号表单：用户名、密码、确认密码、角色（admin/user）、启用勾选（默认勾选）。
- 自身账号行的角色下拉与启停用开关禁用（按 `username` 与当前登录用户比对），避免用户操作后才被后端 409 拒绝——真正的拒绝仍在后端。
- 非 admin：`AppShell` 侧栏不显示"系统设置"入口；直接访问 `/settings` 路由会被重定向到 `/channels`（`App.tsx`）；对应 API 一律后端 403，不依赖前端隐藏。
- 密码字段：`type="password"`，从不回显、从不预填。
- 不提供"删除"操作（对应 §4 的删除/停用决策）。
