import { useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useI18n } from '../i18n/I18nProvider'
import AnimatedCharacters from './AnimatedCharacters'

interface LoginModalProps {
    isOpen: boolean
    onClose: () => void
}

export default function LoginModal({ isOpen, onClose }: LoginModalProps) {
    const { login, register } = useAuth()
    const { t } = useI18n()

    const [mode, setMode] = useState<'login' | 'register'>('login')
    const [username, setUsername] = useState('')
    const [password, setPassword] = useState('')
    const [confirmPassword, setConfirmPassword] = useState('')
    const [showPassword, setShowPassword] = useState(false)
    const [isTyping, setIsTyping] = useState(false)
    const [error, setError] = useState('')
    const [loading, setLoading] = useState(false)

    const resetForm = () => {
        setUsername('')
        setPassword('')
        setConfirmPassword('')
        setShowPassword(false)
        setError('')
        setLoading(false)
        setIsTyping(false)
    }

    const handleClose = () => {
        if (!loading) {
            resetForm()
            onClose()
        }
    }

    const switchMode = () => {
        setMode(m => m === 'login' ? 'register' : 'login')
        setError('')
        setPassword('')
        setConfirmPassword('')
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setError('')

        if (!username.trim() || !password) {
            setError(t('login.fillFields'))
            return
        }

        if (mode === 'register') {
            if (username.trim().length < 2) {
                setError(t('login.usernameMin'))
                return
            }
            if (password.length < 6) {
                setError(t('login.passwordMin'))
                return
            }
            if (password !== confirmPassword) {
                setError(t('login.passwordMismatch'))
                return
            }
        }

        setLoading(true)
        try {
            if (mode === 'login') {
                await login(username.trim(), password)
            } else {
                await register(username.trim(), password)
            }
            resetForm()
            onClose()
        } catch (err) {
            setError(err instanceof Error ? err.message : t('login.genericError'))
        } finally {
            setLoading(false)
        }
    }

    if (!isOpen) return null

    return (
        <div className="login-modal-overlay" onClick={e => { if (e.target === e.currentTarget) handleClose() }}>
            <div className="login-modal-container">
                {/* Left side - animated characters */}
                <div className="login-modal-left">
                    <div className="login-modal-characters">
                        <AnimatedCharacters
                            isTyping={isTyping}
                            showPassword={showPassword}
                            passwordLength={password.length}
                        />
                    </div>
                    <div className="login-modal-left-text">
                        <h2>{t('common.skillsHub')}</h2>
                        <p>{t('login.leftTagline')}</p>
                    </div>
                </div>

                {/* Right side - form */}
                <div className="login-modal-right">
                    <button className="login-modal-close" onClick={handleClose} disabled={loading}>
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                        </svg>
                    </button>

                    <div className="login-modal-header">
                        <h2>{mode === 'login' ? t('login.welcomeBack') : t('login.createAccount')}</h2>
                        <p>{mode === 'login' ? t('login.enterCredentials') : t('login.signUpToShare')}</p>
                    </div>

                    <form className="login-modal-form" onSubmit={handleSubmit}>
                        <div className="login-form-group">
                            <label htmlFor="login-username">{t('login.username')}</label>
                            <div className="login-input-wrapper">
                                <svg className="login-input-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
                                </svg>
                                <input
                                    id="login-username"
                                    type="text"
                                    placeholder={t('login.usernamePlaceholder')}
                                    value={username}
                                    onChange={e => setUsername(e.target.value)}
                                    onFocus={() => setIsTyping(true)}
                                    onBlur={() => setIsTyping(false)}
                                    disabled={loading}
                                    autoComplete="username"
                                />
                            </div>
                        </div>

                        <div className="login-form-group">
                            <label htmlFor="login-password">{t('login.password')}</label>
                            <div className="login-input-wrapper">
                                <svg className="login-input-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" />
                                </svg>
                                <input
                                    id="login-password"
                                    type={showPassword ? 'text' : 'password'}
                                    placeholder={t('login.passwordPlaceholder')}
                                    value={password}
                                    onChange={e => setPassword(e.target.value)}
                                    disabled={loading}
                                    autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                                />
                                <button
                                    type="button"
                                    className="login-password-toggle"
                                    onClick={() => setShowPassword(!showPassword)}
                                    tabIndex={-1}
                                >
                                    {showPassword ? (
                                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                                            <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                                            <line x1="1" y1="1" x2="23" y2="23" />
                                        </svg>
                                    ) : (
                                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" />
                                        </svg>
                                    )}
                                </button>
                            </div>
                        </div>

                        {mode === 'register' && (
                            <div className="login-form-group">
                                <label htmlFor="login-confirm-password">{t('login.confirmPassword')}</label>
                                <div className="login-input-wrapper">
                                    <svg className="login-input-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" />
                                    </svg>
                                    <input
                                        id="login-confirm-password"
                                        type={showPassword ? 'text' : 'password'}
                                        placeholder={t('login.confirmPasswordPlaceholder')}
                                        value={confirmPassword}
                                        onChange={e => setConfirmPassword(e.target.value)}
                                        disabled={loading}
                                        autoComplete="new-password"
                                    />
                                </div>
                            </div>
                        )}

                        {error && (
                            <div className="login-error">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <circle cx="12" cy="12" r="10" /><line x1="15" y1="9" x2="9" y2="15" /><line x1="9" y1="9" x2="15" y2="15" />
                                </svg>
                                {error}
                            </div>
                        )}

                        {/* Keycloak SSO placeholder - hidden for now */}
                        {/* <button type="button" className="login-sso-btn" onClick={handleKeycloakLogin}>Enterprise SSO</button> */}

                        <button type="submit" className="login-submit-btn" disabled={loading}>
                            {loading
                                ? (mode === 'login' ? t('login.signingIn') : t('login.creatingAccount'))
                                : (mode === 'login' ? t('login.signIn') : t('login.createAccountBtn'))
                            }
                        </button>
                    </form>

                    <div className="login-modal-footer">
                        <span>{mode === 'login' ? t('login.noAccount') : t('login.hasAccount')}</span>
                        <button type="button" className="login-switch-btn" onClick={switchMode}>
                            {mode === 'login' ? t('login.signUp') : t('login.signIn')}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}
