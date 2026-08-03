# 10 · 账号管理（Admin-only User Management，V1 单管理员 MVP）

> RBAC 收敛为两档角色（admin / user）见 `server/internal/auth/auth.go`、`server/internal/model/model.go` 顶部注释与迁移 `000010_role_collapse`。本文档只讲在此之上新增的「账号管理」能力。
>
> **V1 的产品定位**：系统里只有一个永久 admin（bootstrap 创建），账号管理只负责「admin 管理普通 user」这一件事——不支持多管理员、不支持改角色、不支持启停用。这是有意的简化：先把最小可用的账号管理跑起来，多管理员等更复杂的场景放到未来版本（见 §7 扩展点）。

## 1. 角色

只有两档，历史上的 `operator`/`viewer` 已在 `000010_role_collapse` 迁移中一次性归一为 `user`：

| 角色 | 权限 |
| --- | --- |
| `admin` | 全部权限：系统设置（商店管理）、渠道归档/删除、账号管理，以及全部日常业务操作。**V1 恒只有一个 admin。** |
| `user` | 除系统设置、渠道归档/删除、账号管理外的全部日常操作：渠道、域名、打包、推送等 |

## 2. 永久 admin

- 由 `seed.EnsureBootstrapAdmin` 在数据库无任何账号时创建（`BOOTSTRAP_ADMIN=user:password` 环境变量），角色恒为 `admin`。
- **不可通过 User Management 新建**：`POST /api/users` 的入参（`CreateUserInput`）根本没有 `role` 字段，请求体里塞 `"role":"admin"` 会被解码器直接忽略（Go 的 `encoding/json` 默认忽略未声明字段），落库角色恒为 `user`。
- **不可通过 User Management 修改**：整个 API 没有「改角色」「启停用」端点。
- **不可被重置密码 / 删除**：`POST /api/users/:id/reset-password` 与 `DELETE /api/users/:id` 在服务层都会先查目标账号的角色，若为 `admin` 直接拒绝（409），见 `server/internal/service/user.go` 的 `ResetUserPassword` / `DeleteUser`。
- **列表里标记为 protected**：`GET /api/users` 返回的 `UserView.Protected` 字段（V1 恒等于 `role === "admin"`）供前端判断「这一行不可编辑/删除」，前端据此隐藏重置密码/删除按钮（见 §6）。

## 3. Version 1 支持的能力（仅此 4 项）

- **列表**：`GET /api/users`，返回 `id / username / role / createdAt / protected`，**绝不下发密码哈希**——`service.UserView` 是专门的响应 DTO，响应形状上就不存在密码相关字段，不依赖某个 json 标签「不被误删」这种脆弱保证。
- **新建用户**：`POST /api/users`，`{ username, password }`，角色恒为 `user`。
  - `username` 去空白、非空、大小写不敏感（应用层 `LOWER()` 比较）。若该用户名当前被一个**未删除**的账号占用（含永久 admin）→ 409「用户名已存在」；若被一个**已软删除**的账号占用 → 原地复活那一行（见 §4「用户名复用」），不算冲突、不报错。
  - 密码用与登录同一套 bcrypt（`auth.HashPassword`）哈希；唯一的「既有密码规则」是 bcrypt 本身 72 字节上限，超出提前拒绝为 400。
- **重置密码**：`POST /api/users/:id/reset-password`，`{ password }`；只能作用于 `role=user` 的账号（对 admin 恒 409）。
- **删除用户**：`DELETE /api/users/:id`；只能作用于 `role=user` 的账号（对 admin 恒 409）。见 §4 「删除是软删除」。

**明确不支持**（V1 故意去掉，见 §5）：创建管理员、改角色（提升/降级）、启用/禁用账号、多管理员、「最后一个启用 admin」保护逻辑。

## 4. 用户删除：软删除，而非物理删除

`DELETE /api/users/:id` 在数据库层面是**软删除**（`AdminUser.DeletedAt gorm.DeletedAt`），不是物理 `DELETE FROM`。

**为什么不是硬删除**：`channel.created_by`、`listing_app.created_by`、`audit_log.user_id` 都以整数列引用 `admin_user.id`，且都**没有外键级联**。物理删除一个账号不会报错，但会让这些字段变成指向一个不存在账号的悬空 ID，讲不清"这条渠道/上架包/审计记录到底是谁建的"。这类破坏审计与归属追溯的代价，在一个自托管、单租户的后台里没有必要为了"物理删除"这一个技术细节而承担。

**GORM 软删除的具体效果**（`AdminUser.DeletedAt` 字段的默认行为，不需要额外代码）：

- `.Delete(&AdminUser{}, id)` 自动变成 `UPDATE admin_user SET deleted_at = now() WHERE id = ?`，行本身保留。
- 常规查询（`First` / `Find`）自动加 `WHERE deleted_at IS NULL` 过滤：
  - **登录**（`GetUserByUsername`）查不到已删除账号 → 登录失败，走与「用户名不存在」相同的错误提示，不需要单独判断。
  - **鉴权中间件**（`GetUserByID`，见 §5）查不到已删除账号 → 判定为「账号不存在」→ 401，已签发、未过期的旧 token 立即失效。
  - **列表**（`ListUsers`）自然不包含已删除账号。
- 只有显式 `.Unscoped()` 才能看到/操作已删除的行——`repo.CreateOrReactivateUser` 用它查找「同用户名的已软删除行」并原地复活。

**用户名复用**：`admin_user.username` 仍是单列全局唯一索引（本轮未改 schema/迁移），删除后同一个用户名**可以复用**，但复用的方式是**复活原有的行**，而不是插入一条新记录：

- `repo.CreateOrReactivateUser` 在一个事务里用 `Unscoped() + SELECT ... FOR UPDATE` 查找同用户名（大小写不敏感）的行：
  - 查不到 → 正常插入新行。
  - 查到但**未删除**（含永久 admin）→ 返回冲突，服务层映射为 409「用户名已存在」。
  - 查到且**已软删除** → 原地更新这一行：清空 `deleted_at`、角色强制改回 `user`、`password_hash` 替换为本次提交的新密码——**保留原 `id` 与原 `created_at`**，作为创建结果返回。
- `SELECT ... FOR UPDATE` 锁住命中行（MySQL 生效；SQLite 无行锁但同一连接事务天然串行），防止两个并发的创建请求同时判断「可以复活」而产生不一致的结果。
- **接受的 V1 权衡**：复活复用的是同一个数据库行/同一个 `id`，这意味着该用户名此前留下的历史记录（`channel.created_by`、`listing_app.created_by`、`audit_log.user_id` 等引用这个 `id` 的记录）在复用后仍然与这个"新"账号关联在一起——即使 admin 实际上是为另一个不同的人重建了这个用户名，也会共享同一份历史身份。这是刻意的简化：为了在不改 schema 的前提下支持用户名复用，把"复用用户名"等同于"延续同一个账号身份"，而不是"分配一个全新、无历史包袱的账号"。若未来需要让两者区分开（如可审计地区分"谁在何时使用过这个用户名"），需要引入新的身份/版本概念，不在 V1 范围内。
- **权限保护同样适用于复活**：复活恒把角色写为 `user`，绝不会把一个曾经存在过的账号复活成 `admin`；永久 admin 的用户名恒被判定为"未删除"（它永远不会被删除），因此既不会被复活，也不会被这条创建路径改动或替换密码。
- **已签发的旧 token 不会因复用而重新变得有效**：JWT 里带有签发时密码哈希的指纹（`auth.Claims.PwFp` / `auth.PasswordFingerprint`），复用/复活必然产生新的 `password_hash`，旧指纹对不上 → `RequireActiveAccount`（及 `POST /api/auth/refresh`）判定旧 token/旧 refresh token 失效，逼迫必须用新密码重新登录才能拿到能用的新 token。这一步不依赖任何新增数据库列，纯粹基于已有的 `password_hash` 内容计算。

## 5. 本轮从 V0（多管理员设计）移除的内容

上一版实现（commit `f6e5456`）是按「支持任意数量管理员、可互相改角色/启停用」设计的，本轮按产品要求收敛为单管理员 MVP，移除了：

- `POST /api/users` 的 `role` 入参与「角色只允许 admin/user」的运行时校验（现在从类型层面就不存在这个字段，无需校验）。
- `PUT /api/users/:id` 整个端点（改角色 / 启停用）。
- `AdminUser.Enabled` 字段与 `enabled`/`updated_at` 两列（迁移 `000011` 已重写，见下）。
- 「最后一个启用中的 admin」保护：`repo.CountEnabledAdmins`、`repo.UpdateUserRoleEnabled` 里 `SELECT ... FOR UPDATE` 事务锁 + 重新计数的并发保护逻辑、`ErrLastEnabledAdmin` 哨兵错误——V1 只有一个 admin 且不可删除/改角色，这类保护没有存在的必要。
- 自身账号保护检查（不能改自己的角色、不能禁用自己）——同样因为「改角色」「启停用」整个能力都不存在了，检查对象消失。
- `handler.RequireEnabled` 中间件（按 `enabled` 字段判断会话是否失效）→ 替换为更简单的 `handler.RequireActiveAccount`（按账号是否还能查到判断，语义等价，但不再需要一个专门的布尔字段）。
- 前端：`changeRole`、`toggleEnabled`、创建表单的角色下拉与启用勾选框、表格行的角色 Select 与启停用 Switch。

## 6. 前端

`Settings（系统设置）→ 账号管理` 区块（`web/src/pages/SettingsPage.tsx` 内 `UsersManager` 组件）：

- 列表：用户名、角色（纯文本 Admin/User，不可编辑）、创建时间、操作列。
- Admin 行：显示「受保护」徽标，**不显示**重置密码/删除按钮。
- User 行：显示「重置密码」「删除」两个操作。
- 新建用户表单：仅用户名、密码、确认密码三个字段——**没有**角色下拉、没有启用勾选框。
- 删除流程：`confirm()` 弹窗明确点名用户名，确认后调用删除；成功后列表自动刷新（TanStack Query invalidate）。
- 重置密码流程：行内展开新密码 / 确认密码两个输入框，从不预填，成功后清空并收起表单。
- 非 admin：`AppShell` 侧栏不显示"系统设置"入口；直接访问 `/settings` 路由会被重定向到 `/channels`（`App.tsx`）；对应 API 一律后端 403，不依赖前端隐藏。
- 密码字段：`type="password"`，从不回显、从不预填。

## 7. 未来多管理员支持的扩展点

如果后续产品需求变为「支持多个管理员」，改动集中在：

1. `service.CreateUserInput` 加回 `Role` 字段 + 角色合法性校验。
2. 新增 `PUT /api/users/:id` 改角色端点，并在改角色时排除"把最后一个 admin 降级"的场景（可参考本次移除前的 `repo.UpdateUserRoleEnabled` 实现思路：事务 + 重新计数，历史版本见 commit `f6e5456`）。
3. `ResetUserPassword` / `DeleteUser` 的「目标是 admin 就拒绝」规则需要改为「目标是**最后一个** admin 才拒绝」。
4. `UserView.Protected` 的判定逻辑从「role==admin」改为「是否为最后一个 admin」或其他业务规则——前端不需要跟着改，因为它只消费这个字段，不自己重新判断。
5. 是否需要「启用/禁用」这个独立于删除之外的状态，取决于届时的产品需求，不是多管理员本身必须的。
