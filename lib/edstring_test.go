package gopled

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type MyTE struct {
	str1 string
	str2 string
	ed   int
}

func generateKnownCases() []MyTE {

	return []MyTE{
		{"", "", 0},
		{" ", "", 1},
		{"", "a", 1},
		{"a", "", 1},
		{"", "abc", 3},
		{"bbb", "a", 3},
		{"abc", "bcd", 2},
		{"kitten", "sitting", 3},
	}
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestEditDistance1(t *testing.T) {

	mytes := generateKnownCases()

	for _, myte := range mytes {
		fmt.Printf("TestEditDistance1: str1 = %s, str2 = %s, ed = %d\n", myte.str1, myte.str2, myte.ed)

		aed := EditDistance(myte.str1, myte.str2)

		if aed != myte.ed {
			t.Fatalf(`str1 = %q, str2 = %q, ed = %v, want = %v`, myte.str1, myte.str2, myte.ed, aed)
		}
	}

}

func TestEditDistanceParallel_v(t *testing.T) {

	v := 1

	mytes := generateKnownCases()

	for _, myte := range mytes {
		fmt.Printf("TestEditDistance1: str1 = %s, str2 = %s, ed = %d\n", myte.str1, myte.str2, myte.ed)

		aed := EditDistanceParallel(myte.str1, myte.str2, v)

		if aed != myte.ed {
			t.Fatalf(`str1 = %q, str2 = %q, ed = %v, want = %v`, myte.str1, myte.str2, myte.ed, aed)
		}
	}

}

func TestEditDistanceParallel128(t *testing.T) {

	mytes := generateKnownCases()

	for _, myte := range mytes {
		fmt.Printf("TestEditDistance1: str1 = %s, str2 = %s, ed = %d\n", myte.str1, myte.str2, myte.ed)

		aed := EditDistanceParallel(myte.str1, myte.str2, 128)

		if aed != myte.ed {
			t.Fatalf(`str1 = %q, str2 = %q, ed = %v, want = %v`, myte.str1, myte.str2, myte.ed, aed)
		}
	}

}

//func TestEditDistanceParallel2(t *testing.T) {
//
//	mytes := generateKnownCases()
//
//	for _, myte := range mytes {
//		fmt.Printf("TestEditDistance1: str1 = %s, str2 = %s, ed = %d\n", myte.str1, myte.str2, myte.ed)
//
//		aed := EditDistanceParallel(myte.str1, myte.str2, 2)
//
//		if aed != myte.ed {
//			t.Fatalf(`str1 = %q, str2 = %q, ed = %v, want = %v`, myte.str1, myte.str2, myte.ed, aed)
//		}
//	}
//
//}

func TestEditDistanceParallelSeq(t *testing.T) {

	lenarr := []int{1, 16, 127, 128, 129, 1000, 5000, 10000}

	for _, alen := range lenarr {

		str1 := generateRandomString(alen)
		str2 := generateRandomString(alen)

		seqed := EditDistance(str1, str2)
		pared := EditDistanceParallel(str1, str2, 128)

		fmt.Printf("TestEditDistanceParallelSeq, len = %d, seq ed = %d, par ed = %d\n", alen, seqed, pared)

		if seqed != pared {
			t.Fatalf(`TestEditDistanceParallelSeq, len = %v, seq ed = %v, par ed = %v`, alen, seqed, pared)
		}

	}

}

func TestEditDistanceParallelTiles(t *testing.T) {

	lentiles := []int{-1, 0, 1, 2, 3, 4, 5, 10, 16, 32, 128, 1000}

	for _, tlen := range lentiles {

		alen := 1000
		str1 := generateRandomString(alen)
		str2 := generateRandomString(alen)

		seqed := EditDistance(str1, str2)
		pared := EditDistanceParallel(str1, str2, tlen)

		fmt.Printf("TestEditDistanceParallelTiles, len = %d, seq ed = %d, par ed = %d, tile size = %d\n", alen, seqed, pared, tlen)

		if seqed != pared {
			t.Fatalf(`TestEditDistanceParallelSeq, len = %v, seq ed = %v, par ed = %v, tile size = %v`, alen, seqed, pared, tlen)
		}

	}

}

func FuzzParallelSec(f *testing.F) {

	f.Add("abc", "bcd")

	f.Fuzz(func(t *testing.T, str1 string, str2 string) {

		rand.Seed(time.Now().UnixNano())
		tilesize := rand.Intn(len(str1) + len(str2) + 1)

		seqed := EditDistance(str1, str2)
		pared := EditDistanceParallel(str1, str2, tilesize)

		if seqed != pared {
			t.Errorf("str1: %q, str2: %q, seq ed: %v, par ed: %v, tile size: %v", str1, str2, seqed, pared, tilesize)
		}
	})

}
