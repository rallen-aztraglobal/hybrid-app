// Command server 是渠道中台后端入口（Echo 单静态二进制）。
//
// 用法：
//
//	go run ./cmd/server                 启动 HTTP 服务
//	go run ./cmd/server import          把 ../channels/*.csv 导入库（清洗脏数据）
//
// 本机零配置即可启动（sqlite + 本地磁盘对象存储）；生产用环境变量切 MySQL + MinIO。
//
//	@title        渠道中台 API
//	@version      1.0
//	@description  hybrid-app 多渠道打包后台：渠道 CRUD / 图标管线 / 域名配置 / 运行时下发。
//	@BasePath     /
//	@securityDefinitions.apikey  BearerAuth
//	@in                          header
//	@name                        Authorization
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/handler"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/seed"
	"github.com/hybrid-app/server/internal/service"
	"github.com/hybrid-app/server/internal/storage"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置校验失败: %v", err)
	}

	app, err := build(cfg)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 子命令：import 导入 CSV 后退出。
	if len(os.Args) > 1 && os.Args[1] == "import" {
		runImport(app)
		return
	}

	runServer(cfg, app)
}

// application 聚合启动后的依赖。
type application struct {
	cfg     *config.Config
	repo    *repo.Repo
	svc     *service.Service
	handler *handler.Handler
	storage storage.Storage
}

// build 装配全部依赖。
func build(cfg *config.Config) (*application, error) {
	// DB。
	db, err := repo.Open(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AutoMigrate {
		if err := repo.AutoMigrate(db); err != nil {
			return nil, err
		}
	}
	r := repo.New(db)

	// 对象存储。
	st, err := buildStorage(cfg)
	if err != nil {
		return nil, err
	}

	// seed。
	ctx := context.Background()
	if cfg.AutoSeed {
		if err := seed.EnsureBrands(ctx, db); err != nil {
			return nil, err
		}
		if err := seed.EnsureBootstrapAdmin(ctx, r, cfg.BootstrapAdmin); err != nil {
			return nil, err
		}
		// 首次部署：渠道表为空时，自动从 SEED_DIR 导入 channels/*.csv 并初始化图标/启动页资产，
		// 让初次 docker compose up 即得「渠道 + 图标 + 启动页」完整数据（需求 2b）。
		// 已有渠道则不动（升级/重启幂等，不重复导入）。
		if err := autoSeedChannelsAndAssets(ctx, cfg, r, st); err != nil {
			return nil, err
		}
	}

	authMgr := auth.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authMgr.RunnerToken = cfg.RunnerToken // 构建机长期静态令牌（ADR-0008，评审 S1）
	svc := service.New(cfg, r, st)
	h := handler.New(cfg, svc, authMgr, r)

	return &application{cfg: cfg, repo: r, svc: svc, handler: h, storage: st}, nil
}

func buildStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.StorageKind {
	case "minio":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return storage.NewMinIO(ctx, storage.MinIOOptions{
			Endpoint:  cfg.MinIOEndpoint,
			AccessKey: cfg.MinIOAccessKey,
			SecretKey: cfg.MinIOSecretKey,
			Bucket:    cfg.MinIOBucket,
			UseSSL:    cfg.MinIOUseSSL,
			PublicURL: cfg.StoragePublicURL,
		})
	default:
		return storage.NewLocal(cfg.StorageLocalDir, cfg.StoragePublicURL)
	}
}

func runServer(cfg *config.Config, app *application) {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpx.HTTPErrorHandler

	app.handler.Register(e)

	// 本地磁盘对象存储：把 /static 映射到磁盘根目录，让 PublicURL 可直接访问。
	if local, ok := app.storage.(*storage.Local); ok {
		e.Static("/static", local.Root())
	}

	// 域名巡检 cron（ADR-0003，进程内）。
	var c *cron.Cron
	if cfg.DomainProbeEnable || cfg.PushCronEnable {
		c = cron.New()
		if cfg.DomainProbeEnable {
			_, _ = c.AddFunc("@every 5m", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				app.svc.RunDomainHealthScan(ctx)
			})
			log.Printf("[cron] 域名巡检已启用（每 5 分钟）")
		}
		// 推送定时任务（ADR-0012）：每分钟扫到期的 scheduled 活动并触发发送。
		if cfg.PushCronEnable {
			_, _ = c.AddFunc("@every 1m", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
				defer cancel()
				app.svc.RunScheduledCampaigns(ctx)
			})
			log.Printf("[cron] 推送定时任务已启用（每 1 分钟）")
		}
		c.Start()
	}

	// 启动 HTTP。
	go func() {
		log.Printf("渠道中台后端启动: addr=%s db=%s storage=%s", cfg.Addr, cfg.DBDriver, app.storage.Kind())
		if err := e.Start(cfg.Addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常退出: %v", err)
		}
	}()

	// 优雅关闭。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("收到退出信号，开始优雅关闭…")
	if c != nil {
		c.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("优雅关闭失败: %v", err)
	}
}

// runImport 把 channels/*.csv 导入库（顺带清洗脏数据），并默认顺带做资产初始化，
// 保证「首部署一条命令搞定渠道 + 图标 + 启动页」（需求 2a）。
//
// 用法：
//
//	server import            导入 CSV + 资产初始化（默认）
//	server import --assets   同上（显式）
//	server import --no-assets 仅导入 CSV，不做资产初始化
//
// CSV 与 res 源目录由 SEED_DIR（默认 /app/seed）推导，兼容相对路径（兼容旧行为）。
func runImport(app *application) {
	ctx := context.Background()
	withAssets := true
	for _, a := range os.Args[2:] {
		switch a {
		case "--assets":
			withAssets = true
		case "--no-assets":
			withAssets = false
		}
	}

	csvDir := seed.ResolveSeedCSVDir(app.cfg.SeedDir)
	if csvDir == "" {
		log.Fatalf("找不到 channels CSV（SEED_DIR=%q 及相对路径均无 ap/bp/gp.csv）", app.cfg.SeedDir)
	}
	reports, err := seed.ImportAllFromDir(ctx, app.repo, csvDir)
	if err != nil {
		log.Fatalf("导入失败: %v", err)
	}
	total, totalSkipped, totalCorrected := 0, 0, 0
	for _, rep := range reports {
		total += rep.Inserted
		totalSkipped += len(rep.Skipped)
		totalCorrected += len(rep.Corrected)
		log.Printf("[import] %s: 新增 %d，修正 %d，跳过 %d", rep.Brand, rep.Inserted, len(rep.Corrected), len(rep.Skipped))
		for _, s := range rep.Corrected {
			log.Printf("[import]   ~ %s", s)
		}
		for _, s := range rep.Skipped {
			log.Printf("[import]   - %s", s)
		}
	}
	log.Printf("[import] 完成：共新增 %d 渠道，按派生值修正 %d 条，跳过 %d 条（来源 %s）", total, totalCorrected, totalSkipped, csvDir)

	if withAssets {
		runAssetInit(ctx, app)
	}
}

// runAssetInit 执行一次资产初始化（图标/启动页/res.zip）并打印报告。
// 找不到 res 源目录时仅警告、不致命（CSV 已导入，资产可后续补）。
func runAssetInit(ctx context.Context, app *application) {
	resRoot := seed.ResolveSeedResRoot(app.cfg.SeedDir)
	if resRoot == "" {
		log.Printf("[assets] 警告：SEED_DIR=%q 下未找到渠道 res 源目录，跳过资产初始化", app.cfg.SeedDir)
		return
	}
	rep, err := seed.InitChannelAssets(ctx, app.repo, app.storage, resRoot)
	if err != nil {
		log.Fatalf("资产初始化失败: %v", err)
	}
	log.Printf("[assets] 完成（源 %s，storage=%s）：%s", resRoot, app.storage.Kind(), rep.Summary())
	for _, n := range rep.Notes {
		log.Printf("[assets]   - %s", n)
	}
}

// autoSeedChannelsAndAssets 在 DB_AUTOSEED=true 且渠道表为空时，自动导入 CSV + 初始化资产。
// 让首次 docker compose up 即得完整数据 + 图（需求 2b）。已有渠道则什么都不做（幂等）。
func autoSeedChannelsAndAssets(ctx context.Context, cfg *config.Config, r *repo.Repo, st storage.Storage) error {
	n, err := r.CountChannels(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil // 已有渠道：升级/重启，不重复 seed。
	}
	csvDir := seed.ResolveSeedCSVDir(cfg.SeedDir)
	if csvDir == "" {
		log.Printf("[seed] 渠道表为空但 SEED_DIR=%q 下无 channels CSV，跳过自动导入", cfg.SeedDir)
		return nil
	}
	reports, err := seed.ImportAllFromDir(ctx, r, csvDir)
	if err != nil {
		return err
	}
	total := 0
	for _, rep := range reports {
		total += rep.Inserted
	}
	log.Printf("[seed] 首次部署自动导入渠道：共新增 %d（来源 %s）", total, csvDir)

	resRoot := seed.ResolveSeedResRoot(cfg.SeedDir)
	if resRoot == "" {
		log.Printf("[seed] 警告：SEED_DIR=%q 下未找到渠道 res 源目录，跳过资产初始化（卡片将显示占位符）", cfg.SeedDir)
		return nil
	}
	rep, err := seed.InitChannelAssets(ctx, r, st, resRoot)
	if err != nil {
		return err
	}
	log.Printf("[seed] 首次部署资产初始化完成（源 %s，storage=%s）：%s", resRoot, st.Kind(), rep.Summary())
	return nil
}
