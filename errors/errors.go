package errors

import (
	"fmt"
	"os"
)

/*CheckErr checks error, outputs error messade and exit program at critical time
err: the returned error
msgIntro: specific error message explainations for user
stop: exits program on error if set to 'true'
*/
func CheckErr(err error, msgIntro string, stop bool) {
	if err != nil {
		fmt.Println(msgIntro, err)
		if stop {
			os.Exit(1)
		}
	}
}
