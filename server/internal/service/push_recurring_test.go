// Package service — 周期定时推送（单次/每天/每 N 天）测试。
package service

import (
	"context"
	"testing"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
)

// TestNextRunAfter 表驱动校验周期步进的纯函数：正常步进、恰好到期、
// 宕机/暂停期间漏跑多次只补最后一次（不逐次补发），以及每 N 天步进。
func TestNextRunAfter(t *testing.T) {
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		prev      time.Time
		now       time.Time
		everyDays int
		want      time.Time
	}{
		{
			name:      "还没到期时步进到下一次",
			prev:      base,
			now:       base.Add(30 * time.Minute),
			everyDays: 1,
			want:      base.AddDate(0, 0, 1),
		},
		{
			name:      "恰好过期后步进到下下次",
			prev:      base,
			now:       base.AddDate(0, 0, 1).Add(time.Second),
			everyDays: 1,
			want:      base.AddDate(0, 0, 2),
		},
		{
			name:      "漏跑多次只补一次_不逐次补发",
			prev:      base,
			now:       base.AddDate(0, 0, 10), // 中间 9 次都被跳过，不补发
			everyDays: 1,
			want:      base.AddDate(0, 0, 11),
		},
		{
			name:      "每N天步进",
			prev:      base,
			now:       base.AddDate(0, 0, 8),
			everyDays: 3,
			want:      base.AddDate(0, 0, 9),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextRunAfter(tc.prev, tc.now, tc.everyDays)
			if !got.Equal(tc.want) {
				t.Errorf("nextRunAfter(%v, %v, %d) = %v，期望 %v", tc.prev, tc.now, tc.everyDays, got, tc.want)
			}
		})
	}
}

// TestScheduleCampaignValidation 校验定时接口的入参校验：过去时间、repeatEveryDays/repeatMaxRuns
// 越界、repeatEndAt 早于 scheduledAt、单次任务的周期字段被忽略清零、kind 不匹配的端点互斥。
func TestScheduleCampaignValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	camp, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
		Name: "sched", Title: "T", Body: "B", TargetAppIDs: []string{"com.arenaplus.ap01001"},
	}, "tester")
	if err != nil {
		t.Fatalf("建活动失败: %v", err)
	}

	// 过去时间应被拒绝。
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: time.Now().Add(-time.Hour),
	}); err == nil {
		t.Error("过去的 scheduledAt 应被拒绝")
	}

	future := time.Now().Add(time.Hour)

	// repeatEveryDays 越界。
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 366,
	}); err == nil {
		t.Error("repeatEveryDays=366 应被拒绝")
	}
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: -1,
	}); err == nil {
		t.Error("repeatEveryDays=-1 应被拒绝")
	}

	// repeatMaxRuns 越界。
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1, RepeatMaxRuns: 1001,
	}); err == nil {
		t.Error("repeatMaxRuns=1001 应被拒绝")
	}
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1, RepeatMaxRuns: -1,
	}); err == nil {
		t.Error("repeatMaxRuns=-1 应被拒绝")
	}

	// repeatEndAt 早于 scheduledAt。
	early := future.Add(-time.Minute)
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1, RepeatEndAt: &early,
	}); err == nil {
		t.Error("repeatEndAt 早于 scheduledAt 应被拒绝")
	}

	// 合法：单次任务即便传了 repeatEndAt/repeatMaxRuns 也应被服务端忽略清零。
	v, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 0, RepeatMaxRuns: 5, RepeatEndAt: &future,
	})
	if err != nil {
		t.Fatalf("合法单次定时应成功: %v", err)
	}
	if v.RepeatEveryDays != 0 || v.RepeatMaxRuns != 0 || v.RepeatEndAt != nil {
		t.Errorf("单次任务的周期字段应被清零，实际 %+v", v)
	}

	// 上架包活动不能走渠道 schedule 端点，反之亦然。
	l := mustCreateListing(t, svc, ctx)
	lc, err := svc.CreateListingCampaign(ctx, CreateListingCampaignInput{
		Name: "listing-sched", Title: "T", Body: "B", ListingIDs: []uint64{l.ID},
	})
	if err != nil {
		t.Fatalf("建上架包活动失败: %v", err)
	}
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), lc.ID, ScheduleCampaignInput{ScheduledAt: future}); err == nil {
		t.Error("kind=listing 的活动不应能走渠道 schedule 端点")
	}
	if _, err := svc.ScheduleListingCampaign(ctx, camp.ID, ScheduleCampaignInput{ScheduledAt: future}); err == nil {
		t.Error("kind=channel 的活动不应能走上架包 schedule 端点")
	}
	// 上架包 schedule 端点本身工作正常。
	if _, err := svc.ScheduleListingCampaign(ctx, lc.ID, ScheduleCampaignInput{ScheduledAt: future, RepeatEveryDays: 2}); err != nil {
		t.Errorf("上架包活动走上架包 schedule 端点应成功: %v", err)
	}
}

// TestPauseResumeCancelStateMachine 覆盖 pause/resume/cancel 状态机：
// 单次任务不可暂停但可取消；周期任务可暂停/恢复；恢复时按周期步进到未来；
// 周期已耗尽（达 repeatMaxRuns）时恢复应报错；paused 状态可取消。
func TestPauseResumeCancelStateMachine(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	mkCamp := func(name string) *PushCampaignView {
		c, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
			Name: name, Title: "T", Body: "B", TargetAppIDs: []string{"com.arenaplus.ap01001"},
		}, "tester")
		if err != nil {
			t.Fatalf("建活动失败: %v", err)
		}
		return c
	}
	future := time.Now().Add(time.Hour)

	// ---- 单次任务：不可暂停，可取消 ----
	single := mkCamp("single")
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), single.ID, ScheduleCampaignInput{ScheduledAt: future}); err != nil {
		t.Fatalf("定时失败: %v", err)
	}
	if _, err := svc.PauseCampaign(ctx, auth.FullScope(), single.ID); err == nil {
		t.Error("单次任务不应支持暂停")
	}
	if v, err := svc.CancelCampaign(ctx, auth.FullScope(), single.ID); err != nil || v.Status != model.CampaignCancelled {
		t.Errorf("单次任务应可取消，v=%+v err=%v", v, err)
	}
	if _, err := svc.CancelCampaign(ctx, auth.FullScope(), single.ID); err == nil {
		t.Error("已取消的活动不应能再取消")
	}

	// ---- 周期任务：暂停 → 恢复（原定时间未过去，恢复后时间不变）----
	rec := mkCamp("recurring")
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), rec.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1,
	}); err != nil {
		t.Fatalf("定时失败: %v", err)
	}
	paused, err := svc.PauseCampaign(ctx, auth.FullScope(), rec.ID)
	if err != nil || paused.Status != model.CampaignPaused {
		t.Fatalf("周期任务应可暂停，v=%+v err=%v", paused, err)
	}
	resumed, err := svc.ResumeCampaign(ctx, auth.FullScope(), rec.ID)
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if resumed.Status != model.CampaignScheduled {
		t.Errorf("恢复后状态应为 scheduled，实际 %q", resumed.Status)
	}
	if resumed.ScheduledAt == nil || !resumed.ScheduledAt.Equal(future) {
		t.Errorf("原定时间未过去时恢复不应改变 scheduledAt，期望 %v 实际 %v", future, resumed.ScheduledAt)
	}

	// ---- 暂停后把 scheduledAt 推到过去，恢复应按周期步进到未来的下一个时间点 ----
	if _, err := svc.PauseCampaign(ctx, auth.FullScope(), rec.ID); err != nil {
		t.Fatalf("暂停失败: %v", err)
	}
	past := time.Now().Add(-25 * time.Hour)
	if err := r.UpdateCampaignFields(ctx, rec.ID, map[string]any{"scheduled_at": past}); err != nil {
		t.Fatalf("直接改 scheduled_at 失败: %v", err)
	}
	resumed2, err := svc.ResumeCampaign(ctx, auth.FullScope(), rec.ID)
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if resumed2.ScheduledAt == nil || !resumed2.ScheduledAt.After(time.Now()) {
		t.Errorf("过去时间应被步进到未来，实际 %v", resumed2.ScheduledAt)
	}

	// ---- 周期已耗尽（runCount 已达 repeatMaxRuns）时恢复应报错 ----
	if err := r.UpdateCampaignFields(ctx, rec.ID, map[string]any{
		"status": model.CampaignPaused, "run_count": 3, "repeat_max_runs": 3,
	}); err != nil {
		t.Fatalf("直接改状态失败: %v", err)
	}
	if _, err := svc.ResumeCampaign(ctx, auth.FullScope(), rec.ID); err == nil {
		t.Error("已达 repeatMaxRuns 时恢复应报错")
	}

	// ---- paused 状态可以直接取消 ----
	if v, err := svc.CancelCampaign(ctx, auth.FullScope(), rec.ID); err != nil || v.Status != model.CampaignCancelled {
		t.Errorf("paused 状态应可取消，v=%+v err=%v", v, err)
	}
}

// TestSendRecurringParentBlockedButDryRunAllowed 验证周期任务本体（父任务）不可被直接真发，
// 但 dry-run 预览不受此限（前端用 dry-run 展示受众预估）。
func TestSendRecurringParentBlockedButDryRunAllowed(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	camp, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
		Name: "rec", Title: "T", Body: "B", TargetAppIDs: []string{"com.arenaplus.ap01001"},
	}, "tester")
	if err != nil {
		t.Fatalf("建活动失败: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1,
	}); err != nil {
		t.Fatalf("定时失败: %v", err)
	}
	if _, err := svc.SendCampaign(ctx, auth.FullScope(), camp.ID, true); err != nil {
		t.Errorf("周期任务本体 dry-run 应允许，实际报错: %v", err)
	}
	if _, err := svc.SendCampaign(ctx, auth.FullScope(), camp.ID, false); err == nil {
		t.Error("周期任务本体不应能被直接真发")
	}
}

// TestRunScheduledCampaignsRecurringTick 端到端验证 cron 对周期任务的一次完整生命周期：
// 到期 → CAS 认领 → 克隆子活动 → 异步真发（FCM 未配置时 Send 返回 Skipped，不算失败）→
// 父活动 runCount 前进、scheduledAt 前进；达到 repeatMaxRuns 后父活动终态 done；
// 重复 tick（含状态已非 scheduled 后）不重复触发。
func TestRunScheduledCampaignsRecurringTick(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	svc.cfg.PushEnabled = true // 允许真发路径跑通；未配 FCM 私钥时发送 no-op（Skipped）

	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	const appID = "com.arenaplus.ap01001"

	camp, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
		Name: "周期活动", Title: "T", Body: "B", TargetAppIDs: []string{appID},
	}, "tester")
	if err != nil {
		t.Fatalf("建活动失败: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: future, RepeatEveryDays: 1, RepeatMaxRuns: 2,
	}); err != nil {
		t.Fatalf("定时失败: %v", err)
	}

	childrenOf := func(parentID uint64) []PushCampaignView {
		list, err := svc.ListCampaigns(ctx, "", auth.FullScope())
		if err != nil {
			t.Fatalf("列表失败: %v", err)
		}
		var out []PushCampaignView
		for _, v := range list {
			if v.ParentID != nil && *v.ParentID == parentID {
				out = append(out, v)
			}
		}
		return out
	}
	waitChildrenDone := func(parentID uint64, want int) []PushCampaignView {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for {
			children := childrenOf(parentID)
			if len(children) >= want {
				allDone := true
				for _, ch := range children {
					if ch.Status != model.CampaignDone && ch.Status != model.CampaignFailed {
						allDone = false
					}
				}
				if allDone {
					return children
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("等待子活动完成超时，当前 children=%+v", children)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	// ---- 第一次触发 ----
	past1 := time.Now().Add(-time.Minute).UTC() // scheduled_at 约定按 UTC 存（见 repo.ListScheduledCampaigns）
	if err := r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"scheduled_at": past1}); err != nil {
		t.Fatalf("改 scheduled_at 失败: %v", err)
	}
	svc.RunScheduledCampaigns(ctx)
	children1 := waitChildrenDone(camp.ID, 1)
	if len(children1) != 1 {
		t.Fatalf("第一次 tick 后应生成 1 个子活动，实际 %d", len(children1))
	}
	if children1[0].Kind != model.CampaignKindChannel {
		t.Errorf("子活动 kind 应继承父活动为 channel，实际 %s", children1[0].Kind)
	}

	parent1, err := svc.GetCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatalf("取父活动失败: %v", err)
	}
	if parent1.RunCount != 1 {
		t.Errorf("父活动 runCount 应为 1，实际 %d", parent1.RunCount)
	}
	if parent1.Status != model.CampaignScheduled {
		t.Errorf("未达 repeatMaxRuns 时父活动应仍为 scheduled，实际 %q", parent1.Status)
	}
	if parent1.ScheduledAt == nil || !parent1.ScheduledAt.After(past1) {
		t.Errorf("父活动 scheduledAt 应已前进，past1=%v 实际=%v", past1, parent1.ScheduledAt)
	}

	// ---- 同一状态重复 tick（scheduled_at 未变化）不应重复触发 ----
	svc.RunScheduledCampaigns(ctx)
	time.Sleep(50 * time.Millisecond)
	if got := len(childrenOf(camp.ID)); got != 1 {
		t.Errorf("重复 tick 不应产生多余子活动，实际 %d 个", got)
	}

	// ---- 第二次触发：达到 repeatMaxRuns=2，父活动应终态 done ----
	past2 := time.Now().Add(-time.Minute).UTC() // scheduled_at 约定按 UTC 存（见 repo.ListScheduledCampaigns）
	if err := r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"scheduled_at": past2}); err != nil {
		t.Fatalf("改 scheduled_at 失败: %v", err)
	}
	svc.RunScheduledCampaigns(ctx)
	children2 := waitChildrenDone(camp.ID, 2)
	if len(children2) != 2 {
		t.Fatalf("第二次 tick 后应累计 2 个子活动，实际 %d", len(children2))
	}

	parent2, err := svc.GetCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatalf("取父活动失败: %v", err)
	}
	if parent2.RunCount != 2 {
		t.Errorf("父活动 runCount 应为 2，实际 %d", parent2.RunCount)
	}
	if parent2.Status != model.CampaignDone {
		t.Errorf("达到 repeatMaxRuns 后父活动应为 done，实际 %q", parent2.Status)
	}

	// ---- 父活动已 done：即便 scheduled_at 再被推到过去也不应再被 cron 捡到 ----
	past3 := time.Now().Add(-time.Minute).UTC() // scheduled_at 约定按 UTC 存（见 repo.ListScheduledCampaigns）
	_ = r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"scheduled_at": past3})
	svc.RunScheduledCampaigns(ctx)
	time.Sleep(50 * time.Millisecond)
	if got := len(childrenOf(camp.ID)); got != 2 {
		t.Errorf("done 状态的父活动不应再被 cron 触发，子活动数应仍为 2，实际 %d", got)
	}
}
