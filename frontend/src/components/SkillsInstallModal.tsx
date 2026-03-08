import { useEffect, useMemo, useState } from 'react'
import { fetchSkillInstallConfig } from '../services/api'
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

        let cancelled = false
        void fetchSkillInstallConfig()
            .then(config => {
                if (cancelled) return
                setInstallConfig({
                    github_repo: config.github_repo || FALLBACK_CONFIG.github_repo,
                    github_base_dir: config.github_base_dir || FALLBACK_CONFIG.github_base_dir,
                })
            })
            .catch(() => {
                if (cancelled) return
                setInstallConfig(FALLBACK_CONFIG)
            })

        return () => {
            cancelled = true
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
                            技能安装后，你可以通过以下命令随时保持它们处于最新状态：
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
                            <strong>只读检查：</strong>扫描所有已安装技能，列出哪些有新版本可供更新，但不会改动任何文件。
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
                            <strong>执行更新：</strong>将检测到有变化的技能拉取并替换为最新版本，并自动同步更新锁文件记录。
                        </p>
                        
                        <div style={{ marginTop: 16, padding: '10px 14px', background: 'var(--accent-glow)', borderRadius: '10px', fontSize: '0.88rem', color: 'var(--text-primary)', border: '1px solid var(--border)' }}>
                            💡 <strong>推荐流程：</strong>先执行 <code>check</code> 预览可更新项，确认无误后再执行 <code>update</code> 完成批量升级。
                        </div>
                    </section>
                </div>
            </div>
        </div>
    )
}
