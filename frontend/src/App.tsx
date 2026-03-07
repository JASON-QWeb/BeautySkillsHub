import { Routes, Route, Navigate, useParams } from 'react-router-dom'
import Navbar from './components/Navbar'
import HomePage from './pages/HomePage'
import ResourceDetailPage from './pages/ResourceDetailPage'
import UploadPage from './pages/UploadPage'
import ReviewPage from './pages/ReviewPage'
import ProfilePage from './pages/ProfilePage'
import AIChatWidget from './components/AIChatWidget'
import { AuthProvider } from './contexts/AuthContext'
import { DialogProvider } from './contexts/DialogContext'
import ProtectedRoute from './components/ProtectedRoute'

function LegacySkillDetailRedirect() {
    const { id } = useParams<{ id: string }>()
    if (!id) return <Navigate to="/resource/skill" replace />
    return <Navigate to={`/resource/skill/${id}`} replace />
}

function App() {
    return (
        <DialogProvider>
            <AuthProvider>
                <Navbar />
                <Routes>
                    <Route path="/" element={<Navigate to="/resource/all" replace />} />
                    <Route path="/resource" element={<Navigate to="/resource/all" replace />} />
                    <Route path="/resource/:type" element={<HomePage />} />
                    <Route path="/resource/:type/upload" element={<ProtectedRoute><UploadPage /></ProtectedRoute>} />
                    <Route path="/resource/:type/:id" element={<ResourceDetailPage />} />
                    <Route path="/skill/:id" element={<LegacySkillDetailRedirect />} />
                    <Route path="/review/:id" element={<ReviewPage />} />
                    <Route path="/upload" element={<Navigate to="/resource/skill/upload" replace />} />
                    <Route path="/profile" element={<ProtectedRoute><ProfilePage /></ProtectedRoute>} />
                </Routes>

                <AIChatWidget />
            </AuthProvider>
        </DialogProvider>
    )
}

export default App
