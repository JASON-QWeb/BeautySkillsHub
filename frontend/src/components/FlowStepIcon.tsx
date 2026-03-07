type FlowStepKind = 'upload' | 'ai' | 'human'

interface FlowStepIconProps {
    kind: FlowStepKind
}

function FlowStepIcon({ kind }: FlowStepIconProps) {
    if (kind === 'upload') {
        return (
            <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M12 15V4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
                <path d="M8.5 7.5 12 4l3.5 3.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                <rect x="5" y="15" width="14" height="5" rx="1.6" stroke="currentColor" strokeWidth="1.8" />
            </svg>
        )
    }

    if (kind === 'ai') {
        return (
            <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M12 3.8 5.8 6.4v5.9c0 4.2 2.7 7.9 6.2 9.9 3.5-2 6.2-5.7 6.2-9.9V6.4L12 3.8Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
                <path d="m9.3 12.1 1.8 1.8 3.6-3.8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
        )
    }

    return (
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M8.2 11.4a3 3 0 1 1 0-6 3 3 0 0 1 0 6Z" stroke="currentColor" strokeWidth="1.8" />
            <path d="M15.9 10.2a2.4 2.4 0 1 0 0-4.8 2.4 2.4 0 0 0 0 4.8Z" stroke="currentColor" strokeWidth="1.8" />
            <path d="M4.6 18.6c.6-2.4 2.3-3.8 4.8-3.8 2.6 0 4.2 1.4 4.8 3.8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
            <path d="M14.5 18.5c.4-1.7 1.6-2.8 3.3-2.8 1.1 0 2 .4 2.7 1.2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
    )
}

export default FlowStepIcon
