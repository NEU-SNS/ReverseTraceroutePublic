from concurrent import futures
import logging

# import grpc
import grpc
from grpc.experimental import aio
from clickhouse_driver import Client

import asyncio

import protos.rankingservice_pb2 as rankingservice_pb2 
import protos.rankingservice_pb2_grpc as rankingservice_pb2_grpc
import rankingservice_utils as rs_utils
import sys


from network_utils import dnets_of, bgpDBFormat, prefix_24_from_ip, ipv4_index_of, ipItoStr
from algorithms.global_cover import GlobalCoverRRVPSelector
from algorithms.destination_cover import DestinationCoverRRVPSelector
from algorithms.ingress_cover import IngressCoverRRVPSelector
from algorithms.random_cover import RandomCoverRRVPSelector
from algorithms.old_revtr_cover import OldRevtrCoverRRVPSelector
from algorithms.database import db_host, clickhouse_pwd


class RankingServicer(rankingservice_pb2_grpc.RankingServicer):
    """Methods that implement functionality of the ranking service server"""

    def __init__(self, db_host, db_password, database, ipasnfile, ingress_table):
        global_cover_vps_file = "resources/global_cover_vps"
        global_table = "RRPingsEvaluation"
        self.selection_algorithms = {
            "global_cover"     : GlobalCoverRRVPSelector(db_host, db_password, database, global_table, ipasnfile, global_cover_vps_file),
            "destination_cover": DestinationCoverRRVPSelector(db_host, db_password, database, ipasnfile),
            "ingress_cover"    : IngressCoverRRVPSelector(db_host, db_password,
                                                          database, ingress_table,
                                                          ipasnfile,
                                                          is_in_memory=True),
            "old_revtr_cover"  : OldRevtrCoverRRVPSelector(db_host, db_password, database, ipasnfile, use_cache=True),
            "random_cover"     : RandomCoverRRVPSelector(db_host, db_password, database, ipasnfile)
        }



    async def getVPs(self, request, context):
        # print("Request")
        # measurement_db = rs_utils.connectDB("measurement")
        # ranking_db = rs_utils.connectDB("ranking")
        destination = request.ip
        n_vp = request.numVPs

        if request.rankingTechnique not in self.selection_algorithms:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details('Selection algorithm not known')
            return rankingservice_pb2.GetVPsResp()
        vps = self.selection_algorithms[request.rankingTechnique].getRRVPs(destination, "BGP", n_vp)
        # Transform vps into sources
        ret = [rankingservice_pb2.Source(
            hostname="dummy_hostname",
            ip=ipItoStr(vp[0]),
            site=vp[3],
            ingress=ipItoStr(vp[1]),
            distance=vp[2]
        )for vp in vps]
        resp = rankingservice_pb2.GetVPsResp()
        resp.vps.extend(ret)
        return resp

    def getTargetsFromHitlist(self, request, context):
        assert(request.granularity in  ["bgp", "/24"])
        '''
        This function returns the destinations to probe from a hitlist, from the clickhouse
        database
        '''
        if not request.is_initial:
            client = Client(rs_utils.db_host, password=rs_utils.db_pwd)
            query = (
                f"SELECT ip from RRResponsiveness WHERE responsiveness > 0"
            )
            settings = {'max_block_size': 1000}
            rows = client.execute_iter(query, settings=settings)
            targets = []
            for row in rows:
                ip = row
                targets.append(rs_utils.long2ip(ip))
            result = rankingservice_pb2.getTargetsResp(targets=targets)
            return result
        else:
            bgp_prefixes_filter = None
            if request.is_only_addresses:
                bgp_prefixes_filter = set()
                for ip in request.ip_addresses:
                    # ip = ipv4_index_of(ip.ip_address)
                    dentry = dnets_of(ip, self.ip2asn)
                    bgp_start, bgp_end = bgpDBFormat(dentry)
                    if bgp_start == 0:
                        bgp_start = prefix_24_from_ip(ip)
                        bgp_end = prefix_24_from_ip(ip) + 255
                    bgp_prefixes_filter.add((bgp_start, bgp_end))
            # Get targets from ISI hitlist score 
            client = Client(rs_utils.db_host, password=rs_utils.db_pwd)
            if request.granularity == "bgp":
                query = (
                    f" WITH max(score) as max_score, "
                    f" groupArray((ip, score)) as ip_score, "
                    f" arrayFilter(x->x.2=max_score, ip_score) as ip_max_score "
                    f" SELECT bgp_prefix_start, bgp_prefix_end, ip_max_score from revtr.hitlist "
                    f" GROUP BY (bgp_prefix_start, bgp_prefix_end) "
                    f" HAVING max_score > 0 "
                )
                print(query)
                settings = {'max_block_size': 100000}
                rows = client.execute_iter(query, settings=settings)
                targets = []
                for bgp_prefix_start, bgp_prefix_end, ip_max_score in rows:
                    if bgp_prefixes_filter is not None:
                        if (bgp_prefix_start, bgp_prefix_end) not in bgp_prefixes_filter:
                            continue
                    # Take 10 IPs at most in the BGP prefix
                    for i, (ip, _) in enumerate(ip_max_score):
                        if i == 1:
                            break
                        targets.append(rs_utils.long2ip(ip))
            elif request.granularity == "/24":
                query = (
                    f" WITH max(score) as max_score, "
                    f" groupArray((ip, score)) as ip_score, "
                    f" arrayFilter(x->x.2=max_score, ip_score) as ip_max_score "
                    f" SELECT ip_24, ip_max_score from revtr.hitlist "
                    f" GROUP BY ip_24 "
                    f" HAVING max_score > 0 "
                    # f" LIMIT 100000"
                )
                print(query)
                settings = {'max_block_size': 100000}
                rows = client.execute_iter(query, settings=settings)
                targets = []
                for ip_24, ip_max_score in rows:
                    # Take 10 IPs at most in the BGP prefix
                    for i, (ip, _) in enumerate(ip_max_score):
                        if i == 1:
                            break
                        targets.append(rs_utils.long2ip(ip))
            result = rankingservice_pb2.GetTargetsResp(targets=targets)
            return result
        

async def serve(ingress_table):
    # ipasnfile = '../ranking_periodic/bgpdumps/latest'
    ipasnfile = 'resources/bgpdumps/latest'
    database  = "revtr"
    server    = aio.server(futures.ThreadPoolExecutor(max_workers=1000))
    rankingservice_pb2_grpc.add_RankingServicer_to_server(
        RankingServicer(db_host, clickhouse_pwd, database, ipasnfile, ingress_table), server
    )
    server.add_insecure_port('[::]:49492')
    await server.start()
    await server.wait_for_termination()

if __name__ == "__main__":
    ingress_table = sys.argv[1]
    logging.basicConfig()
    asyncio.run(serve(ingress_table))
