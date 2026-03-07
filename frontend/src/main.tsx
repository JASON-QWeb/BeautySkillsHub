import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { ThemeProvider } from './contexts/ThemeContext'
import { I18nProvider } from './i18n/I18nProvider'
import './index.css'

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <ThemeProvider>
            <I18nProvider>
                <BrowserRouter>
                    <App />
                </BrowserRouter>
            </I18nProvider>
        </ThemeProvider>
    </StrictMode>,
)
