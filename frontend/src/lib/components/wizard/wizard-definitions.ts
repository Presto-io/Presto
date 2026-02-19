export type TriggerType =
  | 'on-mount'         // Show when element appears on page load (with delay)
  | 'on-idle'          // Show after user is idle for N seconds
  | 'on-action'        // Show when triggerAction(id) is called by another component
  | 'on-navigate';     // Show when user navigates to a specific route

export type HintPosition = 'top' | 'bottom' | 'left' | 'right';

export interface WizardPointDef {
  id: string;
  trigger: TriggerType;
  /** CSS selector for the anchor element */
  anchorSelector: string;
  /** Position of the hint relative to anchor */
  position: HintPosition;
  title: string;
  body: string;
  /** Optional keyboard shortcut display */
  shortcut?: string;
  /** For on-idle: seconds of inactivity before nudge */
  idleSeconds?: number;
  /** Route where this hint is relevant ('/' = main page) */
  route: string;
  /** Delay in ms after trigger condition before showing nudge */
  triggerDelay?: number;
}

export const WIZARD_POINTS: WizardPointDef[] = [
  // ── Tier 1: First-visit essentials (on-mount with staggered delays) ──

  {
    id: 'template-selector',
    trigger: 'on-mount',
    anchorSelector: '.template-select',
    position: 'bottom',
    title: '选择模板',
    body: '在这里切换文档模板，不同模板适用于不同类型的文档。',
    route: '/',
    triggerDelay: 1500,
  },
  {
    id: 'export-pdf',
    trigger: 'on-mount',
    anchorSelector: '.btn-export',
    position: 'bottom',
    title: '导出 PDF',
    body: '编辑完成后，点击此按钮将文档导出为 PDF 文件。',
    shortcut: '⌘E',
    route: '/',
    triggerDelay: 8000,
  },
  {
    id: 'default-document',
    trigger: 'on-mount',
    anchorSelector: '.preview-container',
    position: 'left',
    title: '模板示例',
    body: '右侧显示的是当前模板的示例文档。在左侧编辑器中输入你的 Markdown 内容，预览会实时更新。',
    route: '/',
    triggerDelay: 15000,
  },

  // ── Tier 2: Idle-triggered contextual hints ──

  {
    id: 'split-divider',
    trigger: 'on-idle',
    anchorSelector: '.divider',
    position: 'right',
    title: '可拖拽分隔栏',
    body: '拖动这条分隔线，可以自由调整编辑器和预览区域的宽度比例。',
    idleSeconds: 30,
    route: '/',
  },
  {
    id: 'preview-scroll-sync',
    trigger: 'on-idle',
    anchorSelector: '.preview-container',
    position: 'left',
    title: '滚动同步',
    body: '编辑器和预览区域的滚动是同步的，滚动一侧，另一侧会自动跟随。',
    idleSeconds: 45,
    route: '/',
  },
  {
    id: 'batch-mode',
    trigger: 'on-idle',
    anchorSelector: '.toolbar',
    position: 'bottom',
    title: '批量转换',
    body: '需要一次转换多个文件？访问设置页面了解批量转换功能。',
    idleSeconds: 120,
    route: '/',
  },

  // ── Tier 2: First-action triggers (activated by components) ──

  {
    id: 'editor-find-replace',
    trigger: 'on-action',
    anchorSelector: '.editor-container',
    position: 'top',
    title: '查找和替换',
    body: '在编辑器中使用快捷键可以快速查找或替换文本。',
    shortcut: '⌘F 查找 · ⌘H 替换',
    route: '/',
  },
  {
    id: 'editor-undo-redo',
    trigger: 'on-action',
    anchorSelector: '.editor-container',
    position: 'top',
    title: '撤销与重做',
    body: '编辑器支持完整的撤销和重做历史记录。',
    shortcut: '⌘Z 撤销 · ⌘⇧Z 重做',
    route: '/',
  },
  {
    id: 'import-export-shortcuts',
    trigger: 'on-action',
    anchorSelector: '.toolbar',
    position: 'bottom',
    title: '快捷键提示',
    body: '除了使用按钮，你也可以通过快捷键快速操作。',
    shortcut: '⌘O 打开 · ⌘E 导出',
    route: '/',
  },
  {
    id: 'image-path',
    trigger: 'on-action',
    anchorSelector: '.editor-container',
    position: 'top',
    title: '图片路径',
    body: '使用 ![描述](路径) 插入图片。通过 ⌘O 打开文件后路径相对于文件目录，直接编辑时请使用绝对路径。',
    route: '/',
  },

  // ── Navigate-triggered hints ──

  {
    id: 'settings-shortcut',
    trigger: 'on-navigate',
    anchorSelector: '.page-header',
    position: 'bottom',
    title: '快速打开设置',
    body: '下次可以直接用快捷键打开设置页面。',
    shortcut: '⌘,',
    route: '/settings',
    triggerDelay: 2000,
  },

  // ── State-change triggered (called explicitly) ──

  {
    id: 'community-template-toggle',
    trigger: 'on-action',
    anchorSelector: '.nav-divider',
    position: 'right',
    title: '模板管理已解锁',
    body: '侧栏新增了「模板管理」和「模板搜索」，点击即可浏览和安装第三方模板。',
    route: '/settings',
  },
];
