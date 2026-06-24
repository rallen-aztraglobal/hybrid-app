import { Component, type ErrorInfo, type ReactNode } from 'react';

/**
 * 全局错误边界（评审 W2）：拦住任一渲染异常，避免整页白屏；
 * 给出可读信息 + 重新加载入口。React 渲染期错误才会触发（事件/异步错误仍需各自 try/catch）。
 */
export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // 仅打到控制台，便于排查；不上报（单租户自托管）。
    console.error('UI 渲染异常：', error, info);
  }

  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 40, fontFamily: 'system-ui, sans-serif', color: '#0f172a' }}>
          <h2 style={{ fontSize: 18, marginBottom: 8 }}>页面出错了</h2>
          <p style={{ color: '#64748b', fontSize: 13, marginBottom: 16, maxWidth: 560 }}>
            {this.state.error.message || '渲染异常'}
          </p>
          <button
            onClick={() => {
              this.setState({ error: null });
              location.reload();
            }}
            style={{
              padding: '8px 16px',
              borderRadius: 8,
              background: '#6366f1',
              color: '#fff',
              border: 0,
              cursor: 'pointer',
              fontSize: 13,
            }}
          >
            重新加载
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
