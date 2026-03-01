// state encapsulates the global state of the applications so we can see all
// mutations in one place
package state

import (
	"fmt"
	"time"

	"github.com/ethancarlsson/hurl-lsp/pkg/hurlfile"
)

var (
	updater func() error       = nil
	lines   []string           = []string{}
	hf      *hurlfile.HurlFile = &hurlfile.HurlFile{}
)

func Lines() []string {
	return lines
}

func SetLines(ls []string) []string {
	lines = ls
	return lines
}

func SetHfFromLines(ls []string) (*hurlfile.HurlFile, error) {
	var err error
	hf, err = hurlfile.Parse(ls)
	if err != nil {
		return hf, fmt.Errorf("Failed to parse the hurl file %s", err.Error())
	}

	return hf, nil
}

func ResetHf(after time.Duration, then func(hf *hurlfile.HurlFile)) {
	if updater != nil {
		return
	}

	updater = func() error {
		time.Sleep(300 * time.Millisecond)

		hf, err := SetHfFromLines(lines)
		if err != nil {
			println(err.Error())
		}

		then(hf)

		updater = nil
		return nil
	}

	go updater()
}

func Hf() *hurlfile.HurlFile {
	return hf
}
