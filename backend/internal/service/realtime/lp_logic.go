package realtime

// MaxLP は 1プレイヤーの最大LP（現在は 100 固定）。
const MaxLP int64 = 100

// InitLP は試合開始時の初期LPを返す。
func InitLP() int64 {
	return MaxLP
}

// ApplyDamage は currentLP から damage を引き、0 未満にならないようにクリップする。
func ApplyDamage(currentLP, damage int64) int64 {
	if damage <= 0 {
		return currentLP
	}
	next := currentLP - damage
	if next < 0 {
		return 0
	}
	if next > MaxLP {
		return MaxLP // 念のため
	}
	return next
}
