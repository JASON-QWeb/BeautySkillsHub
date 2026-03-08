import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Skill, fetchTrending } from '../services/api'
import { isAbortError } from '../services/api/request'

function TrendingList() {
    const [skills, setSkills] = useState<Skill[]>([])
    const [loading, setLoading] = useState(true)
    const navigate = useNavigate()

    useEffect(() => {
        const controller = new AbortController()

        const loadTrending = async () => {
            try {
                const data = await fetchTrending(10, '', { signal: controller.signal })
                setSkills(data)
            } catch (err) {
                if (isAbortError(err)) return
                console.error('加载趋势榜单失败:', err)
            } finally {
                if (!controller.signal.aborted) {
                    setLoading(false)
                }
            }
        }

        void loadTrending()
        // Refresh every 30 seconds
        const timer = setInterval(() => {
            void loadTrending()
        }, 30000)
        return () => {
            controller.abort()
            clearInterval(timer)
        }
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
                        onClick={() => navigate(`/resource/${skill.resource_type || 'skill'}/${skill.id}`, { state: { resourceType: skill.resource_type } })}
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
