import { useState, useRef, useEffect } from 'react'

interface Message {
    role: 'user' | 'assistant' | 'system'
    content: string
}

function AIChatWidget() {
    const [isOpen, setIsOpen] = useState(false)
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: '👋 Hi! I\'m the Skills Hub assistant.\n\nClick me anytime to find or analyze useful skills.' }
    ])
    const [input, setInput] = useState('')
    const [isLoading, setIsLoading] = useState(false)
    const messagesEndRef = useRef<HTMLDivElement>(null)

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }

    useEffect(() => {
        scrollToBottom()
    }, [messages])

    const handleSend = async () => {
        if (!input.trim() || isLoading) return

        const userMsg = input.trim()
        setInput('')
        setMessages(prev => [...prev, { role: 'user', content: userMsg }])
        setIsLoading(true)

        try {
            setTimeout(() => {
                setMessages(prev => [...prev, {
                    role: 'assistant',
                    content: 'This is a demo reply. Based on your input, I recommend checking related content.'
                }])
                setIsLoading(false)
            }, 1000)
        } catch (error) {
            console.error('Chat error:', error)
            setIsLoading(false)
        }
    }

    return (
        <>
            {/* Half-hidden floating bot icon */}
            <div
                className="ai-float-btn"
                onClick={() => setIsOpen(true)}
                title="AI Assistant"
                style={{ opacity: isOpen ? 0 : 1, pointerEvents: isOpen ? 'none' : 'auto' }}
            >
                🤖
            </div>

            {/* Chat Modal */}
            <div className={`ai-chat-modal ${isOpen ? 'open' : ''}`}>
                <div className="ai-chat-header">
                    <div className="ai-chat-header-title">
                        <span>🤖</span>
                        AI Assistant
                    </div>
                    <button className="ai-chat-close" onClick={() => setIsOpen(false)}>×</button>
                </div>

                <div className="ai-chat-messages">
                    {messages.map((msg, i) => (
                        <div key={i} className={`chat-message ${msg.role}`}>
                            {msg.content}
                        </div>
                    ))}
                    {isLoading && (
                        <div className="chat-message assistant">
                            <em>AI is thinking...</em>
                        </div>
                    )}
                    <div ref={messagesEndRef} />
                </div>

                <div className="ai-chat-input">
                    <input
                        type="text"
                        value={input}
                        onChange={e => setInput(e.target.value)}
                        onKeyPress={e => e.key === 'Enter' && handleSend()}
                        placeholder="Ask anything..."
                        disabled={isLoading}
                    />
                    <button onClick={handleSend} disabled={isLoading || !input.trim()}>
                        Send
                    </button>
                </div>
            </div>
        </>
    )
}

export default AIChatWidget
