package repo

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

func nextDailyDocNo(db *gorm.DB, model any, col, prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || col == "" {
		return "", fmt.Errorf("doc no prefix required")
	}
	var last string
	err := db.Model(model).
		Where(col+" LIKE ?", prefix+"%").
		Order(col + " DESC").
		Limit(1).
		Pluck(col, &last).Error
	if err != nil {
		return "", err
	}
	seq := 1
	if last != "" && len(last) > len(prefix) {
		var n int
		if _, scanErr := fmt.Sscanf(last[len(prefix):], "%d", &n); scanErr == nil && n >= 0 {
			seq = n + 1
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

func dailyPrefix(code string) string {
	return strings.TrimSpace(code) + time.Now().Format("20060102")
}
