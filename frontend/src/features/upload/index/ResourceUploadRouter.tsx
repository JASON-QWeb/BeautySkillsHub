import { Navigate, useParams } from 'react-router-dom'
import SkillUploadPage from '../skill/SkillUploadPage'
import RulesUploadPage from '../rules/RulesUploadPage'
import McpUploadPage from '../mcp/McpUploadPage'
import ToolsUploadPage from '../tools/ToolsUploadPage'

function ResourceUploadRouter() {
    const { type } = useParams<{ type: string }>()
    const normalizedType = (type || 'skill').toLowerCase()

    if (normalizedType === 'skill') return <SkillUploadPage />
    if (normalizedType === 'rules') return <RulesUploadPage />
    if (normalizedType === 'mcp') return <McpUploadPage />
    if (normalizedType === 'tools') return <ToolsUploadPage />

    return <Navigate to="/resource/skill/upload" replace />
}

export default ResourceUploadRouter

