import { Routes, Route, Navigate } from 'react-router-dom'
import Navbar from './components/Navbar'
import HomePage from './pages/HomePage'
import SkillDetailPage from './pages/SkillDetailPage'
import UploadPage from './pages/UploadPage'
import ReviewPage from './pages/ReviewPage'
import ProfilePage from './pages/ProfilePage'
import AIChatWidget from './components/AIChatWidget'
import { AuthProvider } from './contexts/AuthContext'
import { DialogProvider } from './contexts/DialogContext'
import ProtectedRoute from './components/ProtectedRoute'

function App() {
    return (
        <DialogProvider>
            <AuthProvider>
                <Navbar />
                <Routes>
                    <Route path="/" element={<Navigate to="/resource/skill" replace />} />
                    <Route path="/resource/:type" element={<HomePage />} />
                    <Route path="/skill/:id" element={<SkillDetailPage />} />
                    <Route path="/review/:id" element={<ReviewPage />} />
                    <Route path="/upload" element={<ProtectedRoute><UploadPage /></ProtectedRoute>} />
                    <Route path="/profile" element={<ProtectedRoute><ProfilePage /></ProtectedRoute>} />
                </Routes>

                <AIChatWidget />
            </AuthProvider>
        </DialogProvider>
    )
}

export default App
