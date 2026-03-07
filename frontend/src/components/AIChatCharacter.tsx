import { useState, useRef, useEffect } from 'react'
import { useI18n } from '../i18n/I18nProvider'

export type SkinType = 'default' | 'robot' | 'ninja' | 'cat' | 'ghost' | 'dragon'
export const SKINS: SkinType[] = ['default', 'robot', 'ninja', 'cat', 'ghost', 'dragon']

interface AIChatCharacterProps {
    isOpen?: boolean
    isTyping?: boolean
    showOnboarding?: boolean
    onClick?: () => void
    className?: string
    style?: React.CSSProperties
}

export function AIChatCharacter({ isOpen, isTyping, showOnboarding, onClick, className = '', style }: AIChatCharacterProps) {
    const { t } = useI18n()
    const [mouseX, setMouseX] = useState(0)
    const [mouseY, setMouseY] = useState(0)
    const [blinking, setBlinking] = useState(false)
    const [peekUp, setPeekUp] = useState(false)
    const [userTyping, setUserTyping] = useState(false)
    const bodyRef = useRef<HTMLDivElement>(null)
    const userTypingTimer = useRef<ReturnType<typeof setTimeout>>()

    const [skin, setSkin] = useState<SkinType>(() => {
        return (localStorage.getItem('ai-skin') as SkinType) || 'default'
    })

    useEffect(() => {
        const handleSkinChange = (e: any) => {
            if (e.detail && typeof e.detail === 'string') {
                setSkin(e.detail as SkinType)
            }
        }
        window.addEventListener('ai-skin-changed', handleSkinChange)
        return () => window.removeEventListener('ai-skin-changed', handleSkinChange)
    }, [])

    useEffect(() => {
        const onMove = (e: MouseEvent) => { setMouseX(e.clientX); setMouseY(e.clientY) }
        window.addEventListener('mousemove', onMove)
        return () => window.removeEventListener('mousemove', onMove)
    }, [])

    useEffect(() => {
        let id: ReturnType<typeof setTimeout>
        const schedule = () => {
            id = setTimeout(() => {
                setBlinking(true)
                setTimeout(() => { setBlinking(false); schedule() }, 150)
            }, Math.random() * 4000 + 2500)
        }
        schedule()
        return () => clearTimeout(id)
    }, [])

    useEffect(() => {
        const isTypableElement = (el: EventTarget | null): boolean => {
            if (!el || !(el instanceof HTMLElement)) return false
            const tag = el.tagName
            if (tag === 'INPUT' || tag === 'TEXTAREA') return true
            if (el.isContentEditable) return true
            return false
        }

        const onFocusIn = (e: FocusEvent) => {
            if (isTypableElement(e.target)) {
                clearTimeout(userTypingTimer.current)
                setUserTyping(true)
            }
        }

        const onFocusOut = () => {
            clearTimeout(userTypingTimer.current)
            userTypingTimer.current = setTimeout(() => setUserTyping(false), 400)
        }

        const onKeyDown = (e: KeyboardEvent) => {
            if (isTypableElement(e.target)) {
                clearTimeout(userTypingTimer.current)
                setUserTyping(true)
            }
        }

        document.addEventListener('focusin', onFocusIn)
        document.addEventListener('focusout', onFocusOut)
        document.addEventListener('keydown', onKeyDown)
        return () => {
            document.removeEventListener('focusin', onFocusIn)
            document.removeEventListener('focusout', onFocusOut)
            document.removeEventListener('keydown', onKeyDown)
            clearTimeout(userTypingTimer.current)
        }
    }, [])

    useEffect(() => {
        if (isTyping || userTyping) {
            setPeekUp(true)
        } else {
            const t = setTimeout(() => setPeekUp(false), 600)
            return () => clearTimeout(t)
        }
    }, [isTyping, userTyping])

    const eyePos = (() => {
        if (!bodyRef.current) return { x: 0, y: 0 }
        const rect = bodyRef.current.getBoundingClientRect()
        const dx = mouseX - (rect.left + rect.width / 2)
        const dy = mouseY - (rect.top + rect.height / 2)
        const dist = Math.min(Math.sqrt(dx * dx + dy * dy), 4)
        const angle = Math.atan2(dy, dx)
        return { x: Math.cos(angle) * dist, y: Math.sin(angle) * dist }
    })()

    const handleContextMenu = (e: React.MouseEvent) => {
        e.preventDefault()
        const nextSkin = SKINS[(SKINS.indexOf(skin) + 1) % SKINS.length]
        setSkin(nextSkin)
        localStorage.setItem('ai-skin', nextSkin)
        window.dispatchEvent(new CustomEvent('ai-skin-changed', { detail: nextSkin }))
    }

    return (
        <div
            className={`ai-float-character skin-${skin} ${showOnboarding && !isOpen ? 'onboarding' : ''} ${className}`}
            onClick={onClick}
            onContextMenu={handleContextMenu}
            title="Right-click to change skin"
            style={{ opacity: isOpen ? 0 : 1, pointerEvents: isOpen ? 'none' : (onClick ? 'auto' : 'none'), ...style }}
        >
            <div
                ref={bodyRef}
                className="ai-char-body"
                style={{
                    height: peekUp ? 90 : 70,
                    transform: peekUp
                        ? 'translateY(-22px) skewX(-10deg) translateX(8px)'
                        : 'translateY(0px) skewX(0deg) translateX(0px)',
                }}
            >
                {/* Animal Tails/Ears */}
                {skin === 'cat' && (
                    <>
                        <div className="ai-char-tail" />
                        <div className="ai-char-ears">
                            <div className="ai-char-ear left" />
                            <div className="ai-char-ear right" />
                        </div>
                    </>
                )}

                {skin === 'dragon' && (
                    <>
                        <div className="ai-char-horns">
                            <div className="ai-char-horn left" />
                            <div className="ai-char-horn right" />
                        </div>
                        <div className="ai-char-wings">
                            <div className="ai-char-wing left" />
                            <div className="ai-char-wing right" />
                        </div>
                    </>
                )}
                
                {/* Body */}
                <div className="ai-char-shape">
                    {/* Ghost Waves */}
                    {skin === 'ghost' && (
                        <div className="ai-char-waves">
                            <div className="wave" />
                            <div className="wave" />
                            <div className="wave" />
                        </div>
                    )}
                </div>

                {/* Eyes Area */}
                <div className="ai-char-eyes-container">
                    {skin === 'ninja' && <div className="ai-char-headband" />}
                    <div className="ai-char-eyes">
                        <div
                            className="ai-char-eye"
                            style={{ height: blinking ? 2 : (skin === 'robot' ? 8 : 12) }}
                        >
                            {!blinking && (
                                <div
                                    className="ai-char-pupil"
                                    style={{ transform: `translate(${eyePos.x}px, ${eyePos.y}px)` }}
                                />
                            )}
                        </div>
                        <div
                            className="ai-char-eye"
                            style={{ height: blinking ? 2 : (skin === 'robot' ? 8 : 12) }}
                        >
                            {!blinking && (
                                <div
                                    className="ai-char-pupil"
                                    style={{ transform: `translate(${eyePos.x}px, ${eyePos.y}px)` }}
                                />
                            )}
                        </div>
                    </div>
                </div>
            </div>
            {showOnboarding && !isOpen && (
                <div className="ai-onboarding-bubble">
                    {t('chat.onboardingTooltip')}
                </div>
            )}
        </div>
    )
}
