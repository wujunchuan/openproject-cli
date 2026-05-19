package printer

import (
	"time"

	"github.com/briandowns/spinner"
)

type function[T any] func() (T, error)

var loadingSpinner *spinner.Spinner
var silent bool

func init() {
	loadingSpinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	loadingSpinner.Suffix = " fetching data"
	_ = loadingSpinner.Color("yellow")
}

func SetSilent(b bool) {
	silent = b
}

func WithSpinner[T any](f function[T]) (T, error) {
	if !silent {
		loadingSpinner.Start()
		defer loadingSpinner.Stop()
	}
	return f()
}
