package constants

import (
	"encoding/csv"
	"fmt"
	"os"
)

var NationMatches map[string][]string

func InitNations() error {
	f, err := os.Open("countries.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}
	NationMatches = make(map[string][]string)
	for i := range records {
		NationMatches[records[i][1]] = append(NationMatches[records[i][1]], records[i][3])
	}
	fmt.Println(NationMatches)
	return nil
}
