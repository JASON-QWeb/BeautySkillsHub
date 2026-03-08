import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Language } from '../i18n'
import { useI18n } from '../i18n/I18nProvider'
import { Skill, fetchTrending, fetchSkills } from '../services/api'

interface RightSidebarProps {
    resourceType?: string
}

const TAG_COLORS = ['#1f2a44', '#f3c614', '#8b8b8b', '#c4a24e']
const TAG_COLORS_DARK = ['#4f83e8', '#35d388', '#f3c614', '#7f98dd']

const RESOURCE_TYPE_LABELS: Record<string, string> = {
    skill: 'Skill',
    mcp: 'MCP',
    tool: 'Tool',
    rules: 'Rules',
}

function timeAgo(value: string, language: Language) {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
        return language === 'zh' ? '刚刚' : 'just now'
    }

    const diff = Date.now() - date.getTime()
    const minute = 60 * 1000
    const hour = 60 * minute
    const day = 24 * hour
    const formatter = new Intl.RelativeTimeFormat(language === 'zh' ? 'zh-CN' : 'en', { numeric: 'auto' })

    if (diff < minute) return formatter.format(0, 'second')
    if (diff < hour) return formatter.format(-Math.floor(diff / minute), 'minute')
    if (diff < day) return formatter.format(-Math.floor(diff / hour), 'hour')
    return formatter.format(-Math.floor(diff / day), 'day')
}

function RightSidebar({ resourceType = '' }: RightSidebarProps) {
    const navigate = useNavigate()
    const { language, t } = useI18n()
    const [trending, setTrending] = useState<Skill[]>([])
    const [recent, setRecent] = useState<Skill[]>([])
    const [tagCounts, setTagCounts] = useState<Record<string, number>>({})

    useEffect(() => {
        const controller = new AbortController()
        const load = async () => {
            try {
                const [trendingData, recentData] = await Promise.all([
                    fetchTrending(10, resourceType, { signal: controller.signal }),
                    fetchSkills('', 1, 5, '', resourceType, { signal: controller.signal }),
                ])
                setTrending(trendingData || [])
                setRecent(recentData.skills || [])
            } catch (err) {
                if ((err as Error).name === 'AbortError') return
                console.error('Failed to load sidebar data', err)
                setTrending([])
                setRecent([])
            }
        }

        void load()
        return () => controller.abort()
    }, [resourceType])

    const isOverview = !resourceType

    useEffect(() => {
        const controller = new AbortController()
        const loadComposition = async () => {
            try {
                const data = await fetchSkills('', 1, 200, '', isOverview ? '' : resourceType, { signal: controller.signal })
                const counts: Record<string, number> = {}
                for (const skill of data.skills || []) {
                    if (isOverview) {
                        // Overview: count by resource_type
                        const rt = skill.resource_type || 'skill'
                        counts[rt] = (counts[rt] || 0) + 1
                    } else {
                        // Category page: count by tags
                        if (skill.tags) {
                            for (const tag of skill.tags.split(',')) {
                                const normalized = tag.trim().toLowerCase()
                                if (normalized) {
                                    counts[normalized] = (counts[normalized] || 0) + 1
                                }
                            }
                        }
                    }
                }
                setTagCounts(counts)
            } catch (err) {
                if ((err as Error).name === 'AbortError') return
                console.error('Failed to load composition', err)
                setTagCounts({})
            }
        }

        void loadComposition()
        return () => controller.abort()
    }, [resourceType, isOverview])

    const topTags = useMemo(() => {
        return Object.entries(tagCounts)
            .sort((a, b) => b[1] - a[1])
            .slice(0, 4)
    }, [tagCounts])

    const totalTagCount = useMemo(() => {
        return Object.values(tagCounts).reduce((a, b) => a + b, 0)
    }, [tagCounts])

    const tagStops = useMemo(() => {
        const total = totalTagCount || 1
        const stops: number[] = []
        let cumulative = 0
        for (const [, count] of topTags) {
            cumulative += (count / total) * 100
            stops.push(cumulative)
        }
        // Fill remaining stops if < 4 tags
        while (stops.length < 4) stops.push(cumulative)
        return stops
    }, [topTags, totalTagCount])

    const isDark = document.documentElement.getAttribute('data-theme') === 'dark'
    const colors = isDark ? TAG_COLORS_DARK : TAG_COLORS

    return (
        <aside className="right-sidebar">
            <section className="sidebar-hot solid-dark-card">
                <header className="sidebar-hot-header">
                    <h3>{t('sidebar.hottest')}</h3>
                    <span className="sidebar-hot-icon">↗</span>
                </header>

                <div className="sidebar-hot-list">
                    {trending.slice(0, 10).map((skill, idx) => (
                        <button
                            key={skill.id}
                            className="sidebar-hot-item"
                            onClick={() => navigate(`/resource/${skill.resource_type || resourceType || 'skill'}/${skill.id}`, { state: { resourceType: skill.resource_type || resourceType } })}
                        >
                            <span className="sidebar-rank">{String(idx + 1).padStart(2, '0')}</span>
                            <span className="sidebar-hot-name">{skill.name}</span>
                            <span className="sidebar-hot-rate download">
                                ↧ {(skill.downloads || 0).toLocaleString()}
                            </span>
                        </button>
                    ))}
                    {trending.length === 0 && <p className="sidebar-empty">{t('sidebar.noTrending')}</p>}
                </div>
            </section>

            <section className="sidebar-recent glass-card">
                <header className="sidebar-section-header">
                    <h3>{t('sidebar.recentlyUploaded')}</h3>
                    <span className="sidebar-arrow">›</span>
                </header>

                <div className="sidebar-timeline">
                    {recent.slice(0, 5).map(skill => (
                        <button
                            key={skill.id}
                            className="sidebar-timeline-item"
                            onClick={() => navigate(`/resource/${skill.resource_type || resourceType || 'skill'}/${skill.id}`, { state: { resourceType: skill.resource_type || resourceType } })}
                        >
                            <span className="timeline-dot" />
                            <span className="timeline-card">
                                <strong>{skill.name}</strong>
                                <small>{timeAgo(skill.created_at, language)}</small>
                            </span>
                        </button>
                    ))}
                    {recent.length === 0 && <p className="sidebar-empty">{t('sidebar.noRecent')}</p>}
                </div>
            </section>

            <section className="sidebar-composition glass-card">
                <div>
                    <p className="comp-label">{t('sidebar.totalComposition')}</p>
                    <p className="comp-value">{totalTagCount.toLocaleString()}</p>
                    <div className="comp-breakdown">
                        {topTags.map(([tag, count], idx) => (
                            <div key={tag} className="comp-breakdown-item">
                                <span>
                                    <span className="comp-color-dot" style={{ background: colors[idx] }} />
                                    {isOverview ? RESOURCE_TYPE_LABELS[tag] || tag : tag}
                                </span>
                                <strong>{count}</strong>
                            </div>
                        ))}
                        {topTags.length === 0 && (
                            <div className="comp-breakdown-item">
                                <span>--</span>
                                <strong>0</strong>
                            </div>
                        )}
                    </div>
                </div>
                <div
                    className="comp-ring multi"
                    style={{
                        background: topTags.length > 0
                            ? `conic-gradient(
                              ${colors[0]} 0% ${tagStops[0]}%${topTags.length > 1 ? `,
                              ${colors[1]} ${tagStops[0]}% ${tagStops[1]}%` : ''}${topTags.length > 2 ? `,
                              ${colors[2]} ${tagStops[1]}% ${tagStops[2]}%` : ''}${topTags.length > 3 ? `,
                              ${colors[3]} ${tagStops[2]}% ${tagStops[3]}%` : ''}
                            )`
                            : 'var(--border)',
                    }}
                >
                    <span>Hub</span>
                </div>
            </section>
        </aside>
    )
}

export default RightSidebar
