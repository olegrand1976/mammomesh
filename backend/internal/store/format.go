package store

import (
	"strconv"
	"strings"
)

func itoa(i int) string { return strconv.Itoa(i) }

// ftoa écrit un nombre à la française (virgule décimale).
func ftoa(f float64) string {
	return strings.Replace(strconv.FormatFloat(f, 'f', -1, 64), ".", ",", 1)
}
