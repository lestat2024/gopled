
#ifndef _LESTAT_BATCHMIN_H_

#define _LESTAT_BATCHMIN_H_

int c_check_avx2_support();

void c_handle_tile(int ts,
		    int* rowbuf,
		    int* colbuf,
		    const char* first_str,
		     const char* second_str);


void __eddpkernel(int ts,
		const char* first_str,
		const char* second_str,
		int* wholematrix);

void __eddpkernel_avx2(int ts,
		const char* first_str,
		const char* second_str,
		int* wholematrix);



#endif
