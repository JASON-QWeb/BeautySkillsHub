import { Link, useLocation } from 'react-router-dom'
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { useI18n } from '../i18n/I18nProvider'
import LoginModal from './LoginModal'

const NAV_ITEMS = [
    { path: 'all', labelKey: 'nav.all' },
    { path: 'skill', labelKey: 'nav.skills' },
    { path: 'rules', labelKey: 'nav.rules' },
    { path: 'mcp', labelKey: 'nav.mcp' },
    { path: 'tools', labelKey: 'nav.tools' },
]

const NAV_TYPE_STORAGE_KEY = 'last_resource_type'

function normalizeResourceType(value?: string | null): string {
    if (!value) return 'all'
    const normalized = value.toLowerCase()
    if (normalized === 'skills') return 'skill'
    return NAV_ITEMS.some(item => item.path === normalized) ? normalized : 'skill'
}

function Navbar() {
    const location = useLocation()
    const { user, logout } = useAuth()
    const { theme, toggleTheme } = useTheme()
    const { language, toggleLanguage, t } = useI18n()

    const [loginOpen, setLoginOpen] = useState(false)
    const [indicator, setIndicator] = useState({ x: 0, width: 0, visible: false })
    const tabsRef = useRef<Array<HTMLAnchorElement | null>>([])

    const activeResourceType = (() => {
        const byPath = NAV_ITEMS.find(item => location.pathname.startsWith(`/resource/${item.path}`))
        if (byPath) return byPath.path

        if (location.pathname.startsWith('/skill/')) {
            const stateType = (location.state as { resourceType?: string } | null)?.resourceType
            if (stateType) return normalizeResourceType(stateType)
            return normalizeResourceType(window.localStorage.getItem(NAV_TYPE_STORAGE_KEY))
        }

        if (location.pathname.startsWith('/upload')) {
            const searchType = new URLSearchParams(location.search).get('type')
            if (searchType) return normalizeResourceType(searchType)
            return normalizeResourceType(window.localStorage.getItem(NAV_TYPE_STORAGE_KEY))
        }

        return ''
    })()

    const activeIndex = NAV_ITEMS.findIndex(item => item.path === activeResourceType)

    useEffect(() => {
        if (location.pathname.startsWith('/resource/')) {
            window.localStorage.setItem(NAV_TYPE_STORAGE_KEY, activeResourceType)
        }
    }, [location.pathname, activeResourceType])

    const updateIndicator = useCallback(() => {
        if (activeIndex < 0) {
            setIndicator(prev => ({ ...prev, visible: false }))
            return
        }

        const el = tabsRef.current[activeIndex]
        if (!el) {
            setIndicator(prev => ({ ...prev, visible: false }))
            return
        }

        setIndicator({
            x: el.offsetLeft,
            width: el.offsetWidth,
            visible: true,
        })
    }, [activeIndex])

    useLayoutEffect(() => {
        updateIndicator()
    }, [updateIndicator, language])

    useEffect(() => {
        window.addEventListener('resize', updateIndicator)
        return () => window.removeEventListener('resize', updateIndicator)
    }, [updateIndicator])

    return (
        <>
            <nav className="navbar">
                <Link to="/resource/all" className="navbar-brand">
                    <span className="brand-name">{t('common.skillsHub')}</span>
                </Link>

                <div className="navbar-center">
                    <span
                        className={`navbar-active-indicator ${indicator.visible ? 'visible' : ''}`}
                        style={{
                            width: `${indicator.width}px`,
                            transform: `translateX(${indicator.x}px)`,
                        }}
                    />
                    {NAV_ITEMS.map((item, index) => {
                        const active = index === activeIndex
                        return (
                            <Link
                                key={item.path}
                                to={`/resource/${item.path}`}
                                ref={el => { tabsRef.current[index] = el }}
                                className={`navbar-tab ${active ? 'active' : ''}`}
                            >
                                <span>{t(item.labelKey)}</span>
                            </Link>
                        )
                    })}
                </div>

                <div className="navbar-actions">
                    <button className="nav-lang-switch" onClick={toggleLanguage} title={t('navbar.toggleLanguage')}>
                        <span className={language === 'en' ? '' : 'active'}>中</span>
                        <span className={language === 'en' ? 'active' : ''}>EN</span>
                    </button>

                    <label className="switch-name" title={t('navbar.toggleTheme')}>
                        <input 
                            type="checkbox" 
                            className="checkbox" 
                            checked={theme === 'light'} 
                            onChange={toggleTheme} 
                        />
                        <div className="back"></div>
                        <svg className="moon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 384 512">
                            <path d="M223.5 32C100 32 0 132.3 0 256S100 480 223.5 480c60.6 0 115.5-24.2 155.8-63.4c5-4.9 6.3-12.5 3.1-18.7s-10.1-9.7-17-8.5c-9.8 1.7-19.8 2.6-30.1 2.6c-96.9 0-175.5-78.8-175.5-176c0-65.8 36-123.1 89.3-153.3c6.1-3.5 9.2-10.5 7.7-17.3s-7.3-11.9-14.3-12.5c-6.3-.5-12.6-.8-19-.8z"></path>
                        </svg>
                        <svg className="sun" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
                            <path d="M361.5 1.2c5 2.1 8.6 6.6 9.6 11.9L391 121l107.9 19.8c5.3 1 9.8 4.6 11.9 9.6s1.5 10.7-1.6 15.2L446.9 256l62.3 90.3c3.1 4.5 3.7 10.2 1.6 15.2s-6.6 8.6-11.9 9.6L391 391 371.1 498.9c-1 5.3-4.6 9.8-9.6 11.9s-10.7 1.5-15.2-1.6L256 446.9l-90.3 62.3c-4.5 3.1-10.2 3.7-15.2 1.6s-8.6-6.6-9.6-11.9L121 391 13.1 371.1c-5.3-1-9.8-4.6-11.9-9.6s-1.5-10.7 1.6-15.2L65.1 256 2.8 165.7c-3.1-4.5-3.7-10.2-1.6-15.2s6.6-8.6 11.9-9.6L121 121 140.9 13.1c1-5.3 4.6-9.8 9.6-11.9s10.7-1.5 15.2 1.6L256 65.1 346.3 2.8c4.5-3.1 10.2-3.7 15.2-1.6zM160 256a96 96 0 1 1 192 0 96 96 0 1 1 -192 0zm224 0a128 128 0 1 0 -256 0 128 128 0 1 0 256 0z"></path>
                        </svg>
                    </label>

                    <div className="navbar-divider" />

                    {user ? (
                        <>
                            <Link to="/profile" className="navbar-user" title={t('navbar.profile')}>
                                {user.avatar_url ? (
                                    <img src={user.avatar_url} alt={user.username} className="navbar-user-avatar" />
                                ) : (
                                    <span className="navbar-user-avatar">{user.username[0].toUpperCase()}</span>
                                )}
                            </Link>
                            <button className="nav-logout-btn" onClick={logout} title={t('navbar.signOut')}>⎋</button>
                        </>
                    ) : (
                        <button className="navbar-login-btn" onClick={() => setLoginOpen(true)}>
                            {t('navbar.signIn')}
                        </button>
                    )}
                </div>
            </nav>

            <LoginModal isOpen={loginOpen} onClose={() => setLoginOpen(false)} />
        </>
    )
}

export default Navbar
