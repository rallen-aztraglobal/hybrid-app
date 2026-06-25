package runner

import (
	"bytes"
	"context"
	"sync"
	"time"
)

// jobLogStreamer 接到 gradle 输出与阶段标记上：按整行缓冲、节流回传 AppendJobLog，
// 让前端「打包中心」终端近实时滚动构建进度（ADR-0008 日志流）。
//
// 设计：
//   - 只在完整行边界 flush；末尾不完整行留到下次 Write / Close。
//   - 节流：pending ≥ flushBytes 或距上次 flush ≥ flushEvery 才回传，避免一行一请求。
//   - 回传失败静默丢弃该批，绝不阻断/失败构建（日志是辅助，产物才是主线）。
type jobLogStreamer struct {
	ctx   context.Context
	be    Backend
	jobID int64

	mu        sync.Mutex
	partial   []byte // 尚未遇到换行的残段
	pending   []byte // 已成行、待回传
	lastFlush time.Time
}

const (
	logFlushBytes = 4096
	logFlushEvery = 500 * time.Millisecond
)

func newJobLogStreamer(ctx context.Context, be Backend, jobID int64) *jobLogStreamer {
	return &jobLogStreamer{ctx: ctx, be: be, jobID: jobID, lastFlush: time.Now()}
}

// Write 实现 io.Writer：接 build.Options.Stdout/Stderr。
func (s *jobLogStreamer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.partial = append(s.partial, p...)
	if i := bytes.LastIndexByte(s.partial, '\n'); i >= 0 {
		s.pending = append(s.pending, s.partial[:i+1]...)
		s.partial = append(s.partial[:0:0], s.partial[i+1:]...) // 保留换行后的残段
	}
	if len(s.pending) >= logFlushBytes || time.Since(s.lastFlush) >= logFlushEvery {
		s.flushLocked()
	}
	return len(p), nil
}

// line 直接推一条阶段标记（自带换行并立即 flush，让关键节点尽快出现在终端）。
func (s *jobLogStreamer) line(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, msg...)
	s.pending = append(s.pending, '\n')
	s.flushLocked()
}

// Close flush 残留（含最后不完整行）。构建结束后调用。
func (s *jobLogStreamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.partial) > 0 {
		s.pending = append(s.pending, s.partial...)
		s.partial = s.partial[:0]
	}
	s.flushLocked()
}

// flushLocked 调用方须持锁。回传失败不阻断构建。
func (s *jobLogStreamer) flushLocked() {
	if len(s.pending) == 0 {
		return
	}
	chunk := string(s.pending)
	s.pending = s.pending[:0]
	s.lastFlush = time.Now()
	_ = s.be.AppendJobLog(s.ctx, s.jobID, chunk)
}
