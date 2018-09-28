package errors

import (
	"fmt"
	"os"
)

//CheckErr, error managing function and exiting program on failure
func CheckErr(err error, msgIntro string, stop bool) {
	if err != nil {
		fmt.Println(msgIntro, err)
		if stop {
			os.Exit(1)
		}
	}
}
