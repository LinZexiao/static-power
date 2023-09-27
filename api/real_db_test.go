package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/test-go/testify/require"
)

func TestFind(t *testing.T) {
	db := realDB(t, "/root/tanlang/others/static-power/test.db")
	a := NewApi(db)

	res, err := a.find(Option{Tag: "invalid_tag"})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func TestFindVenus(t *testing.T) {
	db := realDB(t, "/root/tanlang/others/static-power/test.db")
	a := NewApi(db)

	// bf := time.Now().Add(-time.Hour * 4)
	bf := time.Now()
	res, err := a.findVenus(Option{Tag: "Japan", Before: bf})
	require.NoError(t, err)
	fmt.Println(len(res), res)
}

// 3325 7786 7830 20522 34548 118330 134516 522948 709366 867300 1114587 1114827 1212159 1227975 1228000 1228008 1228087 1228089 1228100 1228105 1285787 1349048 1717477 1757717 1874059 1874063 1886543 1889491 1889610 1889619 1889627 1908671 1915133 1934613 1967469 1967501 1967748 1968116 1968296 1968604 1968673 1969306 1969323 1969339 1971310 1971588 1975316 1975326 1975336 1975338 1984576 1984580 1984586 1984593 1993339 1993388 2002688 2002827 2002869 2002888 2003333 2003555 2003866 2003888 2006999 2012955 2013269 2042122 2042287 2046718 2059869 2086809 2104858 2214491 2224855 2239302 2251680
