package radix

import (
	"fmt"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/stretchr/testify/assert"
)


func TestCreateRadix(t *testing.T) {
	
	before := time.Now()
	r :=  CreateRadix("/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/resources/bgpdumps/latest")
	elapsed := time.Since(before)
	print("Elapsed to create radix: %d seconds", elapsed.Seconds())
	// Find the longest prefix match
	ip, _ := util.IPStringToInt32("1.0.0.1")
	ipKey := fmt.Sprintf("%0.32b", ip)
	_, v, _ := r.Root().LongestPrefix([]byte(ipKey))
	assert.Equal(t, v, 13335, "Not the right AS")
	prefix := "1.0.0.0"
	prefixLen := "25"
	r     = Insert(r, prefix, prefixLen, 1)
	bs := []byte(ipKey)
	_, v, _ = r.Root().LongestPrefix(bs)
	assert.Equal(t, v, 1, "Not the right AS")
	
}