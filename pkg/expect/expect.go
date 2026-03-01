package expect

import (
	"reflect"
	"strings"
	"testing"
)

func NoErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
}

func Err(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected an error but got nil")
	}
}

func ErrContains(t *testing.T, errMsg string, err error) {
	if !strings.Contains(err.Error(), errMsg) {
		t.Errorf("expected err to contain %s got %s", errMsg, err.Error())
	}
}

func Equals(t *testing.T, expected, actual any) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("failed asserting %v equals %v", actual, expected)
	}
}
