# npx skills 常用指令

## 基础

```bash
npx skills
```
显示可用命令帮助。

```bash
npx skills add <pkg>
npx skills a <pkg>
```
安装技能（支持 GitHub 仓库、URL、本地路径）。

```bash
npx skills add <pkg> --list
```
只查看该来源可安装的技能，不执行安装。

```bash
npx skills add <pkg> --skill <name>
```
只安装指定技能（可多次传 `--skill`）。

```bash
npx skills add <pkg> --all
```
安装来源中的全部技能。

## 查看与检索

```bash
npx skills list
npx skills ls
```
列出已安装技能。

```bash
npx skills find [query]
```
搜索技能（可交互或关键词模式）。

## 更新相关

```bash
npx skills check
```
检查哪些技能有可用更新。

```bash
npx skills update
```
更新所有已安装技能到最新版本。

```bash
npx skills install
npx skills i
npx skills experimental_install
```
从 `skills-lock.json` 恢复安装。

```bash
npx skills experimental_sync
```
从 `node_modules` 同步技能到 agent 目录。

## 初始化

```bash
npx skills init [name]
```
创建新的 `SKILL.md` 模板。

## 更新机制简述

- `check/update` 会读取锁文件中的 `source + skillFolderHash`。
- CLI 请求更新检查 API（`/check-updates`，带 `forceRefresh: true`）对比远端最新哈希。
- 如果哈希变化，`update` 会重新安装该技能并更新锁文件。
- 这通常等价于“按来源重新拉取最新版并替换本地技能”，不是单纯本地 `git pull`。

### 常用命令

| 命令 | 全局 Skill (`-g`) | 项目本地 Skill |
| :--- | :---: | :---: |
| `npx skills add` | ✅ 支持 | ✅ 支持 |
| `npx skills check` | ✅ **检测远程更新** | ❌ 不支持 |
| `npx skills update` | ✅ **一键升级** | ❌ 不支持 |