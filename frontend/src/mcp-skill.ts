import skill from '../../skills/ezruantu-design/SKILL.md?raw'
import modeling from '../../skills/ezruantu-design/references/modeling.md?raw'
import workflows from '../../skills/ezruantu-design/references/mcp-workflows.md?raw'

export function buildSkillHandoff(config: string): string {
  const instructions = skill
    .replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n/, '')
    .replace(/\[([^\]]+)\]\(references\/[^)]+\)/g, '$1（完整内容见下文）')
    .trim()

  return [
    '请帮我接入 EZ软图，并按照以下完整 Skill 处理我接下来提出的软件设计图任务。',
    '## 接入与执行',
    '先检查当前客户端是否已经连接 EZ软图 MCP。已连接则直接使用；未连接时，按当前客户端支持的方式添加下面的 stdio MCP 配置。若需要用户在客户端启用、确认或重启，明确说明具体步骤。配置中的本机路径必须在实际运行 MCP 的电脑上有效。',
    '这些文字本身不会建立 MCP 连接。先确认工具可调用，再执行绘图任务；无法接入时说明原因，不假装已生成文件。',
    '以下已经包含完整工作指引及全部参考内容，无需查找外部 Skill 文件，也不必依赖客户端的 Skill 安装功能。若尚未提供具体项目或绘图需求，接入完成后询问需求，不自动创建示例图。',
    '## MCP 配置\n\n```json\n' + config + '\n```',
    instructions,
    modeling.trim(),
    workflows.trim(),
  ].join('\n\n') + '\n'
}
