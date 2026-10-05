# 私密反馈与管理员回复

本文描述本分支实现的站内私密反馈接口。课程评论仍为公开内容，反馈采用独立对话。

## 已确认的体验

- 登录用户可提交纯文字反馈，分类可选为“问题反馈／功能建议／其他”，不选时为“其他”。第一版不支持截图或其他附件。
- 反馈条目、正文、回复和作者身份仅作者本人及管理员可见；其他普通用户无法查询或推断反馈是否存在。
- 作者和管理员可多轮对话。新建为 `pending`；管理员回复后为 `answered`；作者继续回复后回到 `pending`；管理员可设为 `closed`，作者在已关闭对话中追加消息时重新打开为 `pending`。
- 作者可查看自己的反馈列表、详情和管理员回复；管理员可查看全部反馈、筛选分类和状态、打开详情并回复。双方有站内未读数和已读状态，不发微信订阅消息。
- 作者可主动关闭反馈；已关闭反馈收到作者新消息后重新打开。第一版不允许作者删除反馈或其中的消息，以保留可核对的完整对话。
- 反馈正文不调用外部内容审核。每条消息最多 2,000 字；每位用户每分钟最多发送 5 条反馈消息、每天最多新建 10 条反馈；超过限制返回稳定业务错误码。管理员可在站内处理异常内容。
- 管理员共用待处理列表，第一版不指派负责人；每位管理员的未读状态独立计算。

## 数据与接口

- 反馈主体记录 ID、作者 ID、分类、状态、创建／最后活动时间、；对话消息独立保存 ID、反馈 ID、发送者 ID 与角色、文本、时间。独立消息集合避免单个反馈的消息数组无限增长。
- 作者接口为 `POST /api/feedback`、`GET /api/feedback/mine`、`GET /api/feedback/:feedbackId`、`POST /api/feedback/:feedbackId/messages`、`POST /api/feedback/:feedbackId/close`；管理员接口为 `GET /api/admin/feedback`、`POST /api/admin/feedback/:feedbackId/reply`、`POST /api/admin/feedback/:feedbackId/close`。完整 schema 见本分支 Swagger。
- 列表按最后活动时间倒序分页，展示分类、状态、最近消息摘要、未读数；详情按 `sequence` 倒序分页读取消息，不自动改变已读。客户端展示后显式提交所看到的最大序号，避免把尚未展示的新消息误标已读。管理员列表可按分类、状态及关键词筛选，按最后活动时间及 ID 稳定倒序。
- 作者与管理员的已读位置分别记录。多位管理员各自阅读不应抹掉其他管理员的未读；待处理状态属于整条反馈，与某个管理员是否已读分开。
- 每次发送消息、回复、关闭和重新打开都要进行登录与归属校验；普通用户猜测其他反馈 ID 时返回与不存在相同的结果。消息创建与反馈状态／最后活动时间更新同事务完成，防止出现消息已入库但状态未变。
- 管理员回复、关闭和重新打开应进入管理员操作日志，目标类型增加 `feedback`，记录操作者和反馈 ID。普通用户消息留在反馈对话中；日志中的私密正文仅管理员可见。

前端需要新建反馈入口、我的反馈／详情／回复页、管理员列表／详情页及未读数展示。当前前端只有 QQ 群反馈文案，没有站内反馈页面；本设计不修改前端代码。权限、状态、未读及错误码已写入 Swagger。

## 完整接口清单

所有接口要求登录；`/api/admin/feedback` 下的接口额外要求管理员权限。

| 方法 | 作者路径 | 管理员路径 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/feedback` | — | `{text, category?}`；分类 `bug/feature/other`，默认 other |
| GET | `/api/feedback/mine` | `/api/admin/feedback` | page/pageSize、status、category；管理员另支持 keyword 搜索整段对话 |
| GET | `/api/feedback/:id` | `/api/admin/feedback/:id` | 对话详情，messages 按 sequence 倒序分页 |
| POST | `/api/feedback/:id/messages` | `/api/admin/feedback/:id/reply` | `{text}` 继续对话 |
| POST | `/api/feedback/:id/read` | `/api/admin/feedback/:id/read` | `{sequence: 已展示的最大序号}`；只前进，不得超过当前最大值 |
| POST | `/api/feedback/:id/close` | `/api/admin/feedback/:id/close` | 关闭对话，无需请求体 |
| GET | `/api/feedback/unread` | `/api/admin/feedback/unread` | 当前查看者未读的对方消息总数，不是对话数量 |

列表返回 `feedbacks[]`、`total`；详情／发送返回 `feedback`、`messages[]`、`total`（消息数）。每个反馈包括 id、userId、category、status、sequence、summary、createdAt、updatedAt、unreadCount；每条消息包括 sequence、role（author/admin）、userId、text、createdAt。管理员身份用于作者理解回复来源，反馈作者身份仅作者和管理员能获取。

错误码：`112000001` 不存在或不属于当前作者，`112000002` 正文／分类／已读序号非法，`112000003` 一分钟 5 条消息超限，`112000004` UTC+8 当天新反馈达到 10 条。沿用 HTTP 200 与非零业务 code；GET 不改变已读，新消息不会因查看旧页而被消耗。

反馈错误码使用独立的 `112` 号段，避免与通用请求参数错误 `110000001` 冲突。类型不正确、缺失必填字段或畸形 JSON 仍返回通用请求参数错误；通过 JSON 绑定后的正文／分类／已读校验使用 `112000002`。
