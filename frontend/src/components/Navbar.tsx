import { Link } from 'react-router-dom'
import { useEffect, useState } from 'react'

function Navbar() {
    const [theme, setTheme] = useState(localStorage.getItem('app-theme') || 'dark')

    useEffect(() => {
        document.documentElement.setAttribute('data-theme', theme)
        localStorage.setItem('app-theme', theme)
    }, [theme])

    const toggleTheme = () => {
        const themes = ['dark', 'glass']
        const currentIndex = themes.indexOf(theme)
        const nextTheme = themes[(currentIndex + 1) % themes.length]
        setTheme(nextTheme)
    }

    return (
        <nav className="navbar">
            <Link to="/" className="navbar-brand">
                <span className="logo-icon">⚡</span>
                <span>Skills Hub</span>
            </Link>
            <div className="navbar-actions">
                <button
                    onClick={toggleTheme}
                    className="theme-toggle-btn"
                    title="Toggle Theme"
                >
                    {theme === 'dark' ? '🌙' : '☀️'}
                </button>
            </div>
        </nav>
    )
}

export default Navbar
