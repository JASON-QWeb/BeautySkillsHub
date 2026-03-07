function escapeHtml(str: string): string {
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
}

export function parseMarkdown(text: string): string {
    if (!text) return ''

    const codeBlocks: string[] = []
    let raw = text.replace(/```([\s\S]*?)```/g, (_match, code) => {
        codeBlocks.push(code)
        return `__CODE_BLOCK_${codeBlocks.length - 1}__`
    })

    const inlineCodes: string[] = []
    raw = raw.replace(/`([^`]+)`/g, (_match, code) => {
        inlineCodes.push(code)
        return `__INLINE_CODE_${inlineCodes.length - 1}__`
    })

    const images: { alt: string; url: string }[] = []
    raw = raw.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_match, alt, url) => {
        images.push({ alt, url })
        return `__IMAGE_${images.length - 1}__`
    })

    const links: { text: string; url: string }[] = []
    raw = raw.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, linkText, url) => {
        links.push({ text: linkText, url })
        return `__LINK_${links.length - 1}__`
    })

    let html = escapeHtml(raw)

    html = html.replace(/__INLINE_CODE_(\d+)__/g, (_match, index) => {
        return `<code>${escapeHtml(inlineCodes[Number(index)])}</code>`
    })

    html = html.replace(/__IMAGE_(\d+)__/g, (_match, index) => {
        const image = images[Number(index)]
        const isHttp = /^https?:\/\//i.test(image.url)
        const isAbsolutePath = image.url.startsWith('/')
        const safeUrl = (isHttp || isAbsolutePath) ? escapeHtml(image.url) : '#'
        return `<img src="${safeUrl}" alt="${escapeHtml(image.alt || 'image')}" />`
    })

    html = html.replace(/__LINK_(\d+)__/g, (_match, index) => {
        const link = links[Number(index)]
        const safeUrl = /^https?:\/\//i.test(link.url) ? escapeHtml(link.url) : '#'
        return `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer">${escapeHtml(link.text)}</a>`
    })

    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
    html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
    html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>')
    html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
    html = html.replace(/^\* (.+)$/gm, '<li>$1</li>')
    html = html.replace(/((?:<li>.*<\/li>\n?)+)/g, '<ul>$1</ul>')

    html = html.replace(/\n{2,}/g, '</p><p>')
    html = '<p>' + html + '</p>'
    html = html.replace(/<p>\s*<(h[23]|ul)/g, '<$1')
    html = html.replace(/<\/(h[23]|ul)>\s*<\/p>/g, '</$1>')
    html = html.replace(/<p>\s*<\/p>/g, '')

    html = html.replace(/__CODE_BLOCK_(\d+)__/g, (_match, index) => {
        return `<pre><code>${escapeHtml(codeBlocks[Number(index)])}</code></pre>`
    })

    return html
}

