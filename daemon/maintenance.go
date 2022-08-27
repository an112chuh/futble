package daemon

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func CommandLine() {
	var s string
	reader := bufio.NewReader(os.Stdin)
	for {
		s, _ = reader.ReadString('\n')
		if runtime.GOOS == "windows" {
			s = strings.Replace(s, "\r\n", "", -1)
		} else {
			s = strings.Replace(s, "\n", "", -1)
		}
		if s != `` {
			Task := strings.Split(s, " ")
			if len(Task) < 1 {
				fmt.Println("Command error")
				continue
			}
			if Task[0] == "Maintenaunce" {
				Task = Task[1:]
				MaintenaunceReason = strings.Join(Task, " ")
				fmt.Printf("Maintenance after 10 min, reason - %s\n", MaintenaunceReason)
				IsMaintenaunce = true
			}
		}
	}
}
