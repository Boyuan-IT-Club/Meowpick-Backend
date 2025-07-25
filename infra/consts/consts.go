package consts

var PageSize int64 = 10

// 数据库相关
const (
	ID           = "_id"
	UserID       = "user_id"
	Status       = "status"
	CreatedTime  = "created_time"
	UpdatedTime  = "updated_time"
	DeleteStatus = 3
	EffectStatus = 0
	Phone        = "phone"
	Timestamp    = "timestamp"
	LogId        = "log_id"
	NotEqual     = "$ne"
)

// http
const (
	Post                      = "POST"
	PlatformSignInUrl         = "https://api.xhpolaris.com/platform/auth/sign_in"
	PlatformSetPasswordUrl    = "https://api.xhpolaris.com/platform/auth/set_password"
	PlatformSendVerifyCodeUrl = "https://api.xhpolaris.com/platform/auth/send_verify_code"
	ContentTypeJson           = "application/json"
	CharSetUTF8               = "UTF-8"
	Beta                      = "beta"
	OpenApiCallUrl            = "https://api.coze.cn/v3/chat"                                // openapi地址
	ExerciseUrl               = "https://essay.cubenlp.com/api/algorithm/generate_exercises" // 练习生成地址
	BeeTitleUrlOcr            = "https://api.xhpolaris.com/essay/sts/ocr/title/bee/url"      //bee的ocr且使用url形式，保留标题
	BetaEvaluateUrl           = "https://api.xhpolaris.com/essay/evaluate"                   // essay-stateless的批改接口
)

// 默认值
const (
	DefaultCount     = 30
	AppId            = 14
	Like             = 1
	DisLike          = -1
	InvitationReward = 10
	AttendReward     = 1
)
