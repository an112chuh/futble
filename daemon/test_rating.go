package daemon

import (
	"fmt"
	"math"
)

func TestCountRateDiff() {
	var ChangeWin, ChangeLose int
	type st struct {
		Score  int
		Trophs int
	}
	var User [2]st
	for {
		fmt.Scanf("%d\n", &User[0].Score)
		fmt.Scanf("%d\n", &User[1].Score)
		fmt.Scanf("%d\n", &User[0].Trophs)
		fmt.Scanf("%d\n", &User[1].Trophs)
		if User[1].Trophs < User[0].Trophs {
			tmp := User[0]
			User[0] = User[1]
			User[1] = tmp
		}
		Diff := User[1].Trophs - User[0].Trophs
		k := (math.Log(float64(Diff+200)) * 1.5) - 6.94747605
		fmt.Printf("Diff = %d, k = %f\n", Diff, k)
		var Sum float64 = 60
		Sum /= (k + 1)
		var k2 float64 = 1.
		if User[1].Score > User[0].Score {
			ChangeWin = int(math.Round(Sum)) + 1
			if User[0].Trophs < 1000 {
				k2 = float64((User[0].Trophs)/10) / 100
			}
			if User[0].Trophs-int(Sum*k2) < 0 {
				ChangeLose = -User[0].Trophs
			} else {
				ChangeLose = -int(math.Round(Sum * k2))
			}
		} else if User[1].Score < User[0].Score {
			fmt.Printf("Here, SumFloat = %f, sum = %d\n", Sum*k, int(math.Round(Sum*k)))
			ChangeWin = int(math.Round(Sum*k)) + 1
			if User[1].Trophs < 1000 {
				k2 = float64((User[1].Trophs)/10) / 100
			}
			fmt.Printf("k2 = %f\n", k2)
			if User[1].Trophs-int(Sum*k2*k) < 0 {
				ChangeLose = -User[1].Trophs
			} else {
				ChangeLose = -int(math.Round(Sum * k2 * k))
			}
		} else if User[1].Score == User[0].Score {
			ChangeWin = int(math.Round(Sum*k)) - 30
			if User[1].Trophs < 1000 {
				k2 = float64((User[0].Trophs)/10) / 100
			}
			if User[1].Trophs-int(Sum*k*k2)+int(math.Round(30*k2)) < 0 {
				ChangeLose = -User[0].Trophs + int(math.Round(30*k2))
			} else {
				ChangeLose = -int(math.Round(Sum*k2*k)) + int(math.Round(30*k2))
			}

		}
		fmt.Printf("Rating diff: Win - %d, Lose - %d\n", ChangeWin, ChangeLose)
	}
}
