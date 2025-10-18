package internal

import (
	"fmt"
	"strings"
)

// PrintTable prints evaluation results in a formatted table
func PrintTable(rows [][3]string) {
	if len(rows) == 0 {
		fmt.Println("No evaluation results.")
		return
	}
	// simple fixed-width columns
	w1, w2 := 6, 8
	for _, r := range rows {
		if len(r[0]) > w1 {
			w1 = len(r[0])
		}
		if len(r[1]) > w2 {
			w2 = len(r[1])
		}
	}
	fmt.Printf("%-*s  %-*s  %s\n", w1, "Action", w2, "Decision", "Matched (details)")
	fmt.Printf("%s  %s  %s\n", strings.Repeat("-", w1), strings.Repeat("-", w2), strings.Repeat("-", 40))
	for _, r := range rows {
		fmt.Printf("%-*s  %-*s  %s\n", w1, r[0], w2, r[1], r[2])
	}
}
