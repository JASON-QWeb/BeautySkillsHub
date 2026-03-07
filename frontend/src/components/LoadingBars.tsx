interface LoadingBarsProps {
    className?: string
}

function LoadingBars({ className = '' }: LoadingBarsProps) {
    return (
        <div className={`loading ${className}`.trim()} aria-label="loading" role="status">
            <span />
            <span />
            <span />
            <span />
            <span />
        </div>
    )
}

export default LoadingBars
