import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Skill, fetchTrending, fetchSkills } from '../services/api'

function RightSidebar() {
    const [hotest, setHotest] = useState<Skill[]>([])
    const [newest, setNewest] = useState<Skill[]>([])
    const [loading, setLoading] = useState(true)
    const navigate = useNavigate()

    const loadData = async () => {
        try {
            const [trendingData, newestData] = await Promise.all([
                fetchTrending(5),
                fetchSkills('', 1, 5)
            ])
            setHotest(trendingData)
            setNewest(newestData.skills || [])
        } catch (err) {
            console.error('Failed to load sidebar:', err)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadData()
        const timer = setInterval(loadData, 30000)
        return () => clearInterval(timer)
    }, [])

    const renderList = (skills: Skill[], icon: string, title: string) => (
        <div className="sidebar-card glass-card">
            <h3>{icon} {title}</h3>
            {loading ? (
                <div className="empty-state" style={{ padding: '20px' }}>
                    <div className="loading-spinner"></div>
                </div>
            ) : skills.length === 0 ? (
                <div className="empty-state" style={{ padding: '20px' }}>
                    <p style={{ fontSize: '0.85rem' }}>No data</p>
                </div>
            ) : (
                skills.map((skill, index) => (
                    <div
                        key={skill.id}
                        className="sidebar-item"
                        onClick={() => navigate(`/skill/${skill.id}`)}
                    >
                        <span className={`sidebar-rank ${index < 3 ? 'top-3' : ''}`}>
                            {index + 1}
                        </span>
                        <div className="sidebar-info">
                            <div className="sidebar-name">{skill.name}</div>
                            <div className="sidebar-meta">📥 {skill.downloads} downloads</div>
                        </div>
                    </div>
                ))
            )}
        </div>
    )

    return (
        <div className="right-sidebar">
            {renderList(hotest, '🔥', 'Hottest')}
            {renderList(newest, '✨', 'Newest')}

            <div className="sidebar-card glass-card">
                <h3>🏆 Top Uploads</h3>
                <div style={{ padding: '0' }}>
                    <div className="sidebar-item">
                        <span className="sidebar-rank top-3">1</span>
                        <div className="sidebar-info">
                            <div className="sidebar-name">Admin</div>
                            <div className="sidebar-meta">🚀 42 works</div>
                        </div>
                    </div>
                    <div className="sidebar-item">
                        <span className="sidebar-rank top-3">2</span>
                        <div className="sidebar-info">
                            <div className="sidebar-name">Developer_X</div>
                            <div className="sidebar-meta">🚀 18 works</div>
                        </div>
                    </div>
                    <div className="sidebar-item" style={{ borderBottom: 'none' }}>
                        <span className="sidebar-rank top-3">3</span>
                        <div className="sidebar-info">
                            <div className="sidebar-name">AI_Bot</div>
                            <div className="sidebar-meta">🚀 9 works</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default RightSidebar
