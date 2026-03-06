import { useState, useEffect, useCallback } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import SkillCard from '../components/SkillCard'
import RightSidebar from '../components/RightSidebar'
import { Skill, fetchSkills, fetchCategories, RESOURCE_TYPES } from '../services/api'

const NAV_ITEMS = [
    { path: 'skill', label: 'Skills', icon: '⚡' },
    { path: 'mcp', label: 'MCP', icon: '🔌' },
    { path: 'rules', label: 'Rules', icon: '📏' },
    { path: 'tools', label: 'Tools', icon: '🛠️' },
]

function HomePage() {
    const { type } = useParams<{ type: string }>()
    const resourceType = type || 'skill'
    const info = RESOURCE_TYPES[resourceType] || RESOURCE_TYPES.skill

    const navigate = useNavigate()
    const [skills, setSkills] = useState<Skill[]>([])
    const [search, setSearch] = useState('')
    const [loading, setLoading] = useState(true)
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const [category, setCategory] = useState('')
    const [categories, setCategories] = useState<string[]>([])
    const pageSize = 20

    useEffect(() => {
        fetchCategories(resourceType).then(setCategories).catch(() => { })
        setCategory('')
        setPage(1)
        setSearch('')
    }, [resourceType])

    const loadSkills = useCallback(async () => {
        setLoading(true)
        try {
            const data = await fetchSkills(search, page, pageSize, category, resourceType)
            setSkills(data.skills || [])
            setTotal(data.total)
        } catch (err) {
            console.error('Failed to load skills:', err)
        } finally {
            setLoading(false)
        }
    }, [search, page, category, resourceType])

    useEffect(() => {
        loadSkills()
    }, [loadSkills])

    useEffect(() => {
        const timer = setTimeout(() => {
            setPage(1)
        }, 300)
        return () => clearTimeout(timer)
    }, [search])

    const totalPages = Math.ceil(total / pageSize)

    return (
        <div className="home-container page-enter">
            <div className="skills-section">
                {/* Nav Tabs Row */}
                <div className="nav-tabs-row">
                    <div className="nav-tabs">
                        {NAV_ITEMS.map(item => (
                            <Link
                                key={item.path}
                                to={`/resource/${item.path}`}
                                className={`nav-tab ${resourceType === item.path ? 'active' : ''}`}
                            >
                                {item.icon} {item.label}
                            </Link>
                        ))}
                    </div>
                    <button
                        className="nav-tab-upload"
                        onClick={() => navigate(`/upload?type=${resourceType}`)}
                    >
                        ➕ Upload
                    </button>
                </div>

                {/* Search Bar - Full Width with Tabs */}
                <div className="skills-search-container" style={{ width: '100%', maxWidth: 'none', justifyContent: 'flex-start' }}>
                    <input
                        type="text"
                        className="skills-search-large"
                        style={{ maxWidth: '100%' }}
                        placeholder={`🔍 Search ${info.label} by name, description or category...`}
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        id="skills-search-input"
                    />
                </div>

                {/* Category filter chips */}
                {categories.length > 0 && (
                    <div className="filter-chips">
                        <button
                            className={`filter-chip ${category === '' ? 'active' : ''}`}
                            onClick={() => { setCategory(''); setPage(1) }}
                        >
                            All
                        </button>
                        {categories.map(cat => (
                            <button
                                key={cat}
                                className={`filter-chip ${category === cat ? 'active' : ''}`}
                                onClick={() => { setCategory(cat); setPage(1) }}
                            >
                                {cat}
                            </button>
                        ))}
                    </div>
                )}

                {loading ? (
                    <div className="empty-state">
                        <div className="loading-spinner" style={{ width: 40, height: 40 }}></div>
                        <p style={{ marginTop: 16 }}>Loading...</p>
                    </div>
                ) : skills.length === 0 ? (
                    <div className="empty-state">
                        <div className="icon">📦</div>
                        <p>No {info.label} yet. Upload the first one!</p>
                    </div>
                ) : (
                    <>
                        <div className="skills-grid">
                            {skills.map(skill => (
                                <SkillCard key={skill.id} skill={skill} />
                            ))}
                        </div>

                        {totalPages > 1 && (
                            <div style={{
                                display: 'flex',
                                justifyContent: 'center',
                                gap: 8,
                                marginTop: 24,
                            }}>
                                <button
                                    className="btn btn-secondary btn-sm"
                                    disabled={page <= 1}
                                    onClick={() => setPage(p => p - 1)}
                                >
                                    ← Prev
                                </button>
                                <span style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    color: 'var(--text-secondary)',
                                    fontSize: '0.88rem',
                                }}>
                                    {page} / {totalPages}
                                </span>
                                <button
                                    className="btn btn-secondary btn-sm"
                                    disabled={page >= totalPages}
                                    onClick={() => setPage(p => p + 1)}
                                >
                                    Next →
                                </button>
                            </div>
                        )}
                    </>
                )}
            </div>

            <RightSidebar />
        </div>
    )
}

export default HomePage
