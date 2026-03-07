import { ReactNode, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useI18n } from '../i18n/I18nProvider'
import LoginModal from './LoginModal'

export default function ProtectedRoute({ children }: { children: ReactNode }) {
    const { user, loading } = useAuth()
    const { t } = useI18n()
    const [loginOpen, setLoginOpen] = useState(false)

    if (loading) {
        return (
            <div className="empty-state">
                <div className="loading-spinner" style={{ width: 40, height: 40 }}></div>
            </div>
        )
    }

    if (!user) {
        return (
            <div className="empty-state" style={{ marginTop: 120 }}>
                <div className="icon" style={{ fontSize: '3rem' }}>🔒</div>
                <h2 style={{ marginBottom: 8 }}>{t('protected.signInRequired')}</h2>
                <p style={{ color: 'var(--text-secondary)', marginBottom: 24 }}>
                    {t('protected.signInToUpload')}
                </p>
                <button className="btn btn-primary" onClick={() => setLoginOpen(true)}>
                    {t('login.signIn')}
                </button>
                <LoginModal isOpen={loginOpen} onClose={() => setLoginOpen(false)} />
            </div>
        )
    }

    return <>{children}</>
}
