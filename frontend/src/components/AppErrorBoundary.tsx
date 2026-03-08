import React from 'react'

interface AppErrorBoundaryState {
    hasError: boolean
}

export default class AppErrorBoundary extends React.Component<React.PropsWithChildren, AppErrorBoundaryState> {
    state: AppErrorBoundaryState = {
        hasError: false,
    }

    static getDerivedStateFromError(): AppErrorBoundaryState {
        return { hasError: true }
    }

    componentDidCatch(error: Error, info: React.ErrorInfo) {
        console.error('Application render failed', error, info)
    }

    private handleReload = () => {
        window.location.reload()
    }

    private handleGoHome = () => {
        window.location.assign('/resource/all')
    }

    render() {
        if (!this.state.hasError) {
            return this.props.children
        }

        return (
            <div className="empty-state" style={{ minHeight: '100vh', padding: '64px 24px' }}>
                <div className="icon" style={{ fontSize: '3rem' }}>⚠</div>
                <h2 style={{ marginBottom: 8 }}>页面渲染失败</h2>
                <p style={{ color: 'var(--text-secondary)', marginBottom: 24 }}>
                    应用已拦截本次错误，你可以返回首页或重新加载页面。
                </p>
                <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', justifyContent: 'center' }}>
                    <button className="btn btn-secondary" type="button" onClick={this.handleGoHome}>
                        返回首页
                    </button>
                    <button className="btn btn-primary" type="button" onClick={this.handleReload}>
                        重新加载
                    </button>
                </div>
            </div>
        )
    }
}
