package initial

import (
	"gorm.io/gorm"
)

// promptRow は、初期データ投入専用のDBモデル。
type promptRow struct {
	ID int `gorm:"column:id;primaryKey"`
	DifficultyID int `gorm:"column:difficulty_id;foreignKey"`
	PromptTextJa string `gorm:"column:prompt_text_ja"`
	TargetRomaji string `gorm:"column:target_romaji"`
}

// TableName は、GORM に使用させるテーブル名を明示的に指定。
func (promptRow) TableName() string {
	return "prompts"
}

// loadPrompts は、お題（Prompt）の初期データを難易度ごとに登録する関数。
func loadPrompts(db *gorm.DB) error {
	// 登録対象の初期お題データ一覧
	prompts := []promptRow{
		// Easy
		{
			ID: 1,
			DifficultyID: 1,
			PromptTextJa: "日本",
			TargetRomaji: "nihonn",
		},
		{
			ID: 2,
			DifficultyID: 1,
			PromptTextJa: "寿司",
			TargetRomaji: "susi",
		},
		{
			ID: 3,
			DifficultyID: 1,
			PromptTextJa: "花火",
			TargetRomaji: "hanabi",
		},
		{
			ID: 4,
			DifficultyID: 1,
			PromptTextJa: "青空",
			TargetRomaji: "aozora",
		},
		{
			ID: 5,
			DifficultyID: 1,
			PromptTextJa: "朝日",
			TargetRomaji: "asahi",
		},
		{
			ID: 6,
			DifficultyID: 1,
			PromptTextJa: "風鈴",
			TargetRomaji: "fuurinn",
		},
		{
			ID: 7,
			DifficultyID: 1,
			PromptTextJa: "夕暮れ",
			TargetRomaji: "yuugure",
		},
		{
			ID: 8,
			DifficultyID: 1,
			PromptTextJa: "昼寝",
			TargetRomaji: "hirune",
		},
		{
			ID: 9,
			DifficultyID: 1,
			PromptTextJa: "ご飯",
			TargetRomaji: "gohann",
		},
		{
			ID: 10,
			DifficultyID: 1,
			PromptTextJa: "そよ風",
			TargetRomaji: "soyokaze",
		},
		{
			ID: 11,
			DifficultyID: 1,
			PromptTextJa: "木漏れ日",
			TargetRomaji: "komorebi",
		},
		{
			ID: 12,
			DifficultyID: 1,
			PromptTextJa: "雨上がり",
			TargetRomaji: "ameagari",
		},
		{
			ID: 13,
			DifficultyID: 1,
			PromptTextJa: "朝ご飯",
			TargetRomaji: "asagohann",
		},
		{
			ID: 14,
			DifficultyID: 1,
			PromptTextJa: "かたつむり",
			TargetRomaji: "katatumuri",
		},
		{
			ID: 15,
			DifficultyID: 1,
			PromptTextJa: "ねこじゃらし",
			TargetRomaji: "nekojarasi",
		},

		// Normal
		{
			ID: 16,
			DifficultyID: 2,
			PromptTextJa: "朝ご飯の時間",
			TargetRomaji: "asagohannojikann",
		},
		{
			ID: 17,
			DifficultyID: 2,
			PromptTextJa: "放課後の図書館",
			TargetRomaji: "houkagonotosyokann",
		},
		{
			ID: 18,
			DifficultyID: 2,
			PromptTextJa: "運動会の練習",
			TargetRomaji: "undoukainorensyuu",
		},
		{
			ID: 19,
			DifficultyID: 2,
			PromptTextJa: "修学旅行",
			TargetRomaji: "shuugakuryokou",
		},
		{
			ID: 20,
			DifficultyID: 2,
			PromptTextJa: "文化祭の準備",
			TargetRomaji: "bunkasainojunbi",
		},
		{
			ID: 21,
			DifficultyID: 2,
			PromptTextJa: "駅前の喫茶店",
			TargetRomaji: "ekimaenokissatenn",
		},
		{
			ID: 22,
			DifficultyID: 2,
			PromptTextJa: "校庭の桜",
			TargetRomaji: "kouteinosakura",
		},
		{
			ID: 23,
			DifficultyID: 2,
			PromptTextJa: "冬休みの宿題",
			TargetRomaji: "fuyuyasuminoshukudai",
		},
		{
			ID: 24,
			DifficultyID: 2,
			PromptTextJa: "教室の黒板",
			TargetRomaji: "kyousitunokokubann",
		},
		{
			ID: 25,
			DifficultyID: 2,
			PromptTextJa: "電車の切符",
			TargetRomaji: "densyanokippu",
		},
		{
			ID: 26,
			DifficultyID: 2,
			PromptTextJa: "理科の実験",
			TargetRomaji: "rikanojikkenn",
		},
		{
			ID: 27,
			DifficultyID: 2,
			PromptTextJa: "体育館の入り口",
			TargetRomaji: "taiikukannnoiriguti",
		},
		{
			ID: 28,
			DifficultyID: 2,
			PromptTextJa: "給食の時間",
			TargetRomaji: "kyuusyokunojikann",
		},
		{
			ID: 29,
			DifficultyID: 2,
			PromptTextJa: "教科書の内容",
			TargetRomaji: "kyoukasyononaiyou",
		},
		{
			ID: 30,
			DifficultyID: 2,
			PromptTextJa: "部活動の仲間",
			TargetRomaji: "bukatudounonakama",
		},

		// Hard
		{
			ID: 31,
			DifficultyID: 3,
			PromptTextJa: "情報処理の授業内容",
			TargetRomaji: "jouhousyorinojugyounaiyou",
		},
		{
			ID: 32,
			DifficultyID: 3,
			PromptTextJa: "夏休みの自由研究",
			TargetRomaji: "natuyasuminojiyuukenkyuu",
		},
		{
			ID: 33,
			DifficultyID: 3,
			PromptTextJa: "科学技術の進歩",
			TargetRomaji: "kagakugijutunosinpo",
		},
		{
			ID: 34,
			DifficultyID: 3,
			PromptTextJa: "環境保護の取り組み",
			TargetRomaji: "kankyohogonotorikumi",
		},
		{
			ID: 35,
			DifficultyID: 3,
			PromptTextJa: "未来都市の開発計画",
			TargetRomaji: "miraitosinokaihatukeikaku",
		},
		{
			ID: 36,
			DifficultyID: 3,
			PromptTextJa: "人工知能の活用事例",
			TargetRomaji: "jinkoutinounokatuyoujirei",
		},
		{
			ID: 37,
			DifficultyID: 3,
			PromptTextJa: "地球温暖化の影響",
			TargetRomaji: "tikyuuondankanoeikyou",
		},
		{
			ID: 38,
			DifficultyID: 3,
			PromptTextJa: "日本経済の現状分析",
			TargetRomaji: "nihonkeizainogenjoubunseki",
		},
		{
			ID: 39,
			DifficultyID: 3,
			PromptTextJa: "歴史教育の重要性",
			TargetRomaji: "rekisikyouikunojuuyousei",
		},
		{
			ID: 40,
			DifficultyID: 3,
			PromptTextJa: "文化交流のイベント",
			TargetRomaji: "bunkakouryuunoibento",
		},
		{
			ID: 41,
			DifficultyID: 3,
			PromptTextJa: "再生可能エネルギー",
			TargetRomaji: "saiseikanouenerugi-",
		},
		{
			ID: 42,
			DifficultyID: 3,
			PromptTextJa: "世界遺産の保護活動",
			TargetRomaji: "sekaiisannnohogokatudou",
		},
		{
			ID: 43,
			DifficultyID: 3,
			PromptTextJa: "医療技術の進化",
			TargetRomaji: "iryogijutunosinka",
		},
		{
			ID: 44,
			DifficultyID: 3,
			PromptTextJa: "教育改革の方針",
			TargetRomaji: "kyouikukaikakunohousinn",
		},
		{
			ID: 45,
			DifficultyID: 3,
			PromptTextJa: "宇宙開発の最前線",
			TargetRomaji: "utyuukaihatunosaizensenn",
		},
	}

	// 各お題データをDBへ登録（存在しない場合のみ作成）
	for _, prompt := range prompts {
		if err := db.FirstOrCreate(&prompt).Error; err != nil {
			return err
		}
	}

	return nil
}
