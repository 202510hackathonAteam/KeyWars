package realtime

type AnswerJudgeResult struct {
	IsQuestionFinished bool
	IsAttackSuccess    bool
	DamageToOpponent   int64
}

// JudgeAnswer は「このプレイヤーのこの問題に対する回答」について、
// ・この回答で問題が終了するか
// ・相手に与えるダメージはいくつか
// を判定する。
// 注意: 「両者とも時間切れでノーダメージにする」判定は、
// 呼び出し側（2人分の結果を見れるところ）で行う。
func JudgeAnswer(
	isCorrect bool, //正解かどうか
	serverElapsedMs int64, //サーバー回答時間
	limitMs int64, //問題の制限時間
	totalMissCount int, //ミスカウント
) AnswerJudgeResult {
	// 制限時間オーバー時は両者ノーダメージ
	if serverElapsedMs > limitMs {
		return AnswerJudgeResult{
			IsQuestionFinished: true,
			IsAttackSuccess:    false,
			DamageToOpponent:   0,
		}
	}

	//制限時間内だけど不正解で問題継続
	if !isCorrect {
		return AnswerJudgeResult{
			IsQuestionFinished: false,
			IsAttackSuccess:    false,
			DamageToOpponent:   0,
		}
	}

	//制限時間内で正解の場合、攻撃成功で問題終了
	baseDamage := int64(10)
	damage := baseDamage - int64(totalMissCount)
	if damage < 0 {
		damage = 0
	}
	return AnswerJudgeResult{
		IsQuestionFinished: true,
		IsAttackSuccess:    true,
		DamageToOpponent:   damage,
	}

}
