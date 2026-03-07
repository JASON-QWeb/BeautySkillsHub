import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Language } from '../i18n'
import { useI18n } from '../i18n/I18nProvider'
import { Skill, fetchTrending, fetchSkills } from '../services/api'

interface RightSidebarProps {
    resourceType?: string
}

type CompositionCounts = {
    skill: number
    mcp: number
    tools: number
    rules: number
}

const defaultComposition: CompositionCounts = {
    skill: 0,
    mcp: 0,
    tools: 0,
    rules: 0,
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
    const [composition, setComposition] = useState<CompositionCounts>(defaultComposition)

    useEffect(() => {
        const load = async () => {
            try {
                const [trendingData, recentData] = await Promise.all([
                    fetchTrending(10, resourceType),
                    fetchSkills('', 1, 5, '', resourceType),
                ])
                setTrending(trendingData || [])
                setRecent(recentData.skills || [])
            } catch (err) {
                console.error('Failed to load sidebar data', err)
                setTrending([])
                setRecent([])
            }
        }

        load()
    }, [resourceType])

    useEffect(() => {
        const loadComposition = async () => {
            try {
                const [skillRes, mcpRes, toolsRes, rulesRes] = await Promise.all([
                    fetchSkills('', 1, 1, '', 'skill'),
                    fetchSkills('', 1, 1, '', 'mcp'),
                    fetchSkills('', 1, 1, '', 'tools'),
                    fetchSkills('', 1, 1, '', 'rules'),
                ])

                setComposition({
                    skill: skillRes.total || 0,
                    mcp: mcpRes.total || 0,
                    tools: toolsRes.total || 0,
                    rules: rulesRes.total || 0,
                })
            } catch (err) {
                console.error('Failed to load composition data', err)
                setComposition(defaultComposition)
            }
        }

        loadComposition()
    }, [])

    const totalComposition = useMemo(() => {
        return composition.skill + composition.mcp + composition.tools + composition.rules
    }, [composition])

    const compositionStops = useMemo(() => {
        const total = totalComposition || 1
        const skillPct = (composition.skill / total) * 100
        const mcpPct = (composition.mcp / total) * 100
        const rulesPct = (composition.rules / total) * 100
        const toolsPct = (composition.tools / total) * 100

        const s1 = skillPct
        const s2 = s1 + mcpPct
        const s3 = s2 + rulesPct
        const s4 = s3 + toolsPct
        return { s1, s2, s3, s4 }
    }, [composition, totalComposition])

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
                            onClick={() => navigate(`/skill/${skill.id}`, { state: { resourceType: skill.resource_type || resourceType } })}
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
                            onClick={() => navigate(`/skill/${skill.id}`, { state: { resourceType: skill.resource_type || resourceType } })}
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
                    <p className="comp-value">{totalComposition.toLocaleString()}</p>
                    <p className="comp-note">{t('sidebar.skillsVsTools')}</p>
                    <div className="comp-breakdown">
                        <div className="comp-breakdown-item">
                            <span>{t('nav.skills')}</span>
                            <strong>{composition.skill}</strong>
                        </div>
                        <div className="comp-breakdown-item">
                            <span>{t('nav.mcp')}</span>
                            <strong>{composition.mcp}</strong>
                        </div>
                        <div className="comp-breakdown-item">
                            <span>{t('nav.tools')}</span>
                            <strong>{composition.tools}</strong>
                        </div>
                        <div className="comp-breakdown-item">
                            <span>{t('nav.rules')}</span>
                            <strong>{composition.rules}</strong>
                        </div>
                    </div>
                </div>
                <div
                    className="comp-ring multi"
                    style={{
                        background: `conic-gradient(
                          #4f83e8 0% ${compositionStops.s1}%,
                          #35d388 ${compositionStops.s1}% ${compositionStops.s2}%,
                          #f3c614 ${compositionStops.s2}% ${compositionStops.s3}%,
                          #7f98dd ${compositionStops.s3}% ${compositionStops.s4}%
                        )`,
                    }}
                >
                    <span>Hub</span>
                </div>
            </section>
        </aside>
    )
}

export default RightSidebar
