import { useNavigate } from 'react-router-dom'
import { Skill } from '../services/api'

interface Props {
    skill: Skill
}

function SkillCard({ skill }: Props) {
    const navigate = useNavigate()

    const thumbnailSrc = skill.thumbnail_url || '/placeholder.png'

    return (
        <div
            className="skill-card glass-card"
            onClick={() => navigate(`/skill/${skill.id}`)}
            id={`skill-card-${skill.id}`}
        >
            <img
                className="skill-card-thumb"
                src={thumbnailSrc}
                alt={skill.name}
                onError={(e) => {
                    (e.target as HTMLImageElement).src =
                        'data:image/svg+xml,' +
                        encodeURIComponent(
                            `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="200" viewBox="0 0 300 200">
                <defs><linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" style="stop-color:#6c5ce7"/><stop offset="100%" style="stop-color:#a29bfe"/></linearGradient></defs>
                <rect fill="url(#g)" width="300" height="200"/>
                <text x="150" y="110" text-anchor="middle" fill="white" font-size="48" font-family="sans-serif">${skill.name.charAt(0).toUpperCase()}</text>
              </svg>`
                        )
                }}
            />
            <div className="skill-card-info">
                <div className="skill-card-name">{skill.name}</div>
                <div className="skill-card-meta">
                    {skill.category && <span className="tag">{skill.category}</span>}
                    <span className="skill-card-downloads">
                        📥 {skill.downloads}
                    </span>
                </div>
            </div>
        </div>
    )
}

export default SkillCard
