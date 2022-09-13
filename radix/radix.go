package radix

import (
	"bufio"
	"fmt"

	"os"
	"strconv"
	"strings"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	goradix "github.com/hashicorp/go-immutable-radix"
)


type BGPRadixTree struct {
	Tree goradix.Tree
}  

func New(mrtFile string) *BGPRadixTree {
	log.Info("Starting to load BGP radix tree")
	r := CreateRadix(mrtFile)
	log.Info("Loaded BGP radix tree")
	return &BGPRadixTree{
		Tree: *r,
	}
} 

func (t *BGPRadixTree) GetI(ip uint32) int {
	ipKey := fmt.Sprintf("%0.32b", ip)
	m, v, _ := t.Tree.Root().LongestPrefix([]byte(ipKey))
	if m == nil {
		return -1
	}
	return v.(int)
}

func (t *BGPRadixTree) Get(ipStr string) int {
	ip, _ := util.IPStringToInt32(ipStr)
	ipKey := fmt.Sprintf("%0.32b", ip)
	m, v, _ := t.Tree.Root().LongestPrefix([]byte(ipKey))
	if m == nil {
		return -1
	}
	return v.(int)
}

func Insert(r *goradix.Tree, prefixStr string, prefixLenStr string, asn int) *goradix.Tree {
	prefix, err := util.IPStringToInt32(prefixStr)
	// prefix, err  := strconv.Atoi(ip)
	if err != nil {
		panic(err)
	}
	prefixLen, err := strconv.Atoi(prefixLenStr)
	if err != nil {
		panic(err)
	}
	ipPrefixStartKey := fmt.Sprintf("%0.32b", prefix)
	// b := false
	// Compute the length of the prefix given the network mask
	r, _, _ = r.Insert([]byte(ipPrefixStartKey[:prefixLen]), asn)
	return r
}

func CreateRadix(mrtFile string) *goradix.Tree {
	file, err := os.Open(mrtFile)
 
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
 
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	
	
 	// Create a tree
	r := goradix.New()
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ";")  {
			continue
		}
		// if i == 1 {
		// 	break
		// }
		i++

		tokens := strings.Split(line,"\t")
		prefixWithLen := tokens[0]
		asn, _ := strconv.Atoi(tokens[1])
		prefixTokens := strings.Split(prefixWithLen, "/")
		r = Insert(r, prefixTokens[0], prefixTokens[1], asn)


	}
 
	file.Close()

	return r
}


