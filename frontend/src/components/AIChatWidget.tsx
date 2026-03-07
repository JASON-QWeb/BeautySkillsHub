import { useState, useRef, useEffect } from 'react'
import { useI18n } from '../i18n/I18nProvider'
import { chatWithAI } from '../services/api'
import { SkillsIntroModal } from './SkillsIntroModal'
import { SkillsInstallModal } from './SkillsInstallModal'
import { AIChatCharacter, SKINS, SkinType } from './AIChatCharacter'

interface Message {
    role: 'user' | 'assistant'
    content: string
}

const RECOMMENDATION_PROMPTS = [
    '介绍一下Skills',
    '目前有哪些skills',
    '如何安装skills',
]
const ACTION_SHOW_INTRO_MODAL = '[ACTION:SHOW_INTRO_MODAL]'
const ACTION_SHOW_INSTALL_MODAL = '[ACTION:SHOW_INSTALL_MODAL]'

function AIChatWidget() {
    const { t } = useI18n()
    const greeting = t('chat.greeting')
    const [isOpen, setIsOpen] = useState(false)
    const [isSkillsModalOpen, setIsSkillsModalOpen] = useState(false)
    const [isSkillsInstallModalOpen, setIsSkillsInstallModalOpen] = useState(false)
    const [showOnboarding, setShowOnboarding] = useState(false)
    const [messages, setMessages] = useState<Message[]>([{ role: 'assistant', content: greeting }])
    const [input, setInput] = useState('')
    const [isLoading, setIsLoading] = useState(false)
    const [showSkinSelector, setShowSkinSelector] = useState(false)
    const [currentSkin, setCurrentSkin] = useState<SkinType>(() => {
        return (localStorage.getItem('ai-skin') as SkinType) || 'cat'
    })
    const messagesEndRef = useRef<HTMLDivElement>(null)

    const INTRO_TRIGGERS = ['介绍一下skills', '介绍一下 skills', 'skills是什么', '什么是skills', '介绍skills', '介绍一下Skills']
    const INSTALL_TRIGGERS = ['如何安装skills', '如何安装 skills', '安装skills', '安装 skills', '怎么安装skills', '怎么安装 skills']
    
    useEffect(() => {
        if (!localStorage.getItem('ai_onboarding_seen')) {
            setShowOnboarding(true)
        }

        const handleSkinChange = (e: any) => {
            if (e.detail && typeof e.detail === 'string') {
                setCurrentSkin(e.detail as SkinType)
            }
        }
        window.addEventListener('ai-skin-changed', handleSkinChange)
        return () => window.removeEventListener('ai-skin-changed', handleSkinChange)
    }, [])

    const handleSkinSelect = (skin: SkinType) => {
        setCurrentSkin(skin)
        localStorage.setItem('ai-skin', skin)
        window.dispatchEvent(new CustomEvent('ai-skin-changed', { detail: skin }))
        setShowSkinSelector(false)
    }

    const handleOpenChat = () => {
        if (showOnboarding) {
            setShowOnboarding(false)
            localStorage.setItem('ai_onboarding_seen', 'true')
        }
        setIsOpen(true)
    }

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }

    useEffect(() => {
        scrollToBottom()
    }, [messages])

    useEffect(() => {
        setMessages(prev => {
            if (prev.length === 1 && prev[0]?.role === 'assistant') {
                return [{ role: 'assistant', content: greeting }]
            }
            return prev
        })
    }, [greeting])

    const handleSend = async (overrideMsg?: string) => {
        const userMsg = (overrideMsg || input).trim()
        if (!userMsg || isLoading) return

        if (!overrideMsg) setInput('')
        setMessages(prev => [...prev, { role: 'user', content: userMsg }])
        
        // Frontend Interception
        if (INSTALL_TRIGGERS.some(trigger => userMsg.toLowerCase() === trigger.toLowerCase())) {
            setMessages(prev => [...prev, { role: 'assistant', content: ACTION_SHOW_INSTALL_MODAL }])
            return
        }

        if (INTRO_TRIGGERS.some(trigger => userMsg.toLowerCase() === trigger.toLowerCase())) {
            setMessages(prev => [...prev, { role: 'assistant', content: ACTION_SHOW_INTRO_MODAL }])
            return
        }

        setIsLoading(true)

        // Add empty assistant message that will be streamed into
        setMessages(prev => [...prev, { role: 'assistant', content: '' }])

        chatWithAI(
            userMsg,
            (chunk) => {
                // Append chunk to the last assistant message
                setMessages(prev => {
                    const updated = [...prev]
                    const last = updated[updated.length - 1]
                    if (last.role === 'assistant') {
                        updated[updated.length - 1] = { ...last, content: last.content + chunk }
                    }
                    return updated
                })
            },
            () => {
                setIsLoading(false)
            },
            (error) => {
                setMessages(prev => {
                    const updated = [...prev]
                    const last = updated[updated.length - 1]
                    if (last.role === 'assistant' && last.content === '') {
                        updated[updated.length - 1] = { ...last, content: t('chat.errorPrefix') + error }
                    }
                    return updated
                })
                setIsLoading(false)
            },
        )
    }

    return (
        <>
            <AIChatCharacter
                isOpen={isOpen}
                isTyping={isLoading}
                showOnboarding={showOnboarding}
                onClick={handleOpenChat}
                style={{ opacity: (isOpen || isSkillsModalOpen || isSkillsInstallModalOpen) ? 0 : 1, pointerEvents: (isOpen || isSkillsModalOpen || isSkillsInstallModalOpen) ? 'none' : 'auto' }}
            />

            {/* Chat Modal */}
            <div className={`ai-chat-modal ${isOpen ? 'open' : ''}`}>
                <div className="ai-chat-header">
                    <div className="ai-chat-header-title">
                        <div className="ai-chat-status-avatar">
                            <div className={`ai-char-mini skin-${currentSkin}`} />
                        </div>
                        {t('chat.modalTitle')}
                    </div>
                    <div className="ai-chat-header-actions">
                        <button 
                            className={`ai-chat-skin-btn ${showSkinSelector ? 'active' : ''}`} 
                            onClick={() => setShowSkinSelector(!showSkinSelector)}
                        >
                            {t('chat.changeSkin')}
                        </button>
                        <button className="ai-chat-close" onClick={() => setIsOpen(false)}>x</button>
                    </div>

                    {showSkinSelector && (
                        <div className="ai-skin-selector glass-card">
                            {SKINS.map(s => (
                                <button 
                                    key={s} 
                                    className={`skin-option ${currentSkin === s ? 'selected' : ''}`}
                                    onClick={() => handleSkinSelect(s)}
                                >
                                    <div className={`ai-char-mini skin-${s}`} />
                                    <span>{t(`chat.skins.${s}` as any)}</span>
                                </button>
                            ))}
                        </div>
                    )}
                </div>

                <div className="ai-chat-messages">
                    {messages.map((msg, i) => (
                        <div key={i} className={`chat-message ${msg.role}`}>
                            {msg.content === ACTION_SHOW_INTRO_MODAL ? (
                                <button 
                                    className="chat-action-btn" 
                                    onClick={() => setIsSkillsModalOpen(true)}
                                >
                                    📖 点击查看 Skills 介绍
                                </button>
                            ) : msg.content === ACTION_SHOW_INSTALL_MODAL ? (
                                <button
                                    className="chat-action-btn"
                                    onClick={() => setIsSkillsInstallModalOpen(true)}
                                >
                                    🛠 点击查看 Skills 安装教程
                                </button>
                            ) : (
                                msg.content
                            )}
                        </div>
                    ))}
                    {isLoading && messages[messages.length - 1]?.content === '' && (
                        <div className="chat-message assistant">
                            <em>{t('chat.thinking')}</em>
                        </div>
                    )}
                    <div ref={messagesEndRef} />
                </div>

                <div className="ai-chat-quick-actions">
                    <div className="chat-recommendations">
                        {RECOMMENDATION_PROMPTS.map((prompt) => (
                            <button
                                key={prompt}
                                className="chat-recommendation-chip"
                                onClick={() => handleSend(prompt)}
                                disabled={isLoading}
                            >
                                {prompt}
                            </button>
                        ))}
                    </div>
                </div>

                <div className="ai-chat-input">
                    <input
                        type="text"
                        value={input}
                        onChange={e => setInput(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && handleSend()}
                        placeholder={t('chat.placeholder')}
                        disabled={isLoading}
                    />
                    <button onClick={() => handleSend()} disabled={isLoading || !input.trim()}>
                        {t('chat.send')}
                    </button>
                </div>
            </div>

            <SkillsIntroModal 
                isOpen={isSkillsModalOpen} 
                onClose={() => setIsSkillsModalOpen(false)} 
            />

            <SkillsInstallModal
                isOpen={isSkillsInstallModalOpen}
                onClose={() => setIsSkillsInstallModalOpen(false)}
            />
        </>
    )
}

export default AIChatWidget
