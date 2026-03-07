export const RESOURCE_TYPES: Record<string, { label: string; icon: string; desc: string }> = {
    all: { label: 'All', icon: '◉', desc: '全部资源总览' },
    skill: { label: 'Skill', icon: '⚡', desc: '自动化技能脚本' },
    mcp: { label: 'MCP', icon: '🔌', desc: 'Model Context Protocol 服务' },
    rules: { label: 'Rules', icon: '📏', desc: '规则与约束配置' },
    tools: { label: 'Tools', icon: '🛠️', desc: '开发与运维工具' },
}
