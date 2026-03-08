import { useEffect, useMemo, useState } from 'react'
import SkillCard from '../../components/SkillCard'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { ProfileActivity, Skill, fetchMyFavorites, fetchMyUploads } from '../../services/api'
import { buildProfileActivityAction } from './profileActivity'

type TabKey = 'uploads' | 'saved' | 'activity'

function hashText(input: string): number {
    let hash = 0
    for (let i = 0; i < input.length; i += 1) {
        hash = (hash * 31 + input.charCodeAt(i)) >>> 0
    }
    return hash
}

function ProfilePage() {
    const { user } = useAuth()
    const { language, t } = useI18n()
    const [skills, setSkills] = useState<Skill[]>([])
    const [favoriteSkills, setFavoriteSkills] = useState<Skill[]>([])
    const [activities, setActivities] = useState<ProfileActivity[]>([])
    const [topTags, setTopTags] = useState<string[]>([])
    const [stats, setStats] = useState({ total_items: 0, total_downloads: 0, total_likes: 0 })
    const [totalUploads, setTotalUploads] = useState(0)
    const [page, setPage] = useState(1)
    const [loading, setLoading] = useState(true)
    const [favoritesLoading, setFavoritesLoading] = useState(true)
    const [activeTab, setActiveTab] = useState<TabKey>('uploads')
    const [search, setSearch] = useState('')
    const [typeFilter, setTypeFilter] = useState('all')

    const pageSize = 20

    useEffect(() => {
        const controller = new AbortController()

        const loadFavorites = async () => {
            setFavoritesLoading(true)
            try {
                const favoritesData = await fetchMyFavorites('', { signal: controller.signal })
                setFavoriteSkills(favoritesData || [])
            } catch (err) {
                if ((err as Error).name === 'AbortError') return
                console.error('Failed to load profile favorites:', err)
                setFavoriteSkills([])
            } finally {
                if (!controller.signal.aborted) {
                    setFavoritesLoading(false)
                }
            }
        }

        void loadFavorites()
        return () => controller.abort()
    }, [])

    useEffect(() => {
        const controller = new AbortController()

        const loadUploads = async () => {
            setLoading(true)
            try {
                const resourceType = typeFilter === 'all' ? '' : typeFilter
                const uploadsData = await fetchMyUploads(search, page, pageSize, resourceType, { signal: controller.signal })
                setSkills(uploadsData.skills || [])
                setTotalUploads(uploadsData.total || 0)
                setStats(uploadsData.stats || { total_items: 0, total_downloads: 0, total_likes: 0 })
                setTopTags(uploadsData.top_tags || [])
                setActivities(uploadsData.activities || [])
            } catch (err) {
                if ((err as Error).name === 'AbortError') return
                console.error('Failed to load profile uploads:', err)
                setSkills([])
                setTotalUploads(0)
                setStats({ total_items: 0, total_downloads: 0, total_likes: 0 })
                setTopTags([])
                setActivities([])
            } finally {
                if (!controller.signal.aborted) {
                    setLoading(false)
                }
            }
        }

        void loadUploads()
        return () => controller.abort()
    }, [page, search, typeFilter])

    useEffect(() => {
        setPage(1)
    }, [search, typeFilter])

    const profileBio = useMemo(() => {
        const options = [
            t('profile.bioOption1'),
            t('profile.bioOption2'),
            t('profile.bioOption3'),
            t('profile.bioOption4'),
            t('profile.bioOption5'),
            t('profile.bioOption6'),
        ].filter(Boolean)

        if (options.length === 0) return t('profile.bio')

        const seed = user ? `${user.id}-${user.username}` : 'guest'
        const index = hashText(seed) % options.length
        return options[index]
    }, [t, user])

    const totalDownloads = useMemo(() => stats.total_downloads, [stats.total_downloads])
    const totalLikes = useMemo(() => stats.total_likes, [stats.total_likes])
    const totalFavorites = favoriteSkills.length
    const totalPages = Math.max(1, Math.ceil(totalUploads / pageSize))

    const visibleSaved = useMemo(() => {
        const keyword = search.trim().toLowerCase()
        return favoriteSkills
            .filter(skill => {
                const matchesKeyword = !keyword
                    || skill.name.toLowerCase().includes(keyword)
                    || (skill.description || '').toLowerCase().includes(keyword)
                    || (skill.tags || '').toLowerCase().includes(keyword)
                const matchesType = typeFilter === 'all' || skill.resource_type === typeFilter
                return matchesKeyword && matchesType
            })
            .sort((a, b) => {
                const downloadDelta = (b.downloads || 0) - (a.downloads || 0)
                if (downloadDelta !== 0) return downloadDelta
                return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
            })
    }, [favoriteSkills, search, typeFilter])

    const activityItems = useMemo(() => {
        return activities.map((item, index) => ({
            id: `${item.kind}-${item.target}-${index}`,
            action: buildProfileActivityAction(t, item.kind, item.resource_type),
            target: item.target,
            time: new Date(item.occurred_at).toLocaleDateString(language === 'zh' ? 'zh-CN' : 'en-US'),
        }))
    }, [activities, language, t])

    const shouldShowResourcePanel = activeTab === 'uploads' || activeTab === 'saved'
    const activeSkills = activeTab === 'saved' ? visibleSaved : skills
    const activeLoading = activeTab === 'saved' ? favoritesLoading : loading

    const handleFavoriteChange = (skillID: number, favorited: boolean) => {
        setSkills(prev => prev.map(skill => (
            skill.id === skillID ? { ...skill, favorited } : skill
        )))

        setFavoriteSkills(prev => {
            if (favorited) {
                if (prev.some(skill => skill.id === skillID)) {
                    return prev.map(skill => (
                        skill.id === skillID ? { ...skill, favorited: true } : skill
                    ))
                }

                const sourceSkill = skills.find(skill => skill.id === skillID)
                if (!sourceSkill) return prev
                return [{ ...sourceSkill, favorited: true }, ...prev]
            }

            return prev.filter(skill => skill.id !== skillID)
        })
    }

    return (
        <div className="profile-page page-enter">
            <aside className="profile-sidebar">
                <div className="profile-card glass-card">
                    <div className="profile-head">
                        {user?.avatar_url ? (
                            <img src={user.avatar_url} alt={user.username} className="profile-avatar" />
                        ) : (
                            <span className="profile-avatar">{(user?.username || 'A')[0].toUpperCase()}</span>
                        )}
                    </div>

                    <h1 className="profile-name">{user?.username || t('profile.defaultName')}</h1>
                    <p className="profile-handle">@{user?.username || t('profile.defaultHandle')}</p>
                    <p className="profile-bio">
                        {profileBio}
                    </p>

                    <div className="profile-badges">
                        {topTags.length > 0 ? topTags.map((tag, idx) => (
                            <span key={tag} className={`profile-badge ${idx === 0 ? 'highlight' : ''}`}>
                                {tag}
                            </span>
                        )) : (
                            <span className="profile-badge">{t('profile.noTopTags')}</span>
                        )}
                    </div>

                    <div className="profile-meta">
                        <span>📍 {t('profile.location')}</span>
                        <span>🗓 {t('profile.joined')}</span>
                        <span>🔗 {t('profile.link', { username: user?.username || 'alex' })}</span>
                    </div>
                </div>

                <div className="profile-stats-card solid-dark-card">
                    <h3>{t('profile.performance')}</h3>
                    <div className="profile-stats-grid">
                        <div>
                            <p>{t('home.downloads')}</p>
                            <strong>{totalDownloads >= 1000 ? `${(totalDownloads / 1000).toFixed(1)}K` : totalDownloads}</strong>
                        </div>
                        <div>
                            <p>{t('profile.reputation')}</p>
                            <strong className="accent">{totalLikes}</strong>
                        </div>
                        <div>
                            <p>{t('profile.items')}</p>
                            <strong>{stats.total_items}</strong>
                        </div>
                        <div>
                            <p>{t('profile.totalFavorites')}</p>
                            <strong className="success">{totalFavorites}</strong>
                        </div>
                    </div>
                </div>
            </aside>

            <section className="profile-workspace">
                <div className="profile-tabs glass-card">
                    <button className={`profile-tab ${activeTab === 'uploads' ? 'active' : ''}`} onClick={() => setActiveTab('uploads')}>
                        {t('profile.tabUploads')} <span>{stats.total_items}</span>
                    </button>
                    <button className={`profile-tab ${activeTab === 'saved' ? 'active' : ''}`} onClick={() => setActiveTab('saved')}>
                        {t('profile.tabSaved')} <span>{favoriteSkills.length}</span>
                    </button>
                    <button className={`profile-tab ${activeTab === 'activity' ? 'active' : ''}`} onClick={() => setActiveTab('activity')}>
                        {t('profile.tabActivity')}
                    </button>
                </div>

                {shouldShowResourcePanel && (
                    <>
                        <div className="profile-toolbar glass-card">
                            <label className="search-bar-container">
                                <span className="search-bar-icon">⌕</span>
                                <input
                                    className="search-bar"
                                    placeholder={t('profile.searchUploads')}
                                    value={search}
                                    onChange={e => setSearch(e.target.value)}
                                />
                            </label>

                            <select
                                className="profile-select"
                                value={typeFilter}
                                onChange={e => setTypeFilter(e.target.value)}
                            >
                                <option value="all">{t('home.allTypes')}</option>
                                <option value="skill">{t('nav.skills')}</option>
                                <option value="mcp">MCP</option>
                                <option value="rules">{t('nav.rules')}</option>
                                <option value="tools">{t('nav.tools')}</option>
                            </select>

                            <button className="profile-sort-btn" type="button" disabled>
                                ↧ {t('home.downloads')}
                            </button>
                        </div>

                        {activeLoading ? (
                            <div className="empty-state">
                                <div className="loading-spinner" style={{ width: 40, height: 40 }} />
                                <p>{t('profile.loading')}</p>
                            </div>
                        ) : activeSkills.length === 0 ? (
                            <div className="empty-state glass-card">
                                <div className="icon">⌽</div>
                                <p>{activeTab === 'saved' ? t('profile.noSaved') : t('profile.noUploads')}</p>
                            </div>
                        ) : (
                            <>
                                <div className="profile-uploads-grid">
                                    {activeSkills.map(skill => (
                                        <SkillCard key={skill.id} skill={skill} onFavoriteChange={handleFavoriteChange} />
                                    ))}
                                </div>

                                {activeTab === 'uploads' && totalPages > 1 && (
                                    <div className="home-pagination" style={{ marginTop: 24 }}>
                                        <button
                                            className="page-btn"
                                            type="button"
                                            disabled={page <= 1}
                                            onClick={() => setPage(prev => Math.max(1, prev - 1))}
                                        >
                                            {t('common.prev')}
                                        </button>
                                        <span className="page-indicator">{page} / {totalPages}</span>
                                        <button
                                            className="page-btn"
                                            type="button"
                                            disabled={page >= totalPages}
                                            onClick={() => setPage(prev => Math.min(totalPages, prev + 1))}
                                        >
                                            {t('common.next')}
                                        </button>
                                    </div>
                                )}
                            </>
                        )}
                    </>
                )}

                {activeTab === 'activity' && (
                    <div className="glass-card profile-activity-card">
                        <div className="profile-timeline">
                            {activityItems.length === 0 && <p>{t('profile.noActivity')}</p>}
                            {activityItems.map(item => (
                                <div className="profile-timeline-item" key={item.id}>
                                    <span className="profile-timeline-dot" />
                                    <strong>{item.action}</strong>
                                    <p>{item.target}</p>
                                    <small>{item.time}</small>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </section>
        </div>
    )
}

export default ProfilePage
