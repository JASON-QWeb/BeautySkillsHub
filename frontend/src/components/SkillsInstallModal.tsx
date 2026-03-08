import { useEffect, useMemo, useState } from 'react'
import { fetchSkillInstallConfig } from '../services/api'
import { isAbortError } from '../services/api/request'
import { AIChatCharacter } from './AIChatCharacter'

interface SkillsInstallModalProps {
    isOpen: boolean
    onClose: () => void
}

interface InstallConfig {
    github_repo: string
    github_base_dir: string
}

const FALLBACK_CONFIG: InstallConfig = {
    github_repo: 'https://github.com/JASON-QWeb/agent-skills',
    github_base_dir: 'skills',
}

function normalizeInstallFlag(baseDir: string) {
    const normalized = baseDir
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9_-]+/g, '')
    return normalized || 'skills'
}

export function SkillsInstallModal({ isOpen, onClose }: SkillsInstallModalProps) {
    const [installConfig, setInstallConfig] = useState<InstallConfig>(FALLBACK_CONFIG)
    const [copiedKey, setCopiedKey] = useState('')

    useEffect(() => {
        if (!isOpen) return

        const controller = new AbortController()
        void fetchSkillInstallConfig({ signal: controller.signal })
            .then(config => {
                setInstallConfig({
                    github_repo: config.github_repo || FALLBACK_CONFIG.github_repo,
                    github_base_dir: config.github_base_dir || FALLBACK_CONFIG.github_base_dir,
                })
            })
            .catch(err => {
                if (isAbortError(err)) return
                setInstallConfig(FALLBACK_CONFIG)
            })

        return () => {
            controller.abort()
        }
    }, [isOpen])

    const installFlag = useMemo(() => normalizeInstallFlag(installConfig.github_base_dir), [installConfig.github_base_dir])
    const installCommand = useMemo(
        () => `npx skills add ${installConfig.github_repo} --${installFlag} ag_grid`,
        [installConfig.github_repo, installFlag],
    )
    const checkCommand = 'npx skills check'
    const updateCommand = 'npx skills update'

    const copyCommand = async (value: string, key: string) => {
        try {
            await navigator.clipboard.writeText(value)
            setCopiedKey(key)
            window.setTimeout(() => setCopiedKey(''), 1200)
        } catch {
            setCopiedKey('')
        }
    }

    if (!isOpen) return null

    return (
        <div className="skills-install-modal-overlay" onClick={onClose}>
            <div className="skills-install-modal-box" onClick={e => e.stopPropagation()}>
                <button className="skills-install-close" onClick={onClose} aria-label="Close">
                    ✕
                </button>

                <AIChatCharacter
                    className="modal-corner-character"
                    isOpen={false}
                    isTyping={false}
                />

                <div className="skills-install-modal-content">
                    <header className="skills-install-header">
                        <h2>Skills 安装教程</h2>
                        <p>先装 Node.js，再执行安装命令，最后掌握更新检查。</p>
                    </header>

                    <section className="skills-install-step">
                        <h3>1. 首先安装 Node.js（任选其一）</h3>
                        <div className="skills-install-command-grid">
                            <div className="skills-install-command-card">
                                <span>macOS</span>
                                <code>brew install node</code>
                            </div>
                            <div className="skills-install-command-card">
                                <span>Windows</span>
                                <code>winget install OpenJS.NodeJS.LTS</code>
                            </div>
                            <div className="skills-install-command-card">
                                <span>Ubuntu/Debian</span>
                                <code>sudo apt-get install -y nodejs npm</code>
                            </div>
                        </div>
                        <p className="skills-install-tip">安装后可用 <code>node -v && npm -v</code> 验证。</p>
                    </section>

                    <section className="skills-install-step">
                        <h3>2. 然后运行安装命令</h3>
                        <div className="skills-install-command-row">
                            <code>{installCommand}</code>
                            <button
                                type="button"
                                onClick={() => void copyCommand(installCommand, 'install')}
                            >
                                {copiedKey === 'install' ? '已复制' : '复制'}
                            </button>
                        </div>
                        <p className="skills-install-tip">
                            当前后端配置映射：<code>GITHUB_REPO</code> ={' '}
                            <a href={installConfig.github_repo} target="_blank" rel="noreferrer">
                                {installConfig.github_repo}
                            </a>
                            ，<code>GITHUB_BASE_DIR</code> = <code>{installConfig.github_base_dir}</code>（命令参数即 <code>--{installFlag}</code>）。
                        </p>
                    </section>

                    <section className="skills-install-step">
                        <h3>3. 更新与维护已安装的技能</h3>
                        <p className="skills-install-tip" style={{ marginTop: 0, marginBottom: 12 }}>
                            技能安装后，你可以通过以下命令随时保持它们处于最新状态。
                            <span style={{ color: 'var(--accent-secondary)', fontWeight: 500, display: 'block', marginTop: '4px' }}>
                                ⚠️ 注意：check 和 update 目前仅支持全局安装的技能库，暂不支持针对项目内部局部安装的技能进行版本管理。
                            </span>
                        </p>
                        
                        <div className="skills-install-command-row">
                            <code>{checkCommand}</code>
                            <button
                                type="button"
                                onClick={() => void copyCommand(checkCommand, 'check')}
                            >
                                {copiedKey === 'check' ? '已复制' : '复制'}
                            </button>
                        </div>
                        <p className="skills-install-tip" style={{ marginTop: 6, marginBottom: 16, fontSize: '0.85rem' }}>
                            <strong>只读检查：</strong>扫描全局已安装技能，预览哪些有新版本可供更新，不会改动任何文件。
                        </p>

                        <div className="skills-install-command-row">
                            <code>{updateCommand}</code>
                            <button
                                type="button"
                                onClick={() => void copyCommand(updateCommand, 'update')}
                            >
                                {copiedKey === 'update' ? '已复制' : '复制'}
                            </button>
                        </div>
                        <p className="skills-install-tip" style={{ marginTop: 6, marginBottom: 0, fontSize: '0.85rem' }}>
                            <strong>执行更新：</strong>将全局技能拉取并替换为最新版本，并自动同步版本记录。
                        </p>
                        
                        <div style={{ marginTop: 16, padding: '10px 14px', background: 'var(--accent-glow)', borderRadius: '10px', fontSize: '0.88rem', color: 'var(--text-primary)', border: '1px solid var(--border)' }}>
                            💡 <strong>推荐流程：</strong>先执行 <code>check</code> 预览可更新项，确认无误后再执行 <code>update</code> 完成批量升级。
                        </div>
                    </section>

                    <section className="skills-install-step" style={{ borderTop: '1px dashed var(--border)', paddingTop: '20px', marginTop: '20px' }}>
                        <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
                            <div>
                                <h4 style={{ margin: '0 0 8px 0', fontSize: '0.95rem', color: 'var(--accent-primary)' }}>Cline 用户小贴士</h4>
                                <p style={{ margin: 0, fontSize: '0.85rem', lineHeight: 1.5, opacity: 0.9 }}>
                                    如果你想让这些技能在 <strong>Cline</strong> 中生效，需要将它们移动到 Cline 的专用技能目录下。请在终端执行以下指令：
                                </p>
                                <div className="skills-install-command-row" style={{ marginTop: '10px', background: 'rgba(0,0,0,0.2)' }}>
                                    <code style={{ fontSize: '0.8rem' }}>mkdir -p .cline/skills && mv .agents/skills/* .cline/skills/</code>
                                    <button
                                        type="button"
                                        style={{ padding: '2px 8px', fontSize: '0.75rem' }}
                                        onClick={() => void copyCommand('mkdir -p .cline/skills && mv .agents/skills/* .cline/skills/', 'cline')}
                                    >
                                        {copiedKey === 'cline' ? '已复制' : '复制'}
                                    </button>
                                </div>
                            </div>
                        </div>
                    </section>
                </div>
            </div>
        </div>
    )
}
