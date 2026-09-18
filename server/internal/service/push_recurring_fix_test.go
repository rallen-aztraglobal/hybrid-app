// Package service — 周期定时推送的边界场景：列表截断、状态并发覆盖、认领事务回滚。
package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

const fixTestAppID = "com.arenaplus.ap01001"

// mustRecurringCampaign 建渠道 + 一条每天一次的周期任务（1 小时后首发）。
func mustRecurringCampaign(t *testing.T, svc *Service, ctx context.Context) *PushCampaignView {
	t.Helper()
	if _, err := svc.CreateChannel(ctx, CreateChannelInput{
		BrandCode: "ap", FlavorName: "ap01001", PalCode: "P1", AppName: "A",
	}); err != nil {
		t.Fatalf("建渠道失败: %v", err)
	}
	camp, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
		Name: "周期", Title: "T", Body: "B", TargetAppIDs: []string{fixTestAppID},
	}, "tester")
	if err != nil {
		t.Fatalf("建活动失败: %v", err)
	}
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp.ID, ScheduleCampaignInput{
		ScheduledAt: time.Now().Add(time.Hour), RepeatEveryDays: 1,
	}); err != nil {
		t.Fatalf("定时失败: %v", err)
	}
	return camp
}

// 周期任务跑久了子活动会超过列表条数上限，父任务仍须出现在列表里（否则 Console 无法停止它）。
func TestListKeepsActiveScheduleBeyondLimit(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	parent := mustRecurringCampaign(t, svc, ctx)

	// 模拟 120 条更新的活动（如周期子活动）把父任务挤出「最近 100 条」。
	for i := 0; i < 120; i++ {
		if _, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
			Name: "filler", Title: "T", Body: "B", TargetAppIDs: []string{fixTestAppID},
		}, "tester"); err != nil {
			t.Fatalf("建填充活动失败: %v", err)
		}
	}

	list, err := svc.ListCampaigns(ctx, "", auth.FullScope())
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	found := false
	for i, v := range list {
		if v.ID == parent.ID {
			found = true
		}
		if i > 0 && list[i-1].ID < v.ID {
			t.Fatalf("列表应按 id 倒序，第 %d 条 id=%d 在 id=%d 之后", i, v.ID, list[i-1].ID)
		}
	}
	if !found {
		t.Fatalf("进行中的周期任务 #%d 被挤出列表（共 %d 条）", parent.ID, len(list))
	}
	if len(list) != 101 {
		t.Errorf("应为最近 100 条 + 1 条定时任务，实际 %d", len(list))
	}
}

// 状态已被别处改掉（例如 cron 刚置 sending）时，基于旧状态的写入必须失败且不覆盖。
func TestStatusCASDoesNotOverwrite(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	camp := mustRecurringCampaign(t, svc, ctx)

	// cron 抢先把活动推进到 sending（模拟「读完状态、写之前」被插队）。
	if err := r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"status": model.CampaignSending}); err != nil {
		t.Fatal(err)
	}
	err := svc.casCampaignStatus(ctx, camp.ID, model.CampaignScheduled, map[string]any{"status": model.CampaignCancelled})
	var se *Error
	if !errors.As(err, &se) || se.Code != http.StatusConflict {
		t.Fatalf("状态已变化时应返回 409，实际 %v", err)
	}
	got, _ := r.GetCampaign(ctx, camp.ID)
	if got.Status != model.CampaignSending {
		t.Errorf("sending 不应被取消覆盖，实际 %q", got.Status)
	}

	// 周期任务刚跑完置 done，迟到的「暂停」不能把它拉回 paused。
	if err := r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"status": model.CampaignDone}); err != nil {
		t.Fatal(err)
	}
	if err := svc.casCampaignStatus(ctx, camp.ID, model.CampaignScheduled, map[string]any{"status": model.CampaignPaused}); err == nil {
		t.Error("done 之后基于 scheduled 的暂停应失败")
	}
	got, _ = r.GetCampaign(ctx, camp.ID)
	if got.Status != model.CampaignDone {
		t.Errorf("done 不应被暂停覆盖，实际 %q", got.Status)
	}

	// 取消先赢：之后的真发必须被拒，状态保持 cancelled。
	camp2, err := svc.CreateCampaign(ctx, auth.FullScope(), PushCampaignInput{
		Name: "单次", Title: "T", Body: "B", TargetAppIDs: []string{fixTestAppID},
	}, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ScheduleCampaign(ctx, auth.FullScope(), camp2.ID, ScheduleCampaignInput{ScheduledAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelCampaign(ctx, auth.FullScope(), camp2.ID); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	svc.cfg.PushEnabled = true
	if _, err := svc.SendCampaign(ctx, auth.FullScope(), camp2.ID, false); err == nil {
		t.Error("已取消的活动不应能真发")
	}
	got, _ = r.GetCampaign(ctx, camp2.ID)
	if got.Status != model.CampaignCancelled {
		t.Errorf("取消后状态应保持 cancelled，实际 %q", got.Status)
	}
}

// 子活动落库失败时，父任务的推进必须整体回滚：不耗次数、不推进时间、不留孤儿子活动，下轮可重试。
func TestRecurringClaimRollsBackWhenChildFails(t *testing.T) {
	svc, r := newTestService(t)
	ctx := context.Background()
	svc.cfg.PushEnabled = true
	camp := mustRecurringCampaign(t, svc, ctx)

	// 造一个会让子活动写入失败的父任务：目标被清空（事务里拒绝派生空子活动）。
	if err := r.ReplaceCampaignTargets(ctx, camp.ID, nil); err != nil {
		t.Fatal(err)
	}
	due := time.Now().Add(-time.Minute).UTC()
	if err := r.UpdateCampaignFields(ctx, camp.ID, map[string]any{"scheduled_at": due}); err != nil {
		t.Fatal(err)
	}

	svc.RunScheduledCampaigns(ctx)

	got, err := r.GetCampaign(ctx, camp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunCount != 0 {
		t.Errorf("子活动失败时 run_count 不应增加，实际 %d", got.RunCount)
	}
	if got.Status != model.CampaignScheduled {
		t.Errorf("父任务应保持 scheduled 以便重试，实际 %q", got.Status)
	}
	if got.ScheduledAt == nil || !got.ScheduledAt.Equal(due) {
		t.Errorf("scheduled_at 不应推进，期望 %v 实际 %v", due, got.ScheduledAt)
	}
	all, err := r.ListCampaigns(ctx, repo.CampaignFilter{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range all {
		if c.ParentID != nil && *c.ParentID == camp.ID {
			t.Errorf("回滚后不应留下子活动 #%d", c.ID)
		}
	}
}
