package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/test-go/testify/require"
)

func TestTimeParse(t *testing.T) {
	s := "2023-10-11T08:00:00.000Z"
	t1, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)

	t2, err := time.Parse(time.RFC3339, "2023-10-11T14:26:10.55912887+08:00")
	require.NoError(t, err)
	fmt.Println(t1, t2, t2.Before(t1))

	t3 := t1.Local()
	fmt.Println(t3)
}
