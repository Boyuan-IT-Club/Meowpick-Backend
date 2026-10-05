# 提案资料修改功能：前端对接说明

本文件描述**旧版前端 → 本分支后端契约**，供前端独立实施。接口 schema、字段和错误码已写入 Swagger；本任务没有修改前端仓库。

## 当前版本的依赖

| 页面或功能 | 当前依赖 | 新版需要调整 |
| --- | --- | --- |
| 新增课程提案表单 | `title` 复制课程名；`course.teachers` 选择已有教师 ID 或填写新教师 | 不再提交 `title`；新增 `type=create_course`；教师选择流程仍可复用 |
| 提案列表与详情 | 以 `course.name` 为主、`title` 回退；假设提案都有 `course` | 根据 `type` 显示课程创建、课程修改或教师修改；用后端显示名及目标资料，不依赖 `title` |
| 我的提案卡片 | 直接显示 `title`，主体按 `finalCourse || course` 渲染 | 改用 `type`、目标摘要及建议／最终变更；教师提案不能套课程卡片 |
| 审批页 | 支持新增课程 `finalCourse`，并复制课程名到 `title` | 按类型展示原值→建议值→最终值；处理重复候选、冲突重审、共同审批影响数量 |
| 课程详情 | 单个 `contributor` | 改读 `contributors[]`；分页读取课程资料历史 |
| 教师界面 | 仅教师建议；无独立教师资料修改页 | 新增教师详情及修改教师提案入口；历史分页展示 |
| 管理员操作记录 | `proposalSnapshot.courseName || title`，动作通用文案 | 根据提案类型、目标及变更摘要展示；同一 `decisionBatchId` 为一次审批决定 |

## 提案类型与表单

- `type`：`create_course`、`update_course`、`update_teacher`。旧提案在响应中统一呈现为 `create_course`，即使数据库记录没有 `type`。
- `create_course` 继续提交完整 `course`；`update_course` 提交课程 `targetId` 与只含拟改字段的 `suggested`；`update_teacher` 提交教师 `targetId` 与对应 `suggested`。例如 `{"type":"update_course","targetId":"课程ID","suggested":{"code":"CS102"}}` 只建议修改代码。
- `suggested` 中没有某字段即“不修改”；教师 `title`、`department` 明确传 `""` 即“申请清空”。任课教师列表如需修改，传完整新列表，不传局部增删操作。
- 新增教师仍可在课程创建／修改表单中填写。已有教师以 ID 标识；建议重复的新教师由管理员选择复用或新建。修改已有教师资料应跳转教师修改提案，不在课程任课教师弹窗里直接修改。
- `title` 从新请求和响应中取消；历史提案的显示名由后端按类型和目标返回。`content` 保留为补充说明。

## 审批结果和异常交互

| 结果 | 前端应展示或执行的动作 |
| --- | --- |
| 已生效资料与提交内容相同 | 提交立即返回专用业务错误码；展示现有课程或教师，不创建无效提案 |
| 六项一致的待审新增课程提案 | 课程名、代码、完整教师身份集合、校区集合、分类、开课院系相同时自动成组；已有教师按 ID、新教师按姓名比较，教师职称不比较；提交时可提示用户已有待审建议，但允许不同作者继续提交；审批一份会联动通过整批 |
| 同名待审提案 | 审批预览展示同名候选及完整字段差异；管理员也可按课程名模糊搜索其他待审提案并勾选需共同审批者 |
| 已存在的正式课程候选 | 与待审提案分开展示；管理员确认已有课程后手动拒绝所选新增提案，可批量勾选并填写共同理由、附课程链接／建议另提修改；其他提案不自动拒绝 |
| 修改字段已等于建议值 | 提示该更正已实现，不再作为新的实际资料修改通过 |
| 修改字段出现第三种值 | 展示原值、当前值、建议值；管理员显式重审并填写最终值 |
| 共同审批 | 显示本次同时处理的提案数量和 ID；手动合并时逐份展示原建议、统一的最终资料及预计积分；每份提案都有通过记录，实体只变更一次 |
| 整批撤回 | 显示整批影响、积分回扣和目标资料恢复；若存在尚未撤回的后续修改则显示不可撤回原因 |

服务端必须在 Swagger 中给出上述每类结果的稳定错误码或响应状态、字段结构、空值语义以及示例。前端不要仅靠中文错误文本分支。

提案课程名输入当前会对同名课程返回多条相同名称。新后端在 `field=courseName` 建议中按名称去重后分页，`total` 变为唯一名称数；前端现有按 `value` 选中的交互可以沿用。

## 展示与权限

- 课程 `contributors[]` 和教师详情贡献者按用户去重。昵称取当前值，可能随用户改名变化。
- 公开资料历史一条对应一次实际创建或修改，即一批共同审批；同批多位作者在该条下汇总。个人提案历史仍是一人一条。
- 匿名提案在其他普通用户可见的列表、详情和公开历史中不返回作者 ID／昵称；作者本人及管理员有权限查看。前端不得通过 `proposalId` 再取作者身份并公开展示。
- 课程、教师资料历史分别分页读取；没有来源提案的旧记录不补造创建事件。教师资料修改者不会自动出现在其所授每门课程的贡献者列表。

## 操作日志页面

- 管理员全局日志、按提案分组日志、时间线日志都要按 `type`、目标 ID、显示名、变更摘要渲染，不再假设 `proposalSnapshot.courseName` 存在。
- 时间线的每份提案操作日志保留，但同批日志共用 `decisionBatchId`；一条课程或教师资料修改事件表示实际实体只变更一次。
- 审批／拒绝／撤回动作和自动联动来源须明确显示。日志中的原值、建议值、最终值来自操作当时的快照，不应拿当前提案内容回填。

## 发布对齐

后端响应将移除 `title` 并将课程 `contributor` 改为 `contributors[]`，旧前端的“我的提案”、课程贡献者和日志页会受影响。前端需要在后端新契约上线时同步发布适配版本；具体 schema 以本分支生成的 Swagger 为准。

## 接口清单与调用顺序

| 方法和路径 | 用途／前端注意事项 |
| --- | --- |
| `POST /api/proposal/add` | 三类提交；响应 `pendingDuplicateIds` 可非阻断提示 |
| `POST /api/proposal/:id/update` | 管理员保存共同草稿：新增传完整 `course`，修改传 `suggested` 的最终值；原建议保持不变 |
| `POST /api/proposal/:id/preview` | 传 `proposalIds`、`finalCourse` 或 `final`、`confirmedNewTeachers`；返回 `previewToken`、成员、疑似项、正式课程、教师候选、差异和预计积分 |
| `POST /api/proposal/:id/approve` | 原样提交本次预览参数，加 `previewToken`；不能直接审批 |
| `POST /api/proposal/:id/reject` | 传额外 `proposalIds` 和共同 `reason`；只拒绝明确选择的成员 |
| `POST /api/proposal/:id/revoke` | 沿用 `actionType=approve/reject`；通过撤回按整个批次执行 |
| `GET /api/proposal/suggest` | 管理员 `type=create_course&status=pending&keyword=课程名` 搜索待审共同审批候选 |
| `GET /api/teacher/:id` | 当前教师资料与 `contributors[]` |
| `GET /api/course/:id/history`、`GET /api/teacher/:id/history` | `page`／`pageSize` 分页；`history[]` 一条对应实际变更；旧来源 `legacy=true` 无 `final` |

`POST /api/teacher/add`、旧 `GET /api/teacher/suggest` 已移除。教师字段建议继续使用 `GET /api/proposal/field-suggestions?field=teacherName`，已有教师 ID 可从课程或教师详情取得。

### 修改提案示例

```json
{"type":"update_teacher","targetId":"教师ID","suggested":{"title":"副教授","department":""},"content":"更新资料","showUsername":true}
```

`changes` 不是客户端必填字段；使用 `suggested` 表达想改的值，`before` 为后端保存的原值，`final` 为管理员确认值。新增课程继续用 `course`／`finalCourse`。

### 审批错误码

业务错误沿用 HTTP 200、非零 `code`、`data=null`，不要从文案中解析候选 ID。

| code | 前端处理 |
| --- | --- |
| `108000008` | 同一作者已有相同待审提案 |
| `108000009` | 精确正式课程已存在；从预览 `existingCourses` 展示并由管理员手动拒绝 |
| `108000029` | 无实际变更；不创建或批准无效提案 |
| `108000030` | 缺少预览，先调用 preview |
| `108000031` | 预览已失效，重新获取候选并确认 |
| `108000032` | 拟改字段冲突，显示原值／当前值／建议值；明确传 final 再预览 |
| `108000033` | 正式目标不存在 |
| `108000034` | 新教师与正式教师同名；选择已有 ID，或姓名加入 confirmedNewTeachers 再预览 |
| `108000022`、`108000028` | 存在后续生效变更或已满 24 小时，不能撤回 |

操作日志全局列表新增 `proposalType`、`entityType`、`entityId`、`decisionBatchId`、`triggerProposalId`、`automatic`、`snapshot`，实际实体事件带 `before`／`final`。旧日志没有 `snapshot` 时显示文字说明即可。课程贡献者昵称随用户改名更新。

反馈接口、状态、未读调用见 [FEEDBACK-DESIGN.md](./FEEDBACK-DESIGN.md)。
