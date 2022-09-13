package survey
/*
#cgo LDFLAGS: -lcperm
#include <cperm.h>
#include <string.h>

typedef struct cperm_t cperm_t; 
cperm_t * create(uint32_t perm_size){
	uint32_t KEYLEN = 16;
	uint8_t  key[KEYLEN];
	memset(key, 0, KEYLEN);
    struct cperm_t* perm = cperm_create(perm_size, PERM_MODE_CYCLE,
										PERM_CIPHER_RC5, key, KEYLEN);
	return perm;
}

uint32_t next(cperm_t * cperm){
	uint32_t val;
	cperm_next(cperm, &val);
	return val;
}

void destroy(cperm_t * cperm){
	cperm_destroy(cperm);
}
*/
import "C"

func CPermCreate(permSize uint32 ) *C.cperm_t{
	return C.create(C.uint32_t(permSize))
}

func CPermNext(cperm *C.cperm_t) uint32 {
    return uint32(C.next(cperm))
}

func CPermDestroy(cperm *C.cperm_t){
	C.cperm_destroy(cperm)
}