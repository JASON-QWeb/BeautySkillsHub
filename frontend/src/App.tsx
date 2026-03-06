import { Routes, Route, Navigate } from 'react-router-dom'
import Navbar from './components/Navbar'
import HomePage from './pages/HomePage'
import SkillDetailPage from './pages/SkillDetailPage'
import UploadPage from './pages/UploadPage'
import AIChatWidget from './components/AIChatWidget'

function App() {
    return (
        <>
            <Navbar />
            <Routes>
                <Route path="/" element={<Navigate to="/resource/skill" replace />} />
                <Route path="/resource/:type" element={<HomePage />} />
                <Route path="/skill/:id" element={<SkillDetailPage />} />
                <Route path="/upload" element={<UploadPage />} />
            </Routes>

            <AIChatWidget />
        </>
    )
}

export default App
