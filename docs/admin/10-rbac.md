# 10 · RBAC:角色管理 + 路由/按钮权限

> 取代 dev_all 上「admin/operator/viewer 三档硬编码」的旧方案。本文档是前后端实现的唯一契约,
> 权限 code 前后端必须逐字一致。

## 概念

- **权限点(perm)**:一个字符串 code,分两类:
  - `kind=route`(路由权限):控制一个页面/菜单是否可见、其读接口是否可调,code 形如 `page:*`;
  - `kind=button`(按钮权限):控制页面内某个操作(写接口 + 对应按钮显隐),code 形如 `<模块>:<动作>`。
- **角色(role)**:一组权限点。超级管理员角色内置(`builtin=true`),拥有全部权限,不可编辑/删除。
- **用户(admin_user)**:挂一个 `role_id`。超级管理员可对用户做新增/改角色/重置密码/删除。

## 权限点清单(catalog,后端 Go 静态定义,前端从 API 拉取渲染勾选树)

| 模块 | code | kind | 说明 |
| --- | --- | --- | --- |
| 渠道管理 | `page:channels` | route | 渠道列表/详情/res.zip/最新包下载 |
| 渠道管理 | `channel:create` | button | 新增渠道 |
| 渠道管理 | `channel:edit` | button | 编辑渠道、上传图标/启动图 |
| 渠道管理 | `channel:archive` | button | 归档(删除)渠道 |
| 域名管理 | `page:domains` | route | 品牌/渠道域名查看 |
| 域名管理 | `domain:edit` | button | 修改品牌域名、渠道域名 |
| 打包中心 | `page:pack` | route | 打包提交页(manifest/当前版本查询) |
| 打包中心 | `build:submit` | button | 提交打包任务 |
| 构建记录 | `page:builds` | route | 构建记录/日志查看 |
| 推送管理 | `page:push` | route | 推送状态/活动/受众查看(含上架包推送活动) |
| 推送管理 | `push:create` | button | 新建/编辑活动、上传图片 |
| 推送管理 | `push:send` | button | 发送/定时发送(含上架包活动发送) |
| 推送管理 | `push:config` | button | 上传 google-services.json |
| 上架包 | `page:listings` | route | 上架包列表/详情/判定流水查看 |
| 上架包 | `listing:edit` | button | 新建/编辑/删除上架包、改域名 |
| 上架包 | `listing:gate` | button | AB 面网关配置与试算 |
| 系统设置 | `page:settings` | route | 设置页(商店/运行时预览等查看) |
| 系统设置 | `store:manage` | button | 商店管理写操作 |
| 系统设置 | `user:manage` | button | 用户管理(增删改/重置密码) |
| 系统设置 | `role:manage` | button | 角色管理(增删改) |

机器权限(不进 catalog、不可分配):构建机静态 token 仅隐式持有 `build:runner`,只放行 4 条写接口
`/build/claim`、`/build/records/:id/{status,logs,artifacts}`,以及服务端打包流水线必经的只读接口——
`GET /build/manifest`、`GET /build/records`、`GET /build/records/:id`、`GET /build/records/:id/logs`、
`GET /channels/:id/res.zip`(runner claim 任务后要靠这些拉 channels.csv 渲染信息 + 各渠道 res.zip
才能渲染出 flavor 资源,不放行这几条整条服务端打包流水线会 403 断掉;渠道的其它接口——列表/详情/
写操作——不放 runner),不再是可以碰任意写接口的泛化角色。

## 数据模型(migration 000010_rbac + Go seed 双路径,生产 AutoMigrate 也要能落地)

```
role            id, name(varchar64 unique), description(varchar255), builtin(bool), created_at, updated_at
role_permission role_id, perm_code(varchar64)   -- 联合唯一 (role_id, perm_code)
admin_user      + role_id(uint64, index)        -- 旧 role 字符串列保留不再使用
```

Go 侧 `seed.EnsureRBAC`(main.go 启动时**无条件**幂等执行,不受 `DB_AUTOSEED` 开关影响——
它是纯粹的自愈型数据初始化/修复,不像建品牌/建渠道/导入 CSV 那样是「无中生有」的业务数据,
关掉 `DB_AUTOSEED` 也必须让角色表就绪,否则空角色表会导致全站鉴权 `401`、登录也拿不到角色,
且无法自愈。**这是生产唯一会跑的数据迁移路径**):
1. 建三个内置初始角色:`超级管理员`(builtin,全部权限)、`运营`(全部 route + 除 store/user/role manage 外全部 button)、`只读`(仅全部 `page:*`);
2. 把存量账号按旧 role 字符串映射:admin→超级管理员、operator→运营、viewer→只读(仅当 role_id 为 0 时回填);
3. 兜底:回填后仍 `role_id=0` 的账号(旧 `role` 字符串不在 admin/operator/viewer 三值内的脏数据,
   比如手工改过库)统一挂「只读」并打日志,避免永久锁死(以前这类账号会被判定「角色缺失」报错、
   需要运维手动修库才能恢复,现在给个最低权限兜底,运维可再手动升权);
4. 清理 role_permission 中已不存在于 catalog 的 code(catalog 演进时防悬挂)。
SQL migration 只建表加列,不做数据搬运(与生产 AutoMigrate 行为对齐)。

## API(全部走既有 Envelope 格式)

- `GET /api/perms/catalog` → `[{module, label, perms: [{code, label, kind}]}]`(登录即可,渲染勾选树)
- `GET /api/roles` → `[{id, name, description, builtin, permCodes: [], userCount}]`(需 `role:manage` 或 `user:manage` 任一——用户管理的角色下拉框也要读它)
- `POST /api/roles` `{name, description, permCodes}`;`PUT /api/roles/:id` 同;`DELETE /api/roles/:id`(需 `role:manage`)
  - builtin 角色不可改/删;删除时若有用户挂在该角色 → 409;permCodes 必须都在 catalog 内 → 否则 400;
    非超管删除角色时同样受下方「最小权限约束」——只能删权限集 ⊆ 自身的角色,即便待删角色 0 成员
- `GET /api/users` → `[{id, username, roleId, roleName, builtinRole, createdAt}]`(需 `user:manage`)
- `POST /api/users` `{username, password, roleId}`;`PUT /api/users/:id` `{roleId}`;
  `POST /api/users/:id/reset-password` `{password}`;`DELETE /api/users/:id`(均需 `user:manage`)
  - 不能删除/改角色「最后一个超级管理员」;不能删除自己;username 唯一(64 字节内,服务层校验)、
    ≤64 字节、且不可为保留名 `runner`(大小写不敏感——避免与构建机静态 token 的机器身份用户名撞名，见下方
    「机器权限」一节；机器身份的判定字段是不可伪造的 token type,不是 Username,但仍在建号时拦截以免运营侧混淆)
  - 密码最小长度 6(服务层校验,与前端一致)

### 最小权限约束(非超管账号操作角色/用户时的额外校验,超管不受限)

`role:manage`/`user:manage` 是「能否进入角色/用户管理」的门槛,不代表「能对任意角色/用户为所欲为」——
否则一个只挂 `user:manage` 的账号能把自己的角色改成超级管理员、或重置 bootstrap admin 的密码来接管全站;
一个只挂 `role:manage` 的账号能把自己所在角色的权限加到超出预期。因此非超管(即当前调用者所挂角色
`builtin=false`)调用以下接口时,额外受最小权限约束,违反一律 `403`「不能操作超出自身权限的角色/用户」；
调用者是超管(所挂角色 `builtin=true`)则不受这些约束:

- `POST /api/roles`、`PUT /api/roles/:id`:请求体 `permCodes` 必须是调用者自身权限集的子集
  (不能通过建角色/改角色给自己或他人授出自己都没有的权限)。
- `DELETE /api/roles/:id`:待删角色(已确认非 builtin)的权限集必须是调用者自身权限集的子集
  (删除本身虽不构成「提权」,但仍是「操作超出自身权限范围的角色」——不能删一个权限比自己大、
  只是恰好 0 成员的角色)。
- `POST /api/users`:目标 `roleId` 若指向 builtin 角色 → 只有超管能挂;指向非 builtin 角色时,
  也要求该角色的权限集是调用者自身权限集的子集(不能把新账号挂到一个权限比自己还大的角色上)。
- `PUT /api/users/:id`:**目标账号「改动前」与「改动后」的角色都要校验**(双向,不能只查新角色)——
  若其中任一个是 builtin → 只有超管能操作;非 builtin 的一侧也要求权限集是调用者自身权限集的子集。
  只查新角色会有漏洞:非超管可以先把一个超管账号的角色改成自己权限子集内的角色(新角色校验能通过、
  旧角色未被校验),这时目标账号权限已被拉到自己管得了的范围,再顺势重置密码接管、
  逐个把超管剪到只剩一个;双向校验堵死这条路。
- `POST /api/users/:id/reset-password`、`DELETE /api/users/:id`:若目标账号当前所挂角色是 builtin
  → 只有超管能操作;目标账号所挂角色非 builtin 时,也要求该角色的权限集是调用者自身权限集的子集
  (不能重置密码或删除一个权限比自己大的账号,防止借此接管更高权限身份)。这两个接口与
  `PUT /api/users/:id` 的「改动前」校验都要求目标角色**查得到**——`GetRoleByID` 出错(悬挂
  `role_id`/查库异常)一律直接拒绝(fail-closed),不能因为查不到就静默跳过校验、放行操作。

以上判定用调用者(`Claims.UserID` → 账号 → 角色 → 权限集)当前的真实权限集现查现算,不经过
`RequirePerm` 的 30s 缓存(角色/用户管理写操作低频,现查现算保证即时一致);
后端内部用 `auth.RBAC.ResolveFresh`/`RolePermCodes` 与鉴权中间件共用同一套「角色 → 权限集」算法,
不各写一份、避免随时间漂移出不一致的口径。
- 登录/刷新返回体增加:`{..., role: {id, name, builtin}, perms: ["page:channels", ...]}`
  (超级管理员返回完整 catalog code 列表,前端逻辑统一按 perms 判断,不特判角色名)
- `GET /api/auth/me` → 同上 role+perms(前端启动时拉新,角色被改后无需重登即可生效)

## 鉴权中间件(后端)

- `RequirePerm(codes ...string)`:any-of 语义。每个请求从 DB 加载用户 → 角色 → 权限集,
  经 30s TTL 进程内缓存(角色写操作时失效);用户已被删除 → `401`(天然实现删号即失效);
  用户存在但其 `role_id` 指向的角色已不存在(脏数据/角色被误删)→ `403` + 明确文案,
  不与「账号不存在」的 401 混为一谈(避免误导排查一个其实存在的账号)。
- 机器身份(构建机静态 token)的判定字段是 JWT `Claims.Type == TokenRunner`,**不是** `Username`——
  `Middleware` 只在校验过静态令牌匹配后才会签发这个 type,普通登录/刷新签发路径永远不会产生它。
  这条边界必须严格：`Username` 是账号可注册的输入,若靠它判定机器身份,则任何人注册一个用户名
  恰好叫 `runner` 的普通账号,其正常签发的 access token 也会被误判成机器身份、绕过 4 条机器接口
  之外的权限限制。建号时另外拦截 `runner` 为保留用户名(见上）,是纵深防御,不是身份边界本身。
- 路由映射:各模块 GET → 对应 `page:*`(build 的 GET 用 any-of `page:pack`,`page:builds`,
  且额外 any-of `build:runner`——见上方「机器权限」,构建机要读这些接口才能拉 manifest 渲染资源);
  `GET /channels/:id/res.zip` 用 any-of(`page:channels`,`build:runner`)(同理,渠道的其它接口不放 runner);
  写接口 → 对应 button code;`GET /api/brands`、`GET /api/stores` 登录即可
  (商店列表是渠道抽屉的基础数据,读不设权限、写才要 `store:manage`)。
- 跨模块基础数据读接口用 any-of 放宽,不能只按「自己所在页面」收权:
  `GET /api/channels` → any-of(`page:channels`,`page:pack`,`page:push`,`page:settings`)
  (打包中心选渠道、推送受众、设置页运行时预览都依赖渠道清单);
  `GET /api/listings` → any-of(`page:listings`,`page:push`)(推送页上架包活动面板依赖)。
- `RequireActiveAccount()`:比 `RequirePerm` 更轻量的中间件,不要求任何具体权限点,只确认账号仍
  存在。`GET /api/auth/me`、`GET /api/perms/catalog`、`GET /api/brands`、`GET /api/stores` 这四条
  「登录即可」端点不进 `RequirePerm`,天然不会触发 `Resolve` 的 401 判定——不额外挂这个中间件的话,
  被删账号的旧 access token 会在其自然过期前一直能读这四条。runner 静态 token 对这四条一律 `403`:
  它们对机器身份没有正当用途(/perms/catalog、/auth/me 对机器无意义;/brands、/stores 是人工维护
  渠道用的基础数据,构建机拉全量配置走 `GET /build/manifest`)。目标:删号后这四条也立刻 `401`。
- 旧 `RequireRole`/`roleRank` 删除。

## 前端

- `AuthUser` 增加 `role: {id, name, builtin}` 与 `perms: string[]`;authStore 暴露 `hasPerm(code)`;
  启动时调 `/api/auth/me` 刷新 perms。
- 路由守卫与侧边栏按 `page:*` 过滤;无权限路由跳转到第一个有权限的页面。
- 按钮按对应 button code 显隐(无权限直接不渲染,而非置灰)。
- 设置页新增「角色管理」(列表 + 抽屉:名称/描述 + 按模块分组的权限勾选树,route 与 button 分区,
  模块级全选;builtin 角色只读展示)与「用户管理」(列表 + 新增(用户名/密码/角色下拉)、
  改角色、重置密码、删除;对最后一个超管的保护性报错如实展示)。
- mock 层(mockDb)补 roles/users/catalog,遵循既有 withFallback 规则。
