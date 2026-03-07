import { useCallback, useEffect, useMemo, useState } from 'react'
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
    const [overviewComposition, setOverviewComposition] = useState({
        skill: 0,
        rules: 0,
        mcp: 0,
        tools: 0,
    })
    const [loginOpen, setLoginOpen] = useState(false)
    const [animKey, setAnimKey] = useState(0)

    const pageSize = 20

    useEffect(() => {
        setSearch('')
        setPage(1)
        setAnimKey(prev => prev + 1)
    }, [normalizedType])

    useEffect(() => {
        const loadSummary = async () => {
            try {
                if (isOverview) {
                    const [allSummary, skillSummary, rulesSummary, mcpSummary, toolsSummary] = await Promise.all([
                        fetchSkillSummary(''),
                        fetchSkillSummary('skill'),
                        fetchSkillSummary('rules'),
                        fetchSkillSummary('mcp'),
                        fetchSkillSummary('tools'),
                    ])
                    setSummary({
                        total: allSummary.total || 0,
                        yesterday_new: allSummary.yesterday_new || 0,
                    })
                    setOverviewComposition({
                        skill: skillSummary.total || 0,
                        rules: rulesSummary.total || 0,
                        mcp: mcpSummary.total || 0,
                        tools: toolsSummary.total || 0,
                    })
                    return
                }

                const data = await fetchSkillSummary(resourceTypeFilter)
                setSummary({
                    total: data.total || 0,
                    yesterday_new: data.yesterday_new || 0,
                })
                setOverviewComposition({
                    skill: 0,
                    rules: 0,
                    mcp: 0,
                    tools: 0,
                })
            } catch (err) {
                console.error('Failed to load summary:', err)
                setSummary({ total: 0, yesterday_new: 0 })
                setOverviewComposition({
                    skill: 0,
                    rules: 0,
                    mcp: 0,
                    tools: 0,
                })
            }
        }

        loadSummary()
    }, [isOverview, resourceTypeFilter])

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

    const totalDownloads = useMemo(() => {
        return skills.reduce((sum, skill) => sum + (skill.downloads || 0), 0)
    }, [skills])

    const displayName = user?.username || t('home.visitor')

    const handleUpload = () => {
        if (user) {
            navigate(`/resource/${normalizedType}/upload`)
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

                    {isOverview ? (
                        <div className="home-stats home-stats-overview">
                            <div className="home-stat-item">
                                <p className="home-stat-value">{overviewComposition.skill.toLocaleString()}</p>
                                <p className="home-stat-label">{t('nav.skills')}</p>
                            </div>
                            <div className="home-stat-item">
                                <p className="home-stat-value">{overviewComposition.rules.toLocaleString()}</p>
                                <p className="home-stat-label">{t('nav.rules')}</p>
                            </div>
                            <div className="home-stat-item">
                                <p className="home-stat-value">{overviewComposition.mcp.toLocaleString()}</p>
                                <p className="home-stat-label">{t('nav.mcp')}</p>
                            </div>
                            <div className="home-stat-item">
                                <p className="home-stat-value">{overviewComposition.tools.toLocaleString()}</p>
                                <p className="home-stat-label">{t('nav.tools')}</p>
                            </div>
                        </div>
                    ) : (
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
                                {t('home.uploadAsset', { label: info.label })}
                            </button>
                        </div>
                    )}
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

                    <button className="home-filter-btn" type="button">
                        ⌁ {t('home.filters')}
                    </button>
                </div>

                {loading ? (
                    <div className="empty-state">
                        <div className="loading-spinner" style={{ width: 40, height: 40 }} />
                        <p>{t('common.loadingResources')}</p>
                    </div>
                ) : skills.length === 0 ? (
                    <div className="empty-state">
                        <div className="icon">⌂</div>
                        <p>{t('home.noResources')}</p>
                    </div>
                ) : (
                    <>
                        <div className="skills-grid" key={animKey}>
                            {skills.map((skill, idx) => (
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
