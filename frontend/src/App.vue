<script lang="ts" setup>
import { ref, watch, computed, reactive, onMounted } from 'vue'
import {
  SelectWorkingDirectory,
  ListSubDirs,
  ListSubFiles,
  SelectFiles,
  SelectTargetDirectory,
  PreviewNames,
  ExecuteRename,
  LoadScripts,
  SaveScripts
} from '../wailsjs/go/main/App'
import { main } from '../wailsjs/go/models'
import { ElMessage } from 'element-plus'

// ============================================================
// 核心状态
// ============================================================
const workingDir = ref('')
const operationMode = ref<'dirs' | 'files'>('files')
const files = ref<main.FileItem[]>([])
const rules = ref<main.RenameRule[]>([])
const loading = ref(false)

// 左侧面板模式: "builtin" 内置规则 | "script" 自定义脚本
const panelMode = ref<'builtin' | 'script'>('builtin')

// 自定义脚本列表
interface ScriptItem { path: string; name: string; desc: string }
const scriptList = ref<ScriptItem[]>([])
const activeScriptIndex = ref(-1) // 当前选中的脚本索引

// ============================================================
// 工作目录 & 列表
// ============================================================
const selectWorkDir = async () => {
  const dir = await SelectWorkingDirectory()
  if (dir) {
    workingDir.value = dir
    rules.value = []
    await loadItems()
  }
}

const loadItems = async () => {
  if (!workingDir.value) return
  files.value = operationMode.value === 'dirs'
    ? (await ListSubDirs(workingDir.value) || [])
    : (await ListSubFiles(workingDir.value) || [])
}

watch(operationMode, () => { if (workingDir.value) loadItems() })

const handleAddFiles = async () => {
  const result = await SelectFiles()
  if (result) {
    const existingKeys = new Set(files.value.map(f => f.path + '/' + f.originalName))
    const newItems = result.filter(f => !existingKeys.has(f.path + '/' + f.originalName))
    files.value = [...files.value, ...newItems]
  }
}

const clearFiles = () => { files.value = [] }

// ============================================================
// 内置规则管理
// ============================================================
const addRule = (type: string) => {
  const newRule: main.RenameRule = { type, params: {} }
  switch (type) {
    case 'replace':   newRule.params = { find: '', replace: '' }; break
    case 'prefix':    newRule.params = { prefix: '' }; break
    case 'suffix':    newRule.params = { suffix: '' }; break
    case 'insert':    newRule.params = { pos: '1', text: '' }; break
    case 'numbering': newRule.params = { start: '1', digits: '2', position: 'start' }; break
    case 'regex':     newRule.params = { find: '', replace: '' }; break
    case 'to_folder': newRule.params = { mode: 'move', targetDir: '', keepExt: 'false' }; break
  }
  rules.value.push(newRule)
}

const removeRule = (index: number) => { rules.value.splice(index, 1) }

// ============================================================
// 自定义脚本管理
// ============================================================
const addScript = async () => {
  const picked = await SelectFiles()
  if (!picked || picked.length === 0) return
  for (const f of picked) {
    const fullPath = f.path + '\\' + f.originalName
    if (scriptList.value.some(s => s.path === fullPath)) continue
    scriptList.value.push({ path: fullPath, name: f.originalName, desc: '' })
  }
  if (activeScriptIndex.value < 0 && scriptList.value.length > 0) {
    activeScriptIndex.value = 0
  }
  persistScripts()
}

const removeScript = (index: number) => {
  scriptList.value.splice(index, 1)
  if (activeScriptIndex.value >= scriptList.value.length) {
    activeScriptIndex.value = scriptList.value.length - 1
  }
  persistScripts()
}

// 持久化：保存脚本列表
const persistScripts = () => {
  SaveScripts(scriptList.value as any)
}

// 监听脚本描述变化，自动保存（防抖）
let scriptSaveTimer: any = null
watch(scriptList, () => {
  if (scriptSaveTimer) clearTimeout(scriptSaveTimer)
  scriptSaveTimer = setTimeout(() => persistScripts(), 1000)
}, { deep: true })

// 启动时加载已保存的脚本
onMounted(async () => {
  const saved = await LoadScripts()
  if (saved && saved.length > 0) {
    scriptList.value = saved.map((s: any) => ({ path: s.path, name: s.name, desc: s.desc || '' }))
    activeScriptIndex.value = 0
  }
})

const selectScript = (index: number) => {
  activeScriptIndex.value = index
}

// ============================================================
// 构建最终的规则（根据模式）
// ============================================================
const effectiveRules = computed<main.RenameRule[]>(() => {
  if (panelMode.value === 'builtin') {
    return rules.value
  } else {
    // 脚本模式：构建一条 script 规则
    if (activeScriptIndex.value >= 0 && activeScriptIndex.value < scriptList.value.length) {
      return [{ type: 'script', params: { scriptPath: scriptList.value[activeScriptIndex.value].path } }]
    }
    return []
  }
})

// ============================================================
// 预览
// ============================================================
const updatePreview = async () => {
  if (files.value.length === 0) return
  try {
    const result = await PreviewNames(files.value, effectiveRules.value)
    const map = new Map(result.map(r => [r.id, r.newName]))
    files.value.forEach(f => {
      const newName = map.get(f.id)
      if (newName !== undefined) f.newName = newName
    })
  } catch (err) {
    console.error('预览更新失败', err)
  }
}

let timer: any = null
watch([rules, activeScriptIndex, panelMode], () => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => updatePreview(), 500)
}, { deep: true })

// ============================================================
// 排序
// ============================================================
const handleSortChange = ({ prop, order }: { prop: string; order: string | null }) => {
  if (!prop || !order) return
  files.value.sort((a: any, b: any) => {
    let va = a[prop], vb = b[prop]
    if (typeof va === 'string') {
      const cmp = va.localeCompare(vb, 'zh-CN')
      return order === 'ascending' ? cmp : -cmp
    }
    return order === 'ascending' ? va - vb : vb - va
  })
  if (effectiveRules.value.some(r => r.type === 'numbering')) updatePreview()
}

// ============================================================
// 执行
// ============================================================
const execute = async () => {
  if (files.value.length === 0) return
  loading.value = true
  try {
    const result = await ExecuteRename(files.value, effectiveRules.value)
    files.value = result
    const sc = result.filter(f => f.status === 'success').length
    const ec = result.filter(f => f.status === 'error').length
    ElMessage({ message: `完成！成功: ${sc}, 失败: ${ec}`, type: ec > 0 ? 'warning' : 'success' })
  } catch (err) {
    ElMessage.error('执行失败: ' + err)
  } finally {
    loading.value = false
  }
}

const pickTargetDir = async (rule: main.RenameRule) => {
  const dir = await SelectTargetDirectory()
  if (dir) rule.params.targetDir = dir
}

// ============================================================
// 工具
// ============================================================
const formatDate = (ms: number) => ms ? new Date(ms).toLocaleString('zh-CN') : '-'
const formatSize = (bytes: number) => {
  if (!bytes) return '-'
  const k = 1024, s = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + s[i]
}
const ruleLabel = (t: string) => ({
  prefix: '添加前缀', suffix: '添加后缀', insert: '指定位置插入',
  replace: '查找替换', regex: '正则替换', numbering: '自动编号', to_folder: '生成同名目录'
}[t] || t)

const canExecute = computed(() => {
  if (files.value.length === 0) return false
  if (panelMode.value === 'builtin') return rules.value.length > 0
  return activeScriptIndex.value >= 0
})
</script>

<template>
  <el-container class="layout-container">
    <!-- ===== 左侧面板 ===== -->
    <el-aside width="360px" class="sidebar">
      <!-- 模式切换 -->
      <div class="sidebar-header">
        <el-radio-group v-model="panelMode" size="default" style="width: 100%">
          <el-radio-button label="builtin" style="width: 50%">内置规则</el-radio-button>
          <el-radio-button label="script" style="width: 50%">自定义脚本</el-radio-button>
        </el-radio-group>
      </div>

      <!-- ====== 内置规则模式 ====== -->
      <template v-if="panelMode === 'builtin'">
        <div class="sub-header">
          <el-dropdown @command="addRule" style="width: 100%">
            <el-button type="primary" size="small" style="width: 100%">
              添加规则<el-icon class="el-icon--right"><Plus /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <li class="group-title">— 添加类 —</li>
                <el-dropdown-item command="prefix">添加前缀</el-dropdown-item>
                <el-dropdown-item command="suffix">添加后缀</el-dropdown-item>
                <el-dropdown-item command="insert">指定位置插入</el-dropdown-item>
                <li class="group-title">— 修改类 —</li>
                <el-dropdown-item command="replace">查找替换</el-dropdown-item>
                <el-dropdown-item command="regex">正则替换</el-dropdown-item>
                <li class="group-title">— 编号 —</li>
                <el-dropdown-item command="numbering">自动编号</el-dropdown-item>
                <li class="group-title">— 骚操作 —</li>
                <el-dropdown-item command="to_folder">根据文件名生成目录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <el-scrollbar class="rules-scroll">
          <div class="rules-list">
            <el-card v-for="(rule, index) in rules" :key="index" class="rule-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span class="rule-label">{{ ruleLabel(rule.type) }}</span>
                  <el-button type="danger" size="small" circle icon="Delete" @click="removeRule(index)" />
                </div>
              </template>

              <div v-if="rule.type === 'replace' || rule.type === 'regex'" class="rule-inputs">
                <el-input v-model="rule.params.find" :placeholder="rule.type === 'regex' ? '正则表达式' : '查找文本'" size="small">
                  <template #prepend>查找</template>
                </el-input>
                <el-input v-model="rule.params.replace" placeholder="替换为" size="small" class="mt-2">
                  <template #prepend>替换</template>
                </el-input>
              </div>

              <div v-if="rule.type === 'prefix'" class="rule-inputs">
                <el-input v-model="rule.params.prefix" placeholder="前缀内容" size="small"><template #prepend>前缀</template></el-input>
              </div>
              <div v-if="rule.type === 'suffix'" class="rule-inputs">
                <el-input v-model="rule.params.suffix" placeholder="后缀内容" size="small"><template #prepend>后缀</template></el-input>
              </div>

              <div v-if="rule.type === 'insert'" class="rule-inputs">
                <div class="flex-row">
                  <span class="p-text">第</span>
                  <el-input v-model="rule.params.pos" type="number" size="small" style="width: 70px; margin: 0 6px" />
                  <span class="p-text">个字符处插入</span>
                </div>
                <el-input v-model="rule.params.text" placeholder="要插入的文本" size="small" class="mt-2">
                  <template #prepend>文本</template>
                </el-input>
              </div>

              <div v-if="rule.type === 'numbering'" class="rule-inputs">
                <div class="flex-row mb-2">
                  <span class="p-text mr-1">起始</span>
                  <el-input v-model="rule.params.start" size="small" style="width: 70px" />
                  <span class="p-text ml-2 mr-1">位数</span>
                  <el-input v-model="rule.params.digits" size="small" style="width: 50px" />
                </div>
                <el-radio-group v-model="rule.params.position" size="small">
                  <el-radio-button label="start">放开头</el-radio-button>
                  <el-radio-button label="end">放末尾</el-radio-button>
                </el-radio-group>
              </div>

              <div v-if="rule.type === 'to_folder'" class="rule-inputs">
                <p class="rule-tips">根据文件名创建同名子目录，并将文件放入。</p>
                <div class="flex-row mt-2 mb-2">
                  <span class="p-text mr-1">目录名</span>
                  <el-select v-model="rule.params.keepExt" size="small" style="width: 140px">
                    <el-option label="去扩展名 (默认)" value="false" />
                    <el-option label="保留扩展名" value="true" />
                  </el-select>
                </div>
                <div class="flex-row mb-2">
                  <span class="p-text mr-1">模　式</span>
                  <el-select v-model="rule.params.mode" size="small" style="width: 120px">
                    <el-option label="移动文件" value="move" />
                    <el-option label="复制文件" value="copy" />
                    <el-option label="仅创建目录" value="create_only" />
                  </el-select>
                </div>
                <div class="flex-row">
                  <span class="p-text mr-1">目标位置</span>
                  <el-input v-model="rule.params.targetDir" placeholder="留空=当前目录" size="small" readonly style="flex: 1" />
                  <el-button size="small" class="ml-1" @click="pickTargetDir(rule)">选择</el-button>
                </div>
              </div>
            </el-card>

            <div v-if="rules.length === 0" class="empty-panel">
              <el-icon :size="36" color="#dcdfe6"><SetUp /></el-icon>
              <p>暂无规则</p>
              <p class="sub">点击「添加规则」开始配置</p>
            </div>
          </div>
        </el-scrollbar>
      </template>

      <!-- ====== 自定义脚本模式 ====== -->
      <template v-else>
        <div class="sub-header">
          <el-button type="primary" size="small" style="width: 100%" @click="addScript">
            <el-icon class="mr-1"><Plus /></el-icon>添加脚本文件 (.bat)
          </el-button>
        </div>

        <el-scrollbar class="rules-scroll">
          <div class="script-list">
            <div
              v-for="(s, i) in scriptList" :key="s.path"
              class="script-card"
              :class="{ 'script-active': activeScriptIndex === i }"
              @click="selectScript(i)"
            >
              <div class="script-header">
                <div class="script-name">
                  <el-icon color="#E6A23C"><Document /></el-icon>
                  <span>{{ s.name }}</span>
                </div>
                <el-button type="danger" size="small" circle icon="Delete" @click.stop="removeScript(i)" />
              </div>
              <el-input
                v-model="s.desc"
                type="textarea"
                :rows="2"
                placeholder="输入功能说明…"
                size="small"
                class="mt-1"
                @click.stop
              />
              <div class="script-path">{{ s.path }}</div>
            </div>

            <div v-if="scriptList.length === 0" class="empty-panel">
              <el-icon :size="36" color="#dcdfe6"><Files /></el-icon>
              <p>暂无脚本</p>
              <p class="sub">点击上方按钮添加 .bat 脚本文件</p>
              <p class="sub">脚本接收原文件名作为参数，输出新文件名</p>
            </div>
          </div>
        </el-scrollbar>
      </template>
    </el-aside>

    <!-- ===== 右侧工作区 ===== -->
    <el-main class="main-content">
      <div class="workspace">
        <div class="action-bar">
          <el-button type="primary" icon="FolderOpened" @click="selectWorkDir">
            {{ workingDir ? '更换目录' : '选择工作目录' }}
          </el-button>
          <el-tag v-if="workingDir" type="info" effect="plain" closable @close="workingDir = ''; files = []" class="dir-tag">
            {{ workingDir }}
          </el-tag>
          <template v-if="workingDir">
            <el-radio-group v-model="operationMode" size="small">
              <el-radio-button label="dirs">子文件夹名</el-radio-button>
              <el-radio-button label="files">子文件名</el-radio-button>
            </el-radio-group>
          </template>
          <el-button v-if="workingDir" type="info" plain icon="Plus" size="small" @click="handleAddFiles">手动添加</el-button>
          <el-button v-if="files.length > 0" type="info" plain icon="Delete" size="small" @click="clearFiles">清空</el-button>
          <div style="flex: 1"></div>
          <el-button type="success" size="large" icon="Check" :loading="loading" @click="execute" :disabled="!canExecute">
            执行
          </el-button>
        </div>

        <el-table :data="files" style="width: 100%" height="calc(100vh - 130px)" border stripe @sort-change="handleSortChange" empty-text="请先选择工作目录">
          <el-table-column width="50" label="#" align="center">
            <template #default="scope">
              <el-icon v-if="scope.row.status === 'pending'" color="#909399"><Clock /></el-icon>
              <el-icon v-else-if="scope.row.status === 'success'" color="#67C23A"><CircleCheckFilled /></el-icon>
              <el-icon v-else-if="scope.row.status === 'error'" color="#F56C6C"><CircleCloseFilled /></el-icon>
            </template>
          </el-table-column>
          <el-table-column label="原名称" min-width="180" sortable="custom" prop="originalName">
            <template #default="scope">
              <div class="name-cell">
                <el-icon v-if="scope.row.isDir" color="#E6A23C"><Folder /></el-icon>
                <el-icon v-else color="#409EFF"><Document /></el-icon>
                <span>{{ scope.row.originalName }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="→ 预览新名称" min-width="200">
            <template #default="scope">
              <span :class="{ 'name-changed': scope.row.originalName !== scope.row.newName, 'name-same': scope.row.originalName === scope.row.newName }">
                {{ scope.row.newName || scope.row.originalName }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="修改时间" width="160" sortable="custom" prop="modTime">
            <template #default="scope"><span class="meta-text">{{ formatDate(scope.row.modTime) }}</span></template>
          </el-table-column>
          <el-table-column label="大小" width="90" sortable="custom" prop="size">
            <template #default="scope"><span class="meta-text">{{ formatSize(scope.row.size) }}</span></template>
          </el-table-column>
          <el-table-column width="50" align="center">
            <template #default="scope">
              <el-tooltip v-if="scope.row.errorMsg" :content="scope.row.errorMsg" placement="left">
                <el-icon color="#F56C6C"><Warning /></el-icon>
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-main>
  </el-container>
</template>

<style scoped>
.layout-container { height: 100vh; background-color: #f0f2f5; }

/* 侧边栏 */
.sidebar { background: #fff; border-right: 1px solid #e4e7ed; display: flex; flex-direction: column; }
.sidebar-header { padding: 12px 16px; border-bottom: 1px solid #ebeef5; }
.sidebar-header :deep(.el-radio-button__inner) { width: 100%; text-align: center; }
.sub-header { padding: 12px 16px 0; }

/* 规则列表 */
.rules-scroll { flex: 1; }
.rules-list { padding: 12px; }
.rule-card { margin-bottom: 12px; }
.rule-card :deep(.el-card__header) { padding: 8px 14px; background: #fafafa; }
.rule-card :deep(.el-card__body) { padding: 12px 14px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.rule-label { font-weight: 600; font-size: 13px; color: #409EFF; }
.rule-inputs { display: flex; flex-direction: column; }
.rule-tips { font-size: 11px; color: #909399; margin: 0; line-height: 1.5; }

/* 菜单分组标题 */
.group-title {
  padding: 6px 20px 4px;
  font-size: 11px;
  color: #909399;
  font-weight: 600;
  list-style: none;
  cursor: default;
  user-select: none;
}

/* 脚本列表 */
.script-list { padding: 12px; }
.script-card {
  padding: 12px;
  border: 2px solid #ebeef5;
  border-radius: 8px;
  margin-bottom: 10px;
  cursor: pointer;
  transition: all 0.2s;
}
.script-card:hover { border-color: #c0c4cc; }
.script-card.script-active { border-color: #409EFF; background: #ecf5ff; }
.script-header { display: flex; justify-content: space-between; align-items: center; }
.script-name { display: flex; align-items: center; gap: 6px; font-weight: 600; font-size: 13px; color: #303133; }
.script-path { font-size: 10px; color: #c0c4cc; margin-top: 6px; word-break: break-all; }

/* 空状态 */
.empty-panel { text-align: center; color: #c0c4cc; padding: 40px 0; }
.empty-panel p { margin: 4px 0; }
.empty-panel .sub { font-size: 12px; }

/* 工作区 */
.main-content { padding: 16px; }
.workspace { background: #fff; padding: 16px; border-radius: 8px; box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08); height: 100%; display: flex; flex-direction: column; }
.action-bar { margin-bottom: 12px; display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.dir-tag { max-width: 280px; overflow: hidden; text-overflow: ellipsis; }

/* 表格 */
.name-cell { display: flex; align-items: center; gap: 6px; }
.name-changed { color: #67C23A; font-weight: 600; background: #f0f9eb; padding: 2px 6px; border-radius: 4px; }
.name-same { color: #909399; }
.meta-text { font-size: 12px; color: #909399; }

/* 工具类 */
.flex-row { display: flex; align-items: center; }
.p-text { font-size: 12px; color: #606266; white-space: nowrap; }
.mt-1 { margin-top: 4px; }
.mt-2 { margin-top: 8px; }
.mb-2 { margin-bottom: 8px; }
.ml-1 { margin-left: 4px; }
.ml-2 { margin-left: 8px; }
.mr-1 { margin-right: 4px; }
</style>
