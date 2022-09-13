package ipoptions

import (
	"bytes"

	"github.com/NEU-SNS/ReverseTraceroute/log"
)

// TimestampAddrToString returns a TS string ready to go into a Ping{} structure.
func TimestampAddrToString(probe [] string) string {
	var tsString bytes.Buffer
	tsString.WriteString("tsprespec=")
	for i, p := range probe {
		// if i == 0 {
		// 	continue
		// }
		tsString.WriteString(p)
		if len(probe)-1 != i {
			tsString.WriteString(",")
		}
	}
	tss := tsString.String()
	log.Debug("tss string: ", tss)
	return tss
}