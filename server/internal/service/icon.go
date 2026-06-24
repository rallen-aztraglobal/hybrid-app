package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/hybrid-app/server/internal/icon"
)

// IconResult 上传图标后的返回，含主图地址、res.zip 地址与各槽位预览 URL（九宫格展示用）。
type IconResult struct {
	MasterURL string         `json:"masterUrl"`
	ResZipURL string         `json:"resZipUrl"`
	Slots     []IconSlotView `json:"slots"`
}

// IconSlotView 单个槽位的预览信息。
type IconSlotView struct {
	Kind string `json:"kind"` // square/round/foreground
	DPI  string `json:"dpi"`
	Px   int    `json:"px"`
	Path string `json:"path"`
	URL  string `json:"url"` // 预览地址（对象存储里单文件）
}

// UploadIcon 接收裁剪后的主图，fan-out 全密度，写对象存储并更新渠道记录。
// 这是图标管线的核心编排（ADR-0005、docs/admin/03 §3）。
func (s *Service) UploadIcon(ctx context.Context, channelID uint64, r io.Reader, backgroundHex string) (*IconResult, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// 解码 + 规范化主图（校验最小尺寸、裁正方形、缩放 1024²）。
	master, err := icon.DecodeMaster(r)
	if err != nil {
		return nil, errBadRequest(err.Error())
	}

	// 保存主图原件（便于后续单槽重切/重生成）。
	masterPNG, err := encodeMasterPNG(master)
	if err != nil {
		return nil, err
	}
	masterKey := fmt.Sprintf("channels/%d/icon/master.png", ch.ID)
	masterURL, err := s.storage.Put(ctx, masterKey, bytes.NewReader(masterPNG), int64(len(masterPNG)), "image/png")
	if err != nil {
		return nil, err
	}

	// fan-out。
	opt := icon.DefaultOptions()
	if backgroundHex != "" {
		opt.BackgroundHex = backgroundHex
	}
	files, err := icon.FanOut(master, opt)
	if err != nil {
		return nil, err
	}

	// 逐文件写对象存储（单文件预览） + 收集 slot 视图。
	slotByPath := map[string]icon.Slot{}
	for _, sl := range icon.Slots() {
		slotByPath[sl.Path] = sl
	}
	views := make([]IconSlotView, 0, len(files))
	for _, f := range files {
		key := fmt.Sprintf("channels/%d/icon/res/%s", ch.ID, f.Path)
		url, err := s.storage.Put(ctx, key, bytes.NewReader(f.Data), int64(len(f.Data)), f.ContentType)
		if err != nil {
			return nil, err
		}
		if sl, ok := slotByPath[f.Path]; ok {
			views = append(views, IconSlotView{Kind: sl.Kind, DPI: sl.DPI, Px: sl.Px, Path: sl.Path, URL: url})
		}
	}

	// 若该渠道已上传过 splash，则把 splash 也并进 res.zip（保持 zip 是完整 res 套件）。
	zipFiles := files
	if splash := s.tryLoadSplash(ctx, ch.ID); splash != nil {
		zipFiles = append(zipFiles, *splash)
	}

	// 打 res.zip 入对象存储。
	zipBytes, err := icon.PackZip(zipFiles)
	if err != nil {
		return nil, err
	}
	zipKey := fmt.Sprintf("channels/%d/res.zip", ch.ID)
	zipURL, err := s.storage.Put(ctx, zipKey, bytes.NewReader(zipBytes), int64(len(zipBytes)), "application/zip")
	if err != nil {
		return nil, err
	}

	// 更新渠道记录。
	if err := s.repo.UpdateChannelFields(ctx, ch.ID, map[string]any{
		"icon_master_url": masterURL,
		"icon_set_url":    zipURL,
	}); err != nil {
		return nil, err
	}

	return &IconResult{MasterURL: masterURL, ResZipURL: zipURL, Slots: views}, nil
}

// UploadSplash 接收 splash 源图，转码为 drawable/splash_fullscreen.png 存储并更新渠道。
func (s *Service) UploadSplash(ctx context.Context, channelID uint64, r io.Reader) (string, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return "", err
	}
	gf, err := icon.SplashFile(r)
	if err != nil {
		return "", errBadRequest(err.Error())
	}
	key := fmt.Sprintf("channels/%d/icon/res/%s", ch.ID, gf.Path)
	url, err := s.storage.Put(ctx, key, bytes.NewReader(gf.Data), int64(len(gf.Data)), gf.ContentType)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateChannelFields(ctx, ch.ID, map[string]any{"splash_url": url}); err != nil {
		return "", err
	}
	return url, nil
}

// ResZip 返回某渠道 res.zip 的内容（CLI 下载用）。优先返回已生成的 zip。
func (s *Service) ResZip(ctx context.Context, channelID uint64) (io.ReadCloser, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch.IconSetURL == "" {
		return nil, errNotFound("该渠道尚未生成图标资源")
	}
	key := fmt.Sprintf("channels/%d/res.zip", ch.ID)
	rc, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, errNotFound("资源 zip 不存在")
	}
	return rc, nil
}

// tryLoadSplash 尝试读取已存的 splash 文件，加入 res.zip；不存在返回 nil。
func (s *Service) tryLoadSplash(ctx context.Context, channelID uint64) *icon.GeneratedFile {
	key := fmt.Sprintf("channels/%d/icon/res/drawable/splash_fullscreen.png", channelID)
	rc, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil
	}
	return &icon.GeneratedFile{Path: "drawable/splash_fullscreen.png", ContentType: "image/png", Data: data}
}

// encodeMasterPNG 把规范化后的主图编码为 PNG 原件。
func encodeMasterPNG(master image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, master); err != nil {
		return nil, fmt.Errorf("编码主图失败: %w", err)
	}
	return buf.Bytes(), nil
}
