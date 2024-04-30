package gopled

/*
#cgo CFLAGS: -O3 -mavx2 -I.
#include <stdlib.h>
#include "ctilekernel.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}




// ------------------------------------------------------------------------------------------------------------------

func editDistance(first, second string) int {
	lenFirst := len(first)
	lenSecond := len(second)

	if lenFirst == 0 || lenSecond == 0 {
		return lenFirst + lenSecond
	}

	dp := make([][]int, lenFirst+1)
	for i := range dp {
		dp[i] = make([]int, lenSecond+1)
	}

	for i := 0; i <= lenFirst; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lenSecond; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= lenFirst; i++ {
		for j := 1; j <= lenSecond; j++ {
			cost := 1
			if first[i-1] == second[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}

	return dp[lenFirst][lenSecond]
}

func computeBoundaryTile(first, second string, dp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int) {

	for i := 0; i < tileSize; i++ {
		for j := 0; j < tileSize; j++ {
			row := tileStartRow + i
			col := tileStartCol + j

			if row <= lenFirst && col <= lenSecond {

				cost := 1
				if first[row-1] == second[col-1] {
					cost = 0
				}
				dp[row][col] = min3(dp[row-1][col]+1, dp[row][col-1]+1, dp[row-1][col-1]+cost)

			}
		}
	}
}

func computeFullTileRegular(first, second string, dp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int) {

	for i := 0; i < tileSize; i++ {
		row := tileStartRow + i
		for j := 0; j < tileSize; j++ {
			col := tileStartCol + j
			cost := 1
			if first[row-1] == second[col-1] {
				cost = 0
			}
			dp[row][col] = min3(dp[row-1][col]+1, dp[row][col-1]+1, dp[row-1][col-1]+cost)
		}
	}
}

func computeFullTileRegular_large(first, second string, vdp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int, g_top_row, g_left_col []int) {

	
	// step 1: make the matrix for the tile, copy the input boundary
	// vdp: [first_half is bottom, second_half is right]
	// my tile (x,y): 1: (x, y-1)[first_half] 2: (x-1, y)[second_half]; 3: (x-1,y-1)[first_half][tilesize -1]
	// out of boundary -- go to the gtop row and gleft col.
	
	// step 2: fill the matrix

	// step 3: copy the output boundary

	
	
	for i := 0; i < tileSize; i++ {
		row := tileStartRow + i
		for j := 0; j < tileSize; j++ {
			col := tileStartCol + j
			cost := 1
			if first[row-1] == second[col-1] {
				cost = 0
			}
			vdp[row][col] = min3(vdp[row-1][col]+1, vdp[row][col-1]+1, vdp[row-1][col-1]+cost)
		}
	}
}






func computeFullTileC(first, second string, dp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int) {


	first_substr := first[tileStartRow - 1 : tileStartRow - 1 + tileSize]
	second_substr := second[tileStartCol - 1 : tileStartCol - 1 + tileSize]

	c_first_substr := C.CString(first_substr)
	defer C.free(unsafe.Pointer(c_first_substr))

	c_second_substr := C.CString(second_substr)
	defer C.free(unsafe.Pointer(c_second_substr))

	
	top_row := tileStartRow - 1
	left_col := tileStartCol - 1


	rowbuf := make([]C.int, (tileSize + 1))
	colbuf := make([]C.int, (tileSize + 1))

	for i := 0; i < tileSize + 1; i++ {
		rowbuf[i] = C.int(dp[top_row][left_col + i])
	}

	for i := 0; i < tileSize + 1; i++ {
		colbuf[i] = C.int(dp[top_row + i][left_col])
	}

	
	
	C.c_handle_tile(
		C.int(tileSize),
		(*C.int)(unsafe.Pointer(&rowbuf[0])),
		(*C.int)(unsafe.Pointer(&colbuf[0])),
		c_first_substr,
		c_second_substr)
	

	lastrow := tileStartRow + tileSize - 1
	lastcol := tileStartCol + tileSize - 1
	
	for i := 0; i < tileSize; i++ {
		dp[lastrow][tileStartCol + i] = int(rowbuf[i])
	}

	for i := 0; i < tileSize; i++ {
		dp[tileStartRow + i][lastcol] = int(colbuf[i])
	}
	

}





func editDistanceParallel(first, second string, tileSize int, useavx bool) int {

	lenFirst, lenSecond := len(first), len(second)

	if lenFirst == 0 || lenSecond == 0 {
		return lenFirst + lenSecond
	}

	dp := make([][]int, lenFirst+1)
	for i := range dp {
		dp[i] = make([]int, lenSecond+1)
	}

	for i := 0; i <= lenFirst; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lenSecond; j++ {
		dp[0][j] = j
	}

	var wg sync.WaitGroup

	m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize
	totalWavefronts := n + m - 1
	
	for wave := 0; wave < totalWavefronts; wave++ {
		minStart := max(0, wave-n+1)
		maxStart := min(wave, m-1)

		for start := minStart; start <= maxStart; start++ {

			tileStartRow := start*tileSize + 1
			tileStartCol := (wave-start)*tileSize + 1

			wg.Add(1)

			isBoundaryTile := 0
			if tileStartRow+tileSize > lenFirst || tileStartCol+tileSize > lenSecond {
				isBoundaryTile = 1
			}

			go func(p1, p2 string, p3 [][]int, p4, p5, p6, p7, p8 int) {

				defer wg.Done()

				if isBoundaryTile == 1 {
					computeBoundaryTile(p1, p2, p3, p4, p5, p6, p7, p8)
				} else {

					if useavx && tileSize > 64 && (tileSize&(tileSize-1)) == 0 {
						computeFullTileC(p1, p2, p3, p4, p5, p6, p7, p8)
					} else {
						computeFullTileRegular(p1, p2, p3, p4, p5, p6, p7, p8)
					}
				}

			}(first, second, dp, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond)

		}

		wg.Wait()
	}

	return dp[lenFirst][lenSecond]
}



func editDistanceParallel_large(first, second string, tileSize int, useavx bool) int {

	lenFirst, lenSecond := len(first), len(second)

	if lenFirst == 0 || lenSecond == 0 {
		return lenFirst + lenSecond
	}

	//dp := make([][]int, lenFirst+1)
	//for i := range dp {
	//	dp[i] = make([]int, lenSecond+1)
	//}

	//for i := 0; i <= lenFirst; i++ {
	//	dp[i][0] = i
	//}
	//for j := 0; j <= lenSecond; j++ {
	//	dp[0][j] = j
	//}


	g_top_row := make([]int, lenSecond + 1)
	for j := 0; j <= lenSecond; j++ {
		g_top_row[j] = j
	}
	
	g_left_col := make([]int, lenFirst + 1)
	for i := 0; i <=lenFirst; i++{
		g_left_col[i] = i
	}

	m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	vdp := make([][]int, m * n)
	for i := range vdp {
		vdp[i] = make([]int, 2 * tileSize)
	}

	

	var wg sync.WaitGroup

	totalWavefronts := n + m - 1
	
	for wave := 0; wave < totalWavefronts; wave++ {
		minStart := max(0, wave-n+1)
		maxStart := min(wave, m-1)

		for start := minStart; start <= maxStart; start++ {

			tileStartRow := start*tileSize + 1
			tileStartCol := (wave-start)*tileSize + 1

			wg.Add(1)

			isBoundaryTile := 0
			if tileStartRow+tileSize > lenFirst || tileStartCol+tileSize > lenSecond {
				isBoundaryTile = 1
			}

			go func(p1, p2 string, p3 [][]int, p4, p5, p6, p7, p8 int, p9, p10 []int) {

				defer wg.Done()

				if isBoundaryTile == 1 {
					computeBoundaryTile_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)
				} else {
					computeFullTileRegular_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)

					//if useavx && tileSize > 64 && (tileSize&(tileSize-1)) == 0 {
					//	computeFullTileC_large(p1, p2, p3, p4, p5, p6, p7, p8)
					//} else {
					//	computeFullTileRegular_large(p1, p2, p3, p4, p5, p6, p7, p8)
					//}
				}

			}(first, second, vdp, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond, g_top_row, g_left_col)

		}

		wg.Wait()
	}

	//return dp[lenFirst][lenSecond]
	return vdp[m*n - 1][tileSize - 1]
}








// ------------------------------------------------------------------------------------------------------------------



func EditDistance(first, second string) int {

	return editDistance(first, second)

}

func EditDistanceParallel(first string, second string, tilesize ...int) int {

	var tsv int
	if len(tilesize) > 0 {
		tsv = tilesize[0]
		if tsv < 1 {
			tsv = 1
		}
	} else {
		tsv = 1024
	}

	
	useavx := (C.c_check_avx2_support() == 1)
		
	return editDistanceParallel(first, second, tsv, useavx)

}
