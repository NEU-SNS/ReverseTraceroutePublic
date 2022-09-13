package clustermap

import (
	"fmt"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

// ClusterMap maps IP addresses to cluster ids
type ClusterMap struct {
	cs types.ClusterSource
	ca types.Cache
}

func (cm ClusterMap) fetchCluster(s string) string {

	ipint, _ := util.IPStringToInt32(s)
	cluster, err := cm.cs.GetClusterIDByIP(ipint)
	if err != nil {
		log.Debug(err)
		cm.ca.Set("CM_"+s, s, time.Hour*6)
		return s
	}
	clusters := fmt.Sprintf("%d", cluster)
	cm.ca.Set("CM_"+s, clusters, time.Hour*6)
	return clusters
}

// Get gets a cluster id for ip s or returns s if there is none
func (cm ClusterMap) Get(s string) string {
    //f, err := os.OpenFile("/home/klee/ips.txt", os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0666)
	//if err != nil {
	//	log.Fatalf("error opening file: %v", err)
	//}
	//defer f.Close()
	//log.SetOutput(f)
	//log.Println(s)
	item, ok := cm.ca.Get("CM_" + s)
	if !ok {
		return cm.fetchCluster(s)
	}
	return string(item.(string))
}

// New creates a new cluster map using ClusterSource cs to retreive
// cluster ids and caching them in ca
func New(cs types.ClusterSource, ca types.Cache) ClusterMap {
	return ClusterMap{
		cs: cs,
		ca: ca,
	}
}
