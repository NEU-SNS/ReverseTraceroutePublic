package alias

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/NEU-SNS/ReverseTraceroute/util"
)

func ExtractAliasCandidates(labelFile string, ofile string, groundTruthIPsFile string) error {
	cmd := fmt.Sprintf("python3 " + 
	"/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/survey/alias/alias_resolution.py %s %s %s",
	 labelFile, ofile, groundTruthIPsFile)
	_, err := util.RunCMD(cmd)	
	return err
}

func AliasResolution(ifile string, ofile string, isDry bool) error {
	cmd := fmt.Sprintf("python3 " + 
	"/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/midar-api/midar-api.py " + 
	"--key=9787efbf8a0d6c57a9a82d42b984174b upload %s", ifile)
	print(cmd)
	var err error
	out := ""
	if isDry {
		out = "URL: https://vela.caida.org/midar-api/upload?key=9787efbf8a0d6c57a9a82d42b984174b"  +
		"HTTP response code: 200 " +
		"HTTP response body: {'result':'success','result_id':100}" +
		" " +
		"result ID: 100"
	} else  {
		out, err = util.RunCMD(cmd)
		if err != nil {
			panic(err)
		}
	}
	// Parse the MIDAR output to retrieve the ID
	tokens := strings.Split(out, "result ID:")
	ID, err := strconv.Atoi(strings.TrimSpace(tokens[1]))
	if err != nil {
		return err
	}
	IDStr := strconv.Itoa(ID)
	// Write the ID into a file 
	err = ioutil.WriteFile(ofile, [] byte(IDStr), 0644)
    if err != nil {
		return err
	}

	return nil
}