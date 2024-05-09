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

func computeTileRegular_large(first, second string, vdp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int, g_top_row, g_left_col []int) {

	// step 1: make the matrix for the tile, copy the input boundary
	// vdp: [first_half is bottom, second_half is right]
	// my tile (x,y): 1: (x, y-1)[first_half] 2: (x-1, y)[second_half]; 3: (x-1,y-1)[first_half][tilesize -1]
	// out of boundary -- go to the gtop row and gleft col.

	// m is height, n is width
	m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	//fmt.Printf("m = %d, n= %d, tileSize = %d, tileStartRow = %d, tileStartCol = %d\n", m, n, tileSize, tileStartRow, tileStartCol)

	tileMatrix := make([][]int, tileSize+1)
	for i := range tileMatrix {
		tileMatrix[i] = make([]int, tileSize+1)
	}

	my_x := (tileStartCol - 1) / tileSize
	my_y := (tileStartRow - 1) / tileSize

	my_up := my_y - 1
	my_left := my_x - 1

	tileMatrix[0][0] = -1

	// copy top row
	if my_up >= 0 {
		//copy from up tile
		for i := 1; i <= tileSize; i++ {
			up_tile_idx := (my_up)*n + (my_x)
			tileMatrix[0][i] = vdp[up_tile_idx][i-1]
		}

	} else {
		//copy from boundary array
		copy_length := tileSize
		if (tileStartCol-1)+tileSize > lenSecond {
			copy_length = lenSecond - tileStartCol + 1
		}

		for i := 1; i <= copy_length; i++ {
			//fmt.Printf("copy_length = %d, i = %d\n", copy_length, i)
			tileMatrix[0][i] = g_top_row[tileStartCol+(i-1)] // wrong!!!!
		}

		tileMatrix[0][0] = g_top_row[tileStartCol-1]
	}

	// copy left col
	if my_left >= 0 {

		for i := 1; i <= tileSize; i++ {
			left_tile_idx := (my_y)*n + my_left
			tileMatrix[i][0] = vdp[left_tile_idx][tileSize+(i-1)]
		}

	} else {
		copy_length := tileSize
		if (tileStartRow-1)+tileSize > lenFirst {
			copy_length = lenFirst - tileStartRow + 1
		}
		for i := 1; i <= copy_length; i++ {
			tileMatrix[i][0] = g_left_col[tileStartRow+(i-1)]
		}

		tileMatrix[0][0] = g_left_col[tileStartRow-1]
	}

	if tileMatrix[0][0] == -1 {
		diag_tile_idx := (my_up)*n + my_left
		tileMatrix[0][0] = vdp[diag_tile_idx][tileSize-1]
	}

	// step 2: fill the matrix

	for i := 0; i < tileSize; i++ {
		for j := 0; j < tileSize; j++ {
			row := tileStartRow + i
			col := tileStartCol + j

			if row <= lenFirst && col <= lenSecond {

				cost := 1
				if first[row-1] == second[col-1] {
					cost = 0
				}
				//dp[row][col] = min3(dp[row-1][col]+1, dp[row][col-1]+1, dp[row-1][col-1]+cost)

				local_tile_row := i + 1
				local_tile_col := j + 1

				tileMatrix[local_tile_row][local_tile_col] = min3(
					tileMatrix[local_tile_row-1][local_tile_col]+1,
					tileMatrix[local_tile_row][local_tile_col-1]+1,
					tileMatrix[local_tile_row-1][local_tile_col-1]+cost)

			}
		}
	}

	// step 3: copy the output boundary

	bottom_row_id := tileSize
	if (tileStartRow-1)+tileSize > lenFirst {
		bottom_row_id = lenFirst - tileStartRow + 1
	}

	right_col_id := tileSize
	if (tileStartCol-1)+tileSize > lenSecond {
		right_col_id = lenSecond - tileStartCol + 1
	}

	this_tile_idx := (my_y)*n + my_x

	//fmt.Printf("m = %d, n= %d ,my_x = %d,my_y = %d\n", m, n, my_x, my_y)
	//fmt.Printf("bottom_row_id, right_col_id: %d, %d\n", bottom_row_id, right_col_id)
	//fmt.Printf("this_tile_idx = %d-----\n", this_tile_idx)
	for i := 0; i < tileSize; i++ {
		vdp[this_tile_idx][i] = tileMatrix[bottom_row_id][i+1]
		vdp[this_tile_idx][tileSize+i] = tileMatrix[i+1][right_col_id]
	}

	// step 4: if this is the right tile, we must copy the last value to the middle position of vdp[last_tile]

	if my_x == (n-1) && my_y == (m-1) {
		vdp[this_tile_idx][tileSize-1] = tileMatrix[bottom_row_id][right_col_id]
	}

}

func computeTileFull_large(first, second string, vdp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int, g_top_row, g_left_col []int) {

	// step 1: make the matrix for the tile, copy the input boundary
	// vdp: [first_half is bottom, second_half is right]
	// my tile (x,y): 1: (x, y-1)[first_half] 2: (x-1, y)[second_half]; 3: (x-1,y-1)[first_half][tilesize -1]
	// out of boundary -- go to the gtop row and gleft col.

	// m is height, n is width
	//m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	//fmt.Printf("m = %d, n= %d, tileSize = %d, tileStartRow = %d, tileStartCol = %d\n", m, n, tileSize, tileStartRow, tileStartCol)

	tileMatrix := make([][]int, tileSize+1)
	for i := range tileMatrix {
		tileMatrix[i] = make([]int, tileSize+1)
	}

	my_x := (tileStartCol - 1) / tileSize
	my_y := (tileStartRow - 1) / tileSize

	my_up := my_y - 1
	my_left := my_x - 1

	tileMatrix[0][0] = -1

	// copy top row
	if my_up >= 0 {
		//copy from up tile
		for i := 1; i <= tileSize; i++ {
			up_tile_idx := (my_up)*n + (my_x)
			tileMatrix[0][i] = vdp[up_tile_idx][i-1]
		}

	} else {
		//copy from boundary array
		copy_length := tileSize
		for i := 1; i <= copy_length; i++ {
			//fmt.Printf("copy_length = %d, i = %d\n", copy_length, i)
			tileMatrix[0][i] = g_top_row[tileStartCol+(i-1)] // wrong!!!!
		}

		tileMatrix[0][0] = g_top_row[tileStartCol-1]
	}

	// copy left col
	if my_left >= 0 {

		for i := 1; i <= tileSize; i++ {
			left_tile_idx := (my_y)*n + my_left
			tileMatrix[i][0] = vdp[left_tile_idx][tileSize+(i-1)]
		}

	} else {
		copy_length := tileSize
		for i := 1; i <= copy_length; i++ {
			tileMatrix[i][0] = g_left_col[tileStartRow+(i-1)]
		}

		tileMatrix[0][0] = g_left_col[tileStartRow-1]
	}

	if tileMatrix[0][0] == -1 {
		diag_tile_idx := (my_up)*n + my_left
		tileMatrix[0][0] = vdp[diag_tile_idx][tileSize-1]
	}

	// step 2: fill the matrix

	for i := 0; i < tileSize; i++ {
		for j := 0; j < tileSize; j++ {
			row := tileStartRow + i
			col := tileStartCol + j

			cost := 1
			if first[row-1] == second[col-1] {
				cost = 0
			}
			//dp[row][col] = min3(dp[row-1][col]+1, dp[row][col-1]+1, dp[row-1][col-1]+cost)

			local_tile_row := i + 1
			local_tile_col := j + 1

			tileMatrix[local_tile_row][local_tile_col] = min3(
				tileMatrix[local_tile_row-1][local_tile_col]+1,
				tileMatrix[local_tile_row][local_tile_col-1]+1,
				tileMatrix[local_tile_row-1][local_tile_col-1]+cost)

		}
	}

	// step 3: copy the output boundary

	bottom_row_id := tileSize

	right_col_id := tileSize

	this_tile_idx := (my_y)*n + my_x

	//fmt.Printf("m = %d, n= %d ,my_x = %d,my_y = %d\n", m, n, my_x, my_y)
	//fmt.Printf("bottom_row_id, right_col_id: %d, %d\n", bottom_row_id, right_col_id)
	//fmt.Printf("this_tile_idx = %d-----\n", this_tile_idx)
	for i := 0; i < tileSize; i++ {
		vdp[this_tile_idx][i] = tileMatrix[bottom_row_id][i+1]
		vdp[this_tile_idx][tileSize+i] = tileMatrix[i+1][right_col_id]
	}

}

func computeTileFull_largeCX(first, second string, vdp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int, g_top_row, g_left_col []int) {

	// step 1: make the matrix for the tile, copy the input boundary
	// vdp: [first_half is bottom, second_half is right]
	// my tile (x,y): 1: (x, y-1)[first_half] 2: (x-1, y)[second_half]; 3: (x-1,y-1)[first_half][tilesize -1]
	// out of boundary -- go to the gtop row and gleft col.

	// m is height, n is width
	//m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	my_x := (tileStartCol - 1) / tileSize
	my_y := (tileStartRow - 1) / tileSize

	my_up := my_y - 1
	my_left := my_x - 1

	rowbuf := make([]C.int, (tileSize + 1))
	colbuf := make([]C.int, (tileSize + 1))

	rowbuf[0] = -1
	colbuf[0] = -1

	// copy top row
	if my_up >= 0 {
		//copy from up tile
		for i := 1; i <= tileSize; i++ {
			up_tile_idx := (my_up)*n + (my_x)
			//tileMatrix[0][i] = vdp[up_tile_idx][i-1]
			rowbuf[i] = C.int(vdp[up_tile_idx][i-1])
		}

	} else {
		//copy from boundary array
		copy_length := tileSize
		for i := 1; i <= copy_length; i++ {
			//fmt.Printf("copy_length = %d, i = %d\n", copy_length, i)
			//tileMatrix[0][i] = g_top_row[tileStartCol+(i-1)]
			rowbuf[i] = C.int(g_top_row[tileStartCol+(i-1)])
		}

		//tileMatrix[0][0] = g_top_row[tileStartCol-1]
		rowbuf[0] = C.int(g_top_row[tileStartCol-1])
		colbuf[0] = rowbuf[0]
	}

	// copy left col
	if my_left >= 0 {

		for i := 1; i <= tileSize; i++ {
			left_tile_idx := (my_y)*n + my_left
			//tileMatrix[i][0] = vdp[left_tile_idx][tileSize+(i-1)]
			colbuf[i] = C.int(vdp[left_tile_idx][tileSize+(i-1)])
		}

	} else {
		copy_length := tileSize
		for i := 1; i <= copy_length; i++ {
			//tileMatrix[i][0] = g_left_col[tileStartRow+(i-1)]
			colbuf[i] = C.int(g_left_col[tileStartRow+(i-1)])
		}

		//tileMatrix[0][0] = g_left_col[tileStartRow-1]
		colbuf[0] = C.int(g_left_col[tileStartRow-1])
		rowbuf[0] = colbuf[0]
	}

	if rowbuf[0] == -1 && colbuf[0] == -1 {
		diag_tile_idx := (my_up)*n + my_left
		rowbuf[0] = C.int(vdp[diag_tile_idx][tileSize-1])
		colbuf[0] = rowbuf[0]
	}

	// step 2: fill the matrix

	first_substr := first[tileStartRow-1 : tileStartRow-1+tileSize]
	second_substr := second[tileStartCol-1 : tileStartCol-1+tileSize]

	c_first_substr := C.CString(first_substr)
	defer C.free(unsafe.Pointer(c_first_substr))

	c_second_substr := C.CString(second_substr)
	defer C.free(unsafe.Pointer(c_second_substr))

	C.c_handle_tile(
		C.int(tileSize),
		(*C.int)(unsafe.Pointer(&rowbuf[0])),
		(*C.int)(unsafe.Pointer(&colbuf[0])),
		c_first_substr,
		c_second_substr)

	// step 3: copy the output boundary

	this_tile_idx := (my_y)*n + my_x

	//fmt.Printf("m = %d, n= %d ,my_x = %d,my_y = %d\n", m, n, my_x, my_y)
	//fmt.Printf("bottom_row_id, right_col_id: %d, %d\n", bottom_row_id, right_col_id)
	//fmt.Printf("this_tile_idx = %d-----\n", this_tile_idx)
	for i := 0; i < tileSize; i++ {
		vdp[this_tile_idx][i] = int(rowbuf[i])
		vdp[this_tile_idx][tileSize+i] = int(colbuf[i])
	}

}

func computeFullTileC(first, second string, dp [][]int, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond int) {

	first_substr := first[tileStartRow-1 : tileStartRow-1+tileSize]
	second_substr := second[tileStartCol-1 : tileStartCol-1+tileSize]

	c_first_substr := C.CString(first_substr)
	defer C.free(unsafe.Pointer(c_first_substr))

	c_second_substr := C.CString(second_substr)
	defer C.free(unsafe.Pointer(c_second_substr))

	top_row := tileStartRow - 1
	left_col := tileStartCol - 1

	rowbuf := make([]C.int, (tileSize + 1))
	colbuf := make([]C.int, (tileSize + 1))

	for i := 0; i < tileSize+1; i++ {
		rowbuf[i] = C.int(dp[top_row][left_col+i])
	}

	for i := 0; i < tileSize+1; i++ {
		colbuf[i] = C.int(dp[top_row+i][left_col])
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
		dp[lastrow][tileStartCol+i] = int(rowbuf[i])
	}

	for i := 0; i < tileSize; i++ {
		dp[tileStartRow+i][lastcol] = int(colbuf[i])
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
			if (tileStartRow-1)+tileSize > lenFirst || (tileStartCol-1)+tileSize > lenSecond {
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

	g_top_row := make([]int, lenSecond+1)
	for j := 0; j <= lenSecond; j++ {
		g_top_row[j] = j
	}

	g_left_col := make([]int, lenFirst+1)
	for i := 0; i <= lenFirst; i++ {
		g_left_col[i] = i
	}

	m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	vdp := make([][]int, m*n)
	for i := range vdp {
		vdp[i] = make([]int, 2*tileSize)
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

			go func(p1, p2 string, p3 [][]int, p4, p5, p6, p7, p8 int, p9, p10 []int) {

				defer wg.Done()

				computeTileRegular_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)

			}(first, second, vdp, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond, g_top_row, g_left_col)

		}

		wg.Wait()
	}

	return vdp[m*n-1][tileSize-1]
}

func editDistanceParallel_largeX(first, second string, tileSize int, useavx bool) int {

	lenFirst, lenSecond := len(first), len(second)

	if lenFirst == 0 || lenSecond == 0 {
		return lenFirst + lenSecond
	}

	g_top_row := make([]int, lenSecond+1)
	for j := 0; j <= lenSecond; j++ {
		g_top_row[j] = j
	}

	g_left_col := make([]int, lenFirst+1)
	for i := 0; i <= lenFirst; i++ {
		g_left_col[i] = i
	}

	m := ((lenFirst) + tileSize - 1) / tileSize
	n := ((lenSecond) + tileSize - 1) / tileSize

	vdp := make([][]int, m*n)
	for i := range vdp {
		vdp[i] = make([]int, 2*tileSize)
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

			isNonFullTile := false
			if (tileStartRow-1)+tileSize > lenFirst || (tileStartCol-1)+tileSize > lenSecond {
				isNonFullTile = true
			}

			go func(p1, p2 string, p3 [][]int, p4, p5, p6, p7, p8 int, p9, p10 []int) {

				defer wg.Done()

				if isNonFullTile {
					computeTileRegular_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)
				} else {
					if useavx && tileSize > 64 && (tileSize&(tileSize-1)) == 0 {
						computeTileFull_largeCX(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)
						//computeTileFull_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)
					} else {
						computeTileFull_large(p1, p2, p3, p4, p5, p6, p7, p8, p9, p10)
					}
				}

			}(first, second, vdp, tileStartRow, tileStartCol, tileSize, lenFirst, lenSecond, g_top_row, g_left_col)

		}

		wg.Wait()
	}

	return vdp[m*n-1][tileSize-1]
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

	return editDistanceParallel_largeX(first, second, tsv, useavx)

}
