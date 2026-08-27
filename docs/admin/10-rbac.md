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
| 角色管理 | `role:manage` | route | 角色管理页(独立侧边栏菜单):查看/增删改角色 |
| 用户管理 | `user:manage` | route | 用户管理页(独立侧边栏菜单):查看/增删改账号/重置密码 |
| 设备管理 | `page:devices` | route | 设备列表查看 |
| 设备管理 | `device:export` | button | 导出设备 CSV |

角色管理、用户管理原来挂在「系统设置」模块下、kind 是 button;现已拆成侧边栏两个独立菜单,
kind 相应改成 route(菜单可见 + 页面读接口的进入权)。**两个 code 字符串本身不变**
(`role:manage`/`user:manage`)——它们已写进存量 `role_permission` 数据,改字符串会让已有授权失效,
只挪模块、改 kind。

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
1. 建三个内置初始角色:`超级管理员`(builtin,全部权限)、`运营`(全部 route + 全部 button,减去
   `perm.SystemManageCodes()`——即 `store:manage`/`role:manage`/`user:manage` 这三个敏感管理权限)、
   `只读`(全部 route,同样减去 `perm.SystemManageCodes()`)。
   陷阱:`role:manage`/`user:manage` 的 kind 是 route,会被"全部 route"捞进去——只读若不做这个减法,
   会自动白捡角色管理/用户管理这两个页面的权限;运营同理,"全部 route"那半也必须做减法,
   不能像以前那样只在 button 那半排除;
2. 把存量账号按旧 role 字符串映射:admin→超级管理员、operator→运营、viewer→只读(仅当 role_id 为 0 时回填);
3. 兜底:回填后仍 `role_id=0` 的账号(旧 `role` 字符串不在 admin/operator/viewer 三值内的脏数据,
   比如手工改过库)统一挂「只读」并打日志,避免永久锁死(以前这类账号会被判定「角色缺失」报错、
   需要运维手动修库才能恢复,现在给个最低权限兜底,运维可再手动升权);
4. 清理 role_permission 中已不存在于 catalog 的 code(catalog 演进时防悬挂)。
SQL migration 只建表加列,不做数据搬运(与生产 AutoMigrate 行为对齐)。

## API(全部走既有 Envelope 格式)

- `GET /api/perms/catalog` → `[{module, label, perms: [{code, label, kind}]}]`(登录即可,渲染勾选树)
- `GET /api/roles` → `[{id, name, description, builtin, permCodes: [], userCount, scopeAllBrands, brandCodes: [], scopeAllChannels, channelIds: []}]`(需 `role:manage` 或 `user:manage` 任一——用户管理的角色下拉框也要读它;数据范围字段见下方「数据权限」一节)
- `POST /api/roles` `{name, description, permCodes, scopeAllBrands, brandCodes, scopeAllChannels, channelIds}`;`PUT /api/roles/:id` 同(整体替换,不是增量 patch);`DELETE /api/roles/:id`(需 `role:manage`)
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

## 数据权限:角色可见的品牌 / 渠道范围(scope)

权限点回答「能不能做这件事」,数据权限回答「能对哪些数据做」。两者正交:
一个角色可以有 `channel:edit`,但只能编辑 ArenaPlus 下的某几个小渠道包。

### 模型

```
role         + scope_all_brands   bool  (默认 true = 全部品牌,含以后新增的品牌)
             + scope_all_channels bool  (默认 true = 范围内品牌下全部渠道,含以后新增的渠道)
role_brand     role_id, brand_code    -- 仅 scope_all_brands=false 时生效,联合唯一
role_channel   role_id, channel_id    -- 仅 scope_all_channels=false 时生效,联合唯一
```

有效范围(`EffectiveScope`)的计算,后端唯一实现、与权限集一样单点提供:
1. 角色 `builtin=true`(超管)→ 全部品牌 + 全部渠道,忽略上面四个字段;
2. `scope_all_brands=true` → 全部品牌;否则 = `role_brand` 列表;
3. `scope_all_channels=true` → 上述品牌下的全部渠道;否则 = `role_channel` 里**且**所属品牌仍在允许品牌内的渠道
   (品牌范围收窄后,原来勾的渠道自动失效,不需要额外清理动作)。

**「全部」是标志位而不是枚举快照**:以后新建品牌/渠道时,`全部` 范围的角色自动覆盖到,
不需要回头改每个角色;反之「指定」范围的角色不会自动获得新数据(安全的默认)。

### 强制点(后端,必须逐个覆盖——漏一个就是越权看到别家数据)

- **列表类**:查询层注入范围过滤——`GET /channels`、`GET /brands`、`GET /build/records`、
  `GET /push/campaigns`、`GET /devices`、`GET /listings`。过滤在 repo/service 层做,
  不允许「查全量再由 handler 挑」,更不允许只在前端过滤。
- **单体类(`:id` / `:code`)**:进 handler 先 `assertChannelInScope` / `assertBrandInScope`,
  越界返回 **404**(不是 403——不泄露「存在一个你看不到的渠道」这个事实)。覆盖:
  `GET/PUT/DELETE /channels/:id`、`/channels/:id/{domains,icon,splash,res.zip,latest-apk}`、
  `GET/PUT /brands/:code/domains`。
- **打包**:`POST /build/jobs` 的 brand 必须在范围内,且 flavors **全部**在范围内(有一个越界就整单拒绝);
  `GET /build/manifest` 按范围裁剪。
- **推送**:活动的 brand 在范围内才能建/改/发;`GET /push/audience` 按范围过滤——
  否则一个只管 ap 的角色能给 bp 的用户发推送。
- **新建渠道**:`POST /channels` 要求该品牌在范围内**且** `scope_all_channels=true`;
  指定渠道范围的角色不允许新建(否则新建完自己就看不见它,是个死局),返回 403 并说明原因。
- **不受数据权限约束**:runner 机器 token(要拉全量 manifest 才能构建)、
  角色/用户管理、商店、`/api/app/*` 公开端点。

### 与最小权限约束的关系

非超管建/改角色时,除了权限集要 ⊆ 自身,**数据范围也必须 ⊆ 自身范围**:
不能把自己只管 ap 的身份,拿去建一个能管 ap+bp 的角色;`scope_all_brands`/`scope_all_channels`
这两个「全部」标志位,非超管一律不能授出(自己是「全部」时才可以)。这条约束接在既有
`assertCanManageRole` 里(与权限集子集校验同一处判断,一次校验两件事),因此
`CreateUser`/`UpdateUser`/`ResetUserPassword`/`DeleteUser` 也自动受益:不能把账号挂到一个
数据范围比自己大的角色上,不能重置/删除一个数据范围比自己大的账号。

### 实现细节(后端)

- **API 字段名(两处形状不同,别混)**:角色端点 `POST/PUT /api/roles` 的请求体、
  `GET /api/roles` 返回的每个角色对象是**平铺**的
  `{scopeAllBrands, brandCodes, scopeAllChannels, channelIds}`;
  而登录/刷新/me 响应体里是**嵌套**的 `scope: {allBrands, brands, allChannels, channelIds}`
  ——语义等价但键名不同。角色端点用 Echo `Bind`,多传的未知字段(如误按嵌套形状传的 `scope`)
  会被**静默丢弃**,落库即「零品牌零渠道」,界面上还会因为读不到平铺字段而回显成「全部品牌」,
  表现为「配了全部品牌的角色登录后什么包都看不到」。回归测试:`handler.TestRoleScopeWireContract`。
  `PUT` 与 `permCodes` 一样是整体替换语义,不是增量
  patch——即便只想改权限、不想动数据范围,也要把当前的数据范围原样传回去。
- **查询层过滤是 opt-in、默认不限**:`repo.ChannelFilter`/`BuildRecordFilter`/`ListingFilter`/
  `CampaignFilter`/`DeviceFilter` 都新增了一组 `Scope*` 字段,只有显式置
  `ScopeRestricted=true` 才生效,零值(未涉及数据权限的调用点,包括这些 filter 结构体自身
  已有的测试)行为不变,不需要为了加这一层过滤就去改一遍所有既有调用签名。
  `handler.callerScope(c)` 解析出调用者的 `auth.Scope` 后,列表类端点在查询前把它写进对应
  filter(`service.ApplyChannelScope` 等)。
- **强制点分两类实现位置**:单体 `:id`/`:code` 越界 404、`POST /channels` 的新建资格 403,
  是在 handler 层直接用 `auth.Scope` 的方法(`BrandAllowed`/`ChannelAllowed`)现算现判——
  不需要额外查库(前者查询目标本身时已经在做)、也不需要改 service 方法签名;
  而 `POST /build/jobs`、推送活动的建/改/发、`GET /push/audience` 这类本来就要在 service
  层解析 appId→渠道→品牌的场景,直接把 `scope auth.Scope` 加成 service 方法的入参,判断逻辑
  就近放在已有的查库逻辑边上(`assertAppIDsInScope`/`filterAppIDsByScope`,见
  `internal/service/scope.go`)。
- **已知缺口**:`GET /api/devices/export.csv`、`GET /api/devices/export.xlsx` 走的是
  `IssueScopedToken` 签发的一次性下载令牌(不挂用户身份,专为避免长效 access token 出现在
  下载 URL 里),这条链路上还原不出发起下载的账号是谁,因此暂未按数据范围过滤/裁剪——持有
  `device:export` 权限的账号可以导出全量。`GET /api/devices`(列表)与
  `POST /api/devices/export`(勾选 id 导出,走正常 JWT)已按范围过滤。收窄前者需要给
  scoped token 加一个用户身份字段,属于后续工作。
- **GORM 零值陷阱**:`Role.ScopeAllBrands`/`ScopeAllChannels` 为了让 AutoMigrate/迁移生成
  `DEFAULT 1`("已有角色默认全部范围")而带了 `gorm:"default:true"` 标签;这个标签会让 GORM
  在结构体 `Create` 时把值为 `false`(bool 的 Go 零值)的字段当成"没传",不但不写进 INSERT、
  还会在插入后把数据库侧的默认值(`true`)回填进传入的结构体指针——也就是说,显式想建一个
  "指定范围"(`false`)的角色,字段会被静默改写成"全部范围"(`true`),且从返回的结构体里也
  看不出来。`repo.CreateRole` 因此在调用 `Create` 前先把调用方的真实意图存进局部变量,插入后
  再用 map 更新强制纠正回来(map 更新不受这条零值省略规则影响)。这个坑不是靠读代码能发现的,
  是靠 `internal/auth/scope_test.go` 的 `TestRoleEffectiveScopeBrandNarrowingInvalidatesChannels`
  等用例实测触发的——后续如果给 `Role`(或任何角色/权限相关模型)新增 `default` 标签的 bool
  字段,务必意识到这个陷阱。

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
- 设备管理:`GET /devices` → `page:devices`;`GET /devices/export-token`、`POST /devices/export`
  → `device:export`;`GET /api/devices/export.csv` 不进 JWT 组,凭 export-token 签发的 5 分钟
  scope 短 token(query 传递)验签放行,不认普通 access token(细节见 [11-devices.md](11-devices.md));
  `POST /api/app/device/register` 是 APK 端公开上报口,与其它 `/api/app/*` 一样零鉴权。
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
- 「角色管理」「用户管理」**不再挂在设置页下面**,是侧边栏两个独立菜单(与 `role:manage`/
  `user:manage` 已改成 `kind=route` 对齐,按 `page:*` 那套路由守卫规则过滤可见性即可,
  不需要特判)。角色管理页 = 列表 + 抽屉(名称/描述 + 按模块分组的权限勾选树,route 与 button 分区,
  模块级全选;builtin 角色只读展示)。用户管理页 = 列表 + 新增(用户名/密码/角色下拉)、
  改角色、重置密码、删除;对最后一个超管的保护性报错如实展示。
  设置页保留商店管理(`store:manage`)与运行时预览等原有内容。
- mock 层(mockDb)补 roles/users/catalog,遵循既有 withFallback 规则。
