const API_BASE = '/api';

export interface Skill {
    id: number;
    name: string;
    description: string;
    category: string;
    resource_type: string;
    author: string;
    file_name: string;
    file_size: number;
    thumbnail_url: string;
    downloads: number;
    ai_approved: boolean;
    ai_feedback: string;
    created_at: string;
    updated_at: string;
}

export interface SkillListResponse {
    skills: Skill[];
    total: number;
    page: number;
    page_size: number;
}

export interface UploadResponse {
    skill: Skill;
    approved: boolean;
    feedback: string;
}

// Resource type labels
export const RESOURCE_TYPES: Record<string, { label: string; icon: string; desc: string }> = {
    skill: { label: 'Skill', icon: '⚡', desc: '自动化技能脚本' },
    mcp: { label: 'MCP', icon: '🔌', desc: 'Model Context Protocol 服务' },
    rules: { label: 'Rules', icon: '📏', desc: '规则与约束配置' },
    tools: { label: 'Tools', icon: '🛠️', desc: '开发与运维工具' },
};

// Fetch all skills with optional search, category, resource_type and pagination
export async function fetchSkills(
    search = '', page = 1, pageSize = 20,
    category = '', resourceType = ''
): Promise<SkillListResponse> {
    const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
    });
    if (search) params.set('search', search);
    if (category) params.set('category', category);
    if (resourceType) params.set('resource_type', resourceType);

    const res = await fetch(`${API_BASE}/skills?${params}`);
    if (!res.ok) throw new Error('获取列表失败');
    return res.json();
}

// Fetch categories for a resource type
export async function fetchCategories(resourceType = ''): Promise<string[]> {
    const params = new URLSearchParams();
    if (resourceType) params.set('resource_type', resourceType);
    const res = await fetch(`${API_BASE}/categories?${params}`);
    if (!res.ok) throw new Error('获取类别失败');
    return res.json();
}

// Fetch a single skill by ID
export async function fetchSkill(id: number): Promise<Skill> {
    const res = await fetch(`${API_BASE}/skills/${id}`);
    if (!res.ok) throw new Error('获取详情失败');
    return res.json();
}

// Fetch trending skills
export async function fetchTrending(limit = 10, resourceType = ''): Promise<Skill[]> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (resourceType) params.set('resource_type', resourceType);
    const res = await fetch(`${API_BASE}/skills/trending?${params}`);
    if (!res.ok) throw new Error('获取趋势榜单失败');
    return res.json();
}

// Upload a new skill
export async function uploadSkill(formData: FormData): Promise<UploadResponse> {
    const res = await fetch(`${API_BASE}/skills`, {
        method: 'POST',
        body: formData,
    });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || '上传失败');
    }
    return res.json();
}

// Delete a skill (also deletes from GitHub)
export async function deleteSkill(id: number): Promise<{ message: string; github_error?: string }> {
    const res = await fetch(`${API_BASE}/skills/${id}`, { method: 'DELETE' });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Delete failed');
    }
    return res.json();
}

// Get download URL for a skill
export function getDownloadUrl(id: number): string {
    return `${API_BASE}/skills/${id}/download`;
}

// Chat with AI using SSE streaming
export async function chatWithAI(
    message: string,
    onChunk: (text: string) => void,
    onDone: () => void,
    onError: (error: string) => void
): Promise<void> {
    try {
        const res = await fetch(`${API_BASE}/ai/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message }),
        });

        if (!res.ok) {
            const err = await res.json();
            onError(err.error || 'AI 请求失败');
            return;
        }

        const reader = res.body?.getReader();
        if (!reader) {
            onError('无法读取响应流');
            return;
        }

        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';

            for (const line of lines) {
                const trimmed = line.trim();
                if (!trimmed) continue;

                if (trimmed.startsWith('data:')) {
                    const data = trimmed.slice(5).trim();
                    if (!data) continue;

                    if (data === '[DONE]') {
                        onDone();
                        return;
                    }

                    try {
                        const parsed = JSON.parse(data);
                        if (typeof parsed === 'string') {
                            onChunk(parsed);
                        }
                    } catch {
                        onChunk(data);
                    }
                } else if (trimmed.startsWith('event:')) {
                    continue;
                }
            }
        }

        onDone();
    } catch (err) {
        onError(err instanceof Error ? err.message : '网络错误');
    }
}
