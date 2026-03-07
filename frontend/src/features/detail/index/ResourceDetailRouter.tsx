import { Navigate, useParams } from 'react-router-dom'
import SkillDetailPage from '../skill/SkillDetailPage'
import RulesDetailPage from '../rules/RulesDetailPage'
import McpDetailPage from '../mcp/McpDetailPage'
import ToolsDetailPage from '../tools/ToolsDetailPage'

function ResourceDetailRouter() {
    const { type } = useParams<{ type: string }>()
    const normalizedType = (type || 'skill').toLowerCase()

    if (normalizedType === 'skill') return <SkillDetailPage />
    if (normalizedType === 'rules') return <RulesDetailPage />
    if (normalizedType === 'mcp') return <McpDetailPage />
    if (normalizedType === 'tools') return <ToolsDetailPage />

    return <Navigate to="/resource/skill" replace />
}

export default ResourceDetailRouter
