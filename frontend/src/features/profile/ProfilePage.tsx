import { useEffect, useMemo, useState } from 'react'
import SkillCard from '../../components/SkillCard'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { Skill, fetchMyFavorites, fetchSkills } from '../../services/api'

type TabKey = 'uploads' | 'saved' | 'activity'

function normalizeAuthorKey(skill: Skill): string {
    if (skill.user_id && skill.user_id > 0) return `u:${skill.user_id}`
    return `a:${(skill.author || '').trim().toLowerCase()}`
}

function isOwnedByUser(skill: Skill, userID: number, username: string): boolean {
    if (skill.user_id && skill.user_id > 0) return skill.user_id === userID
    return (skill.author || '').trim().toLowerCase() === username.trim().toLowerCase()
}

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
    const [loading, setLoading] = useState(true)
    const [favoritesLoading, setFavoritesLoading] = useState(true)
    const [activeTab, setActiveTab] = useState<TabKey>('uploads')
    const [search, setSearch] = useState('')
    const [typeFilter, setTypeFilter] = useState('all')

    useEffect(() => {
        const load = async () => {
            setLoading(true)
            setFavoritesLoading(true)
            try {
                const [skillsData, favoritesData] = await Promise.all([
                    fetchSkills('', 1, 500),
                    fetchMyFavorites(),
                ])
                setSkills(skillsData.skills || [])
                setFavoriteSkills(favoritesData || [])
            } catch (err) {
                console.error('Failed to load profile data:', err)
                setSkills([])
                setFavoriteSkills([])
            } finally {
                setLoading(false)
                setFavoritesLoading(false)
            }
        }
        load()
    }, [])

    const authoredSkills = useMemo(() => {
        if (!user?.username || !user?.id) return []
        return skills.filter(skill => isOwnedByUser(skill, user.id, user.username))
    }, [skills, user])

    const topUploadedTags = useMemo(() => {
        const tagStats = new Map<string, { label: string; count: number; index: number }>()
        let index = 0

        for (const skill of authoredSkills) {
            const tags = (skill.tags || '')
                .split(',')
                .map(tag => tag.trim())
                .filter(Boolean)

            for (const tag of tags) {
                const key = tag.toLowerCase()
                const existing = tagStats.get(key)
                if (existing) {
                    existing.count += 1
                    continue
                }

                tagStats.set(key, { label: tag, count: 1, index })
                index += 1
            }
        }

        return Array.from(tagStats.values())
            .sort((a, b) => {
                if (b.count !== a.count) return b.count - a.count
                return a.index - b.index
            })
            .slice(0, 3)
            .map(item => item.label)
    }, [authoredSkills])

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

    const totalDownloads = useMemo(() => {
        return authoredSkills.reduce((sum, skill) => sum + (skill.downloads || 0), 0)
    }, [authoredSkills])

    const totalLikes = useMemo(() => {
        return authoredSkills.reduce((sum, skill) => sum + (skill.likes_count || 0), 0)
    }, [authoredSkills])

    const downloadRank = useMemo(() => {
        if (!user || skills.length === 0) return null

        const totalsByAuthor = new Map<string, number>()
        for (const skill of skills) {
            const key = normalizeAuthorKey(skill)
            if (!key || key === 'a:') continue
            totalsByAuthor.set(key, (totalsByAuthor.get(key) || 0) + (skill.downloads || 0))
        }

        let currentKey = `u:${user.id}`
        let currentTotal = totalsByAuthor.get(currentKey)
        if (currentTotal === undefined) {
            currentKey = `a:${user.username.toLowerCase()}`
            currentTotal = totalsByAuthor.get(currentKey)
        }
        if (currentTotal === undefined) return null

        let higherCount = 0
        for (const value of totalsByAuthor.values()) {
            if (value > currentTotal) higherCount += 1
        }
        return higherCount + 1
    }, [skills, user])

    const filterAndSortByDownloads = (source: Skill[]) => {
        const keyword = search.trim().toLowerCase()
        return source
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
    }

    const visibleUploads = useMemo(() => filterAndSortByDownloads(authoredSkills), [authoredSkills, search, typeFilter])
    const visibleSaved = useMemo(() => filterAndSortByDownloads(favoriteSkills), [favoriteSkills, search, typeFilter])

    const reviewedSkills = useMemo(() => {
        if (!user?.id) return []
        return skills.filter(skill => skill.human_reviewer_id === user.id)
    }, [skills, user])

    const activities = useMemo(() => {
        const uploads = authoredSkills.map(skill => ({
            id: `up-${skill.id}`,
            action: t('profile.publishedResource'),
            target: skill.name,
            timestamp: new Date(skill.created_at).getTime(),
            time: new Date(skill.created_at).toLocaleDateString(language === 'zh' ? 'zh-CN' : 'en-US'),
        }))

        const reviews = reviewedSkills.map(skill => ({
            id: `rev-${skill.id}`,
            action: skill.human_review_status === 'approved' ? t('profile.approvedResource') : t('profile.reviewedResource'),
            target: skill.name,
            timestamp: new Date(skill.human_reviewed_at || skill.updated_at).getTime(),
            time: new Date(skill.human_reviewed_at || skill.updated_at).toLocaleDateString(language === 'zh' ? 'zh-CN' : 'en-US'),
        }))

        return [...uploads, ...reviews]
            .sort((a, b) => b.timestamp - a.timestamp)
            .slice(0, 10)
    }, [authoredSkills, reviewedSkills, language, t])

    const shouldShowResourcePanel = activeTab === 'uploads' || activeTab === 'saved'
    const activeSkills = activeTab === 'saved' ? visibleSaved : visibleUploads
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
                        {topUploadedTags.length > 0 ? topUploadedTags.map((tag, idx) => (
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
                            <strong>{authoredSkills.length}</strong>
                        </div>
                        <div>
                            <p>{t('profile.globalRank')}</p>
                            <strong className="success">{downloadRank ? `#${downloadRank}` : '-'}</strong>
                        </div>
                    </div>
                </div>
            </aside>

            <section className="profile-workspace">
                <div className="profile-tabs glass-card">
                    <button className={`profile-tab ${activeTab === 'uploads' ? 'active' : ''}`} onClick={() => setActiveTab('uploads')}>
                        {t('profile.tabUploads')} <span>{authoredSkills.length}</span>
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
                            <div className="profile-uploads-grid">
                                {activeSkills.map(skill => (
                                    <SkillCard key={skill.id} skill={skill} onFavoriteChange={handleFavoriteChange} />
                                ))}
                            </div>
                        )}
                    </>
                )}

                {activeTab === 'activity' && (
                    <div className="glass-card profile-activity-card">
                        <div className="profile-timeline">
                            {activities.length === 0 && <p>{t('profile.noActivity')}</p>}
                            {activities.map(item => (
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
