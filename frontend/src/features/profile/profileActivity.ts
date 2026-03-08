type ProfileActivityKind = 'published' | 'reviewed' | 'approved'

type Translate = (key: string, values?: Record<string, string | number>) => string

const PROFILE_ACTIVITY_ACTION_KEYS: Record<ProfileActivityKind, string> = {
    published: 'profile.publishedResource',
    reviewed: 'profile.reviewedResource',
    approved: 'profile.approvedResource',
}

const PROFILE_ACTIVITY_TYPE_KEYS: Record<string, string> = {
    skill: 'profile.activityTypeSkill',
    mcp: 'profile.activityTypeMcp',
    rules: 'profile.activityTypeRule',
    tools: 'profile.activityTypeTool',
}

function resolveProfileActivityTypeLabel(t: Translate, resourceType?: string): string {
    const normalizedType = (resourceType || '').trim().toLowerCase()
    const translationKey = PROFILE_ACTIVITY_TYPE_KEYS[normalizedType] || 'profile.activityTypeResource'
    return t(translationKey)
}

export function buildProfileActivityAction(
    t: Translate,
    kind: ProfileActivityKind,
    resourceType?: string,
): string {
    return t(PROFILE_ACTIVITY_ACTION_KEYS[kind], {
        type: resolveProfileActivityTypeLabel(t, resourceType),
    })
}
