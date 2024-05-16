#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <immintrin.h> 

#include "ctilekernel.h"


#define BEIBA 16


int c_check_avx2_support() {
    uint32_t eax, ebx, ecx, edx;
    eax = 7; ecx = 0; 

    __asm__ __volatile__ (
        "cpuid"
        : "=a"(eax), "=b"(ebx), "=c"(ecx), "=d"(edx)
        : "a"(eax), "c"(ecx)
        : );

    return (ebx >> 5) & 1; 
}


inline int __p_min3(int a, int b, int c) {
    int min = a;
    if (b < min) min = b;
    if (c < min) min = c;
    return min;
}


void c_handle_tile(int ts,
		    int* rowbuf,
		    int* colbuf,
		    const char* first_str,
		     const char* second_str)
{
  int md = ts + 1;
  int* wholematrix = (int*)malloc((md*md)*sizeof(int));
  
  for(int i = 0; i < md; i++)
    {
      wholematrix[0 * md + i] = rowbuf[i];
    }
  for(int i = 0; i < md; i++)
    {
      wholematrix[i * md + 0] = colbuf[i];
    }


  //__eddpkernel(ts, first_str, second_str, wholematrix);
  __eddpkernel_avx2(ts, first_str, second_str, wholematrix);

  for(int i = 0; i < ts; i++)
    {
      rowbuf[i] = wholematrix[ts * md + (i + 1)];
    }
  for(int i = 0; i < ts; i++)
    {
      colbuf[i] = wholematrix[(i + 1) * md + ts];
    }
  
  free(wholematrix);  
}


void c_handle_tile_vdp(int ts,
		    int* uprow,
			int* leftcol,
			int diagvalue,
		    int* newboundary,
		    const char* first_str,
		     const char* second_str)
{
  int md = ts + 1;
  int* wholematrix = (int*)malloc((md*md)*sizeof(int));
  
  for(int i = 1; i < md; i++)
    {
      wholematrix[0 * md + i] = uprow[i-1];
    }
  for(int i = 1; i < md; i++)
    {
      wholematrix[i * md + 0] = leftcol[i-1];
    }

	wholematrix[0] = diagvalue;


  //__eddpkernel(ts, first_str, second_str, wholematrix);
  __eddpkernel_avx2(ts, first_str, second_str, wholematrix);

  for(int i = 0; i < ts; i++)
    {
      newboundary[i] = wholematrix[ts * md + (i + 1)];
    }
  for(int i = 0; i < ts; i++)
    {
      newboundary[i+ts] = wholematrix[(i + 1) * md + ts];
    }
  
  free(wholematrix);  
}



void __eddpkernel(int ts,
		const char* first_str,
		const char* second_str,
		int* wholematrix)
{
  int md = ts + 1;
  
  for(int i = 0; i < ts; i++)
    {
      for(int j = 0; j < ts; j++)
	{
	  int row = i + 1;
	  int col = j + 1;

	  int cost = ((first_str[i] == second_str[j]) ? 0 : 1);
	  
	  wholematrix[row * md + col] = __p_min3(
						  wholematrix[(row-1) * md + col] +1,
						  wholematrix[row * md + (col-1)]+1,
						  wholematrix[(row-1) * md + (col-1)] + cost
						 );
	}
    }
}

void __eddpkernel_avx2(int ts,
		const char* first_str,
		const char* second_str,
		int* wholematrix)
{

  int md = ts + 1;
  int costarr[BEIBA];
  int* wavespace = malloc((BEIBA+1)*(ts-BEIBA+1)*sizeof(int));
  
  for(int rs = 0; rs < (ts / BEIBA); rs++)
    {
      int toprow = rs * BEIBA;

      // STEP 1: Fill left triangle
      for(int ri = 0; ri < BEIBA; ri++)
	{
	  int rowt = toprow + ri;
	  int colcount = BEIBA - ri - 1;
	  for(int colt = 0; colt < colcount; colt++)
	    {
	      int row = rowt + 1;
	      int col = colt + 1;
	      int cost = ((first_str[row-1] == second_str[col-1]) ? 0 : 1);
	      wholematrix[row * md + col] = __p_min3(
						     wholematrix[(row-1) * md + col] +1,
						     wholematrix[row * md + (col-1)]+1,
						     wholematrix[(row-1) * md + (col-1)] + cost
						     );

	    }
	}

      // STEP 2 : execute middle waves
      int gsr = 1 + toprow;
      
      // STEP 2.1: copy last column
      for(int ci = BEIBA - 1; ci < ts - 1; ci++)
	{
	  int gsc = 1 + ci;
	  wavespace[(ci - (BEIBA - 1))*(BEIBA+1)+(BEIBA)] = wholematrix[(gsr-1)*(md)+(gsc+1)];
	}
      
      // STEP 2.2: calculate and fill the top 2 rows
      for(int ci = BEIBA - 1; ci < BEIBA + 1; ci++)
	{
	  int gsc = 1 + ci;

	  for(int wi = 0; wi < BEIBA; wi++)
	    {
	      int arow = gsr + wi;
	      int acol = gsc - wi;
	      
	      int lv = wholematrix[(arow) * md + (acol-1)] + 1;
	      int tv = wholematrix[(arow-1) * md + (acol)] + 1;
	      int cost = ((first_str[arow-1] == second_str[acol-1]) ? 0 : 1);
	      int dv = wholematrix[(arow-1) * md + (acol-1)] + cost;
	      int retv = __p_min3(lv, tv, dv);

	      wholematrix[arow * md + acol] = retv;
	      wavespace[(ci - (BEIBA - 1))*(BEIBA+1) + ((BEIBA-1)-wi)] = retv;
	    }

	}


      // STEP 2.3: waves going with AVX2
      int startwave = 2;

      // BEIBA/8 ---> how many loops we need (16/8 = 2!)
      __m256i lm;
      __m256i tm;
      __m256i dm;
      lm = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (0)]);
      tm = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (1)]);
      dm = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-2)*(BEIBA+1) + (1)]);

      __m256i lm2;
      __m256i tm2;
      __m256i dm2;
      lm2 = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (0 + 8 * 1)]);
      tm2 = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (1 + 8 * 1)]);
      dm2 = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-2)*(BEIBA+1) + (1 + 8 * 1)]);

      for(int ci = BEIBA + 1; ci < ts; ci++)
	{
	  //STEP 2.3.1: calcuate cost array
	  int gsc = 1 + ci;
	  for(int wi = 0; wi < BEIBA; wi++)
	    {
	      int arow = gsr + wi;
	      int acol = gsc - wi;
	      costarr[(BEIBA-1)-wi] = ((first_str[arow-1] == second_str[acol-1]) ? 0 : 1);
	    }

	  //STEP 2.3.2: prepare vectors
	  __m256i cost_vec = _mm256_loadu_si256((__m256i*)&costarr[0]);
	  __m256i cost_vec2 = _mm256_loadu_si256((__m256i*)&costarr[0 + 8 * 1]);

	  __m256i dmpc = _mm256_add_epi32(dm, cost_vec);
	  __m256i dmpc2 = _mm256_add_epi32(dm2, cost_vec2);
	  
	  __m256i one_vec = _mm256_set1_epi32(1);
	  __m256i lmp1 = _mm256_add_epi32(lm, one_vec);
	  __m256i tmp1 = _mm256_add_epi32(tm, one_vec);


	  __m256i lmp12 = _mm256_add_epi32(lm2, one_vec);
	  __m256i tmp12 = _mm256_add_epi32(tm2, one_vec);


	  //STEP 2.3.3: min and store
	  __m256i min_temp = _mm256_min_epi32(lmp1, tmp1);
	  __m256i min_temp2 = _mm256_min_epi32(lmp12, tmp12);
	  
	  __m256i nr = _mm256_min_epi32(min_temp, dmpc);
	  __m256i nr2 = _mm256_min_epi32(min_temp2, dmpc2);

	  _mm256_storeu_si256((__m256i*)&wavespace[(startwave)*(BEIBA+1) + (0)], nr);
	  _mm256_storeu_si256((__m256i*)&wavespace[(startwave)*(BEIBA+1) + (0 + 8 * 1)], nr2);

	  //STEP 2.3.4: forward to next wave
	  startwave += 1;
	  lm = nr;
	  dm = tm;  
	  tm = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (1)]);

	  lm2 = nr2;
	  dm2 = tm2;
	  tm2 = _mm256_loadu_si256((__m256i*)&wavespace[(startwave-1)*(BEIBA+1) + (1 + 8 * 1)]);
	}
      
      //STEP 2.4: copy back the first column in wavespace
      for(int ci = BEIBA - 1; ci < ts; ci++)
	{
	  wholematrix[(gsr+(BEIBA-1))*(md)+(ci-(BEIBA-1) + 1)] = wavespace[(ci-(BEIBA-1))*(BEIBA+1) + (0)];
	}
      
      //STEP 2.5: copy back last 2 rows
      for(int ci = ts - 2; ci < ts; ci++)
	{
	  int gsc = 1 + ci;
	  for(int wi = 0; wi < BEIBA; wi++)
	    {
	      int arow = gsr + wi;
	      int acol = gsc - wi;

	      wholematrix[arow * md + acol] = wavespace[(ci - (BEIBA - 1))*(BEIBA+1) + ((BEIBA-1)-wi)];
	    }
	}
     
      //STEP 3: right triangle
      for(int ri = 0; ri < BEIBA; ri++)
	{
	  int rowt = toprow + ri;
	  int colcount = ri;
	  for(int colt = 0; colt < colcount; colt++)
	    {
	      int row = 1 + rowt;
	      int col = 1 + ts - colcount + colt;
	      int cost = ((first_str[row-1] == second_str[col-1]) ? 0 : 1);
	      wholematrix[row * md + col] = __p_min3(
						     wholematrix[(row-1) * md + col] +1,
						     wholematrix[row * md + (col-1)]+1,
						     wholematrix[(row-1) * md + (col-1)] + cost
						     );
	      
	    }
	}
       
    }

  free(wavespace);  
}
