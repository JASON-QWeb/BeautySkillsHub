import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import LoginModal from '../../components/LoginModal'
import RightSidebar from '../../components/RightSidebar'
import SkillCard from '../../components/SkillCard'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { RESOURCE_TYPES, Skill, fetchSkillSummary, fetchSkills } from '../../services/api'

type HomeResourceType = 'all' | 'skill' | 'rules' | 'mcp' | 'tools'

function normalizeHomeResourceType(rawType: string | undefined): HomeResourceType {
    const normalized = (rawType || '').trim().toLowerCase()
    if (normalized === 'skills') return 'skill'
    if (normalized === 'all' || normalized === 'skill' || normalized === 'rules' || normalized === 'mcp' || normalized === 'tools') {
        return normalized
    }
    return 'all'
}

function HomePage() {
    const { type } = useParams<{ type: string }>()
    const normalizedType = normalizeHomeResourceType(type)
    const isOverview = normalizedType === 'all'
    const resourceTypeFilter = isOverview ? '' : normalizedType
    const info = RESOURCE_TYPES[normalizedType] || RESOURCE_TYPES.all

    const navigate = useNavigate()
    const { user } = useAuth()
    const { t } = useI18n()

    const [skills, setSkills] = useState<Skill[]>([])
    const [loading, setLoading] = useState(true)
    const [summary, setSummary] = useState({ total: 0, yesterday_new: 0 })
    const [search, setSearch] = useState('')
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const [loginOpen, setLoginOpen] = useState(false)
    const [animKey, setAnimKey] = useState(0)
    const [filterTag, setFilterTag] = useState('')
    const [filterOpen, setFilterOpen] = useState(false)
    const [allTags, setAllTags] = useState<string[]>([])
    const filterRef = useRef<HTMLDivElement>(null)

    const pageSize = 20

    useEffect(() => {
        setSearch('')
        setPage(1)
        setFilterTag('')
        setAnimKey(prev => prev + 1)
    }, [normalizedType])

    // Load available tags for filter
    useEffect(() => {
        const loadTags = async () => {
            try {
                const data = await fetchSkills('', 1, 100, '', resourceTypeFilter)
                const counts: Record<string, number> = {}
                for (const skill of data.skills || []) {
                    if (skill.tags) {
                        for (const tag of skill.tags.split(',')) {
                            const t = tag.trim().toLowerCase()
                            if (t) counts[t] = (counts[t] || 0) + 1
                        }
                    }
                }
                const sorted = Object.entries(counts)
                    .sort((a, b) => b[1] - a[1])
                    .map(([tag]) => tag)
                setAllTags(sorted)
            } catch {
                setAllTags([])
            }
        }
        loadTags()
    }, [resourceTypeFilter])

    // Close filter dropdown on outside click
    useEffect(() => {
        const handler = (e: MouseEvent) => {
            if (filterRef.current && !filterRef.current.contains(e.target as Node)) {
                setFilterOpen(false)
            }
        }
        if (filterOpen) document.addEventListener('mousedown', handler)
        return () => document.removeEventListener('mousedown', handler)
    }, [filterOpen])

    useEffect(() => {
        const loadSummary = async () => {
            try {
                const data = await fetchSkillSummary(resourceTypeFilter)
                setSummary({
                    total: data.total || 0,
                    yesterday_new: data.yesterday_new || 0,
                })
            } catch (err) {
                console.error('Failed to load summary:', err)
                setSummary({ total: 0, yesterday_new: 0 })
            }
        }

        loadSummary()
    }, [resourceTypeFilter])

    const loadSkills = useCallback(async () => {
        setLoading(true)
        try {
            const data = await fetchSkills(search, page, pageSize, '', resourceTypeFilter)
            setSkills(data.skills || [])
            setTotal(data.total || 0)
        } catch (err) {
            console.error('Failed to load skills:', err)
            setSkills([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }, [search, page, resourceTypeFilter])

    useEffect(() => {
        loadSkills()
    }, [loadSkills])

    const totalPages = Math.max(1, Math.ceil(total / pageSize))

    const filteredSkills = useMemo(() => {
        if (!filterTag) return skills
        return skills.filter(skill =>
            skill.tags?.split(',').some(t => t.trim().toLowerCase() === filterTag)
        )
    }, [skills, filterTag])

    const totalDownloads = useMemo(() => {
        return skills.reduce((sum, skill) => sum + (skill.downloads || 0), 0)
    }, [skills])

    const displayName = user?.username || t('home.visitor')

    const handleUpload = () => {
        if (user) {
            const uploadType = isOverview ? 'skill' : normalizedType
            navigate(`/resource/${uploadType}/upload`)
            return
        }
        setLoginOpen(true)
    }

    return (
        <div className="home-container page-enter">
            <section className="home-main">
                <header className="home-hero">
                    <div className="home-welcome">
                        <h1>
                            {t('home.welcomePrefix')} <strong>{displayName}</strong>
                        </h1>

                        <div className="ai-summary-card solid-dark-card">
                            <div className="ai-summary-header">
                                <span className="ai-summary-dot" />
                                <p className="ai-summary-title">{t('home.summaryTitle')}</p>
                                <span className="ai-summary-badge">{t('common.today')}</span>
                            </div>
                            <p className="ai-summary-text">
                                {t('home.summaryText', {
                                    label: info.label,
                                    total: summary.total,
                                    yesterday: summary.yesterday_new,
                                })}
                            </p>
                        </div>
                    </div>

                    <div className="home-stats">
                        <div className="home-stat-item">
                            <p className="home-stat-value">{total.toLocaleString()}</p>
                            <p className="home-stat-label">{t('home.totalItems')}</p>
                        </div>
                        <div className="home-stat-item">
                            <p className="home-stat-value">{totalDownloads >= 1000 ? `${(totalDownloads / 1000).toFixed(1)}K` : totalDownloads}</p>
                            <p className="home-stat-label">{t('home.downloads')}</p>
                        </div>
                        <button className="home-upload-cta" onClick={handleUpload}>
                            {isOverview ? t('home.uploadResource') : t('home.uploadAsset', { label: info.label })}
                        </button>
                    </div>
                </header>

                <div className="home-toolbar glass-card">
                    <label className="search-bar-container">
                        <span className="search-bar-icon">⌕</span>
                        <input
                            type="text"
                            className="search-bar"
                            placeholder={t('home.searchPlaceholder', { label: info.label })}
                            value={search}
                            onChange={e => {
                                setSearch(e.target.value)
                                setPage(1)
                            }}
                        />
                    </label>

                    <div className="home-filter-wrap" ref={filterRef}>
                        <button
                            className={`home-filter-btn ${filterOpen ? 'active' : ''} ${filterTag ? 'has-filter' : ''}`}
                            type="button"
                            onClick={() => setFilterOpen(prev => !prev)}
                        >
                            ⌁ {filterTag || t('home.filters')}
                        </button>
                        {filterOpen && (
                            <div className="home-filter-dropdown glass-card">
                                <button
                                    type="button"
                                    className={`home-filter-option ${!filterTag ? 'active' : ''}`}
                                    onClick={() => { setFilterTag(''); setFilterOpen(false); setPage(1) }}
                                >
                                    {t('home.filters')} (All)
                                </button>
                                {allTags.map(tag => (
                                    <button
                                        key={tag}
                                        type="button"
                                        className={`home-filter-option ${filterTag === tag ? 'active' : ''}`}
                                        onClick={() => { setFilterTag(tag); setFilterOpen(false); setPage(1) }}
                                    >
                                        {tag}
                                    </button>
                                ))}
                                {allTags.length === 0 && (
                                    <div className="home-filter-empty">No tags</div>
                                )}
                            </div>
                        )}
                    </div>
                </div>

                {loading ? (
                    <div className="empty-state">
                        <div className="loading-spinner" style={{ width: 40, height: 40 }} />
                        <p>{t('common.loadingResources')}</p>
                    </div>
                ) : filteredSkills.length === 0 ? (
                    <div className="empty-state">
                        <div className="icon">⌂</div>
                        <p>{t('home.noResources')}</p>
                    </div>
                ) : (
                    <>
                        <div className="skills-grid" key={animKey}>
                            {filteredSkills.map((skill, idx) => (
                                <div
                                    key={skill.id}
                                    className="skill-card-enter"
                                    style={{ animationDelay: `${idx * 60}ms` }}
                                >
                                    <SkillCard skill={skill} />
                                </div>
                            ))}
                        </div>

                        {totalPages > 1 && (
                            <div className="pagination">
                                <button
                                    className="btn btn-secondary btn-sm"
                                    disabled={page <= 1}
                                    onClick={() => setPage(prev => prev - 1)}
                                >
                                    {t('common.prev')}
                                </button>
                                <span className="pagination-info">{page} / {totalPages}</span>
                                <button
                                    className="btn btn-secondary btn-sm"
                                    disabled={page >= totalPages}
                                    onClick={() => setPage(prev => prev + 1)}
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        )}
                    </>
                )}
            </section>

            <RightSidebar resourceType={resourceTypeFilter} />
            <LoginModal isOpen={loginOpen} onClose={() => setLoginOpen(false)} />
        </div>
    )
}

export default HomePage
