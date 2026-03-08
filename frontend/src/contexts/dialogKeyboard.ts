export type DialogKeydownType = 'alert' | 'confirm'

export function getDialogEscapeResult(key: string, type: DialogKeydownType): boolean | null {
    if (key !== 'Escape') return null
    return type === 'confirm' ? false : true
}
