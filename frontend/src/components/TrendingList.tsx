import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Skill, fetchTrending } from '../services/api'

function TrendingList() {
    const [skills, setSkills] = useState<Skill[]>([])
    const [loading, setLoading] = useState(true)
    const navigate = useNavigate()

    const loadTrending = async () => {
        try {
            const data = await fetchTrending(10)
            setSkills(data)
        } catch (err) {
            console.error('加载趋势榜单失败:', err)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadTrending()
        // Refresh every 30 seconds
        const timer = setInterval(loadTrending, 30000)
        return () => clearInterval(timer)
    }, [])

    return (
        <div className="trending-panel glass-card">
            <h3>🔥 热门趋势</h3>
            {loading ? (
                <div className="empty-state">
                    <div className="loading-spinner"></div>
                </div>
            ) : skills.length === 0 ? (
                <div className="empty-state">
                    <div className="icon">📊</div>
                    <p>暂无趋势数据</p>
                </div>
            ) : (
                skills.map((skill, index) => (
                    <div
                        key={skill.id}
                        className="trending-item"
                        onClick={() => navigate(`/skill/${skill.id}`)}
                    >
                        <span className={`trending-rank ${index < 3 ? 'top-3' : ''}`}>
                            {index + 1}
                        </span>
                        <div className="trending-info">
                            <div className="trending-name">{skill.name}</div>
                            <div className="trending-downloads">📥 {skill.downloads} 次下载</div>
                        </div>
                    </div>
                ))
            )}
        </div>
    )
}

export default TrendingList
