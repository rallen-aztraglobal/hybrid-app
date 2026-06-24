import { useCallback, useEffect, useRef, useState } from 'react';
import { buildApi } from '@/lib/api';
import type { BuildStatus } from '@/lib/types';

/**
 * 构建日志流（ADR-0008）—— 轮询 GET /api/build/jobs/:id/logs?after= 增量拉日志，
 * 直到任务结束（done）。后端可换成 SSE/WS；此处用增量游标轮询，最朴素也最稳。
 *
 * 用法：const { lines, status, streaming, start, reset } = useBuildLogStream();
 *       start(jobId) 开始；切任务/重打前 reset()。
 */
export interface BuildLogStreamState {
  lines: string[];
  status: BuildStatus | null;
  /** 正在轮询中 */
  streaming: boolean;
  jobId: string | null;
  start: (jobId: string) => void;
  reset: () => void;
}

const POLL_MS = 600;

export function useBuildLogStream(): BuildLogStreamState {
  const [lines, setLines] = useState<string[]>([]);
  const [status, setStatus] = useState<BuildStatus | null>(null);
  const [jobId, setJobId] = useState<string | null>(null);
  const [streaming, setStreaming] = useState(false);

  const cursorRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 防止已卸载/已重置后旧轮询继续写状态。
  const activeJobRef = useRef<string | null>(null);

  const clearTimer = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  };

  const reset = useCallback(() => {
    clearTimer();
    activeJobRef.current = null;
    cursorRef.current = 0;
    setLines([]);
    setStatus(null);
    setJobId(null);
    setStreaming(false);
  }, []);

  const start = useCallback((id: string) => {
    clearTimer();
    activeJobRef.current = id;
    cursorRef.current = 0;
    setJobId(id);
    setLines([]);
    setStatus('queued'); // 后端入队即 queued，runner 领取后才 running（评审 W3）
    setStreaming(true);

    const tick = async () => {
      if (activeJobRef.current !== id) return; // 已被 reset / 切换
      try {
        const chunk = await buildApi.getJobLogs(id, cursorRef.current);
        if (activeJobRef.current !== id) return;
        if (chunk.lines.length) {
          setLines((prev) => [...prev, ...chunk.lines]);
          cursorRef.current = chunk.cursor;
        } else if (chunk.cursor > cursorRef.current) {
          cursorRef.current = chunk.cursor;
        }
        setStatus(chunk.status);
        if (chunk.done) {
          setStreaming(false);
          return;
        }
      } catch (err) {
        if (activeJobRef.current !== id) return;
        setLines((prev) => [...prev, `# 日志拉取失败：${(err as Error)?.message ?? err}`]);
        // 失败不立即终止，下次轮询重试。
      }
      timerRef.current = setTimeout(() => void tick(), POLL_MS);
    };
    void tick();
  }, []);

  // 卸载清理。
  useEffect(() => () => clearTimer(), []);

  return { lines, status, streaming, jobId, start, reset };
}
