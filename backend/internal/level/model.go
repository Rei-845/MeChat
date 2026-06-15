package level

const MaxLevel = 10

var thresholds = [MaxLevel]int{0, 10, 20, 30, 50, 70, 90, 120, 150, 190}

// 经验换算等级
func LevelFromXP(xp int) int {
	lv := 1
	for i := MaxLevel - 1; i >= 0; i-- {
		if xp >= thresholds[i] {
			lv = i + 1
			break
		}
	}
	return lv
}

// 等级对应牌段
func TierOf(lv int) string {
	switch {
	case lv <= 3:
		return "gray"
	case lv <= 6:
		return "blue"
	case lv <= 8:
		return "yellow"
	default:
		return "orange"
	}
}

// 到达该级所需累计经验
func XPForLevel(lv int) int {
	if lv < 1 {
		return 0
	}
	if lv > MaxLevel {
		return thresholds[MaxLevel-1]
	}
	return thresholds[lv-1]
}

// LevelInfo 等级信息
type LevelInfo struct {
	Experience     int    `json:"experience"`
	Level          int    `json:"level"`
	Tier           string `json:"tier"`             // gray / blue / yellow / orange
	CurrentLevelXP int    `json:"current_level_xp"` // 当前级内已累积经验
	NextLevelXP    int    `json:"next_level_xp"`    // 距下一级所需经验 满级为 0
	TodayCheckin   bool   `json:"today_checkin"`
}

// 组装等级信息
func BuildLevelInfo(xp int, checkin bool) LevelInfo {
	lv := LevelFromXP(xp)
	tier := TierOf(lv)
	currentStart := XPForLevel(lv)
	inLevel := xp - currentStart
	var needed int
	if lv < MaxLevel {
		needed = XPForLevel(lv+1) - xp
	}
	return LevelInfo{
		Experience:     xp,
		Level:          lv,
		Tier:           tier,
		CurrentLevelXP: inLevel,
		NextLevelXP:    needed,
		TodayCheckin:   checkin,
	}
}
