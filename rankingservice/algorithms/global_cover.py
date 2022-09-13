from clickhouse_driver import Client
import os
import time
from algorithms.database import dropTable, clickhouse_pwd, db_host, dump_query, insert_from_file
from algorithms.selection_technique import RRVPSelector

def createRRPingsGlobalCoverTable(host, db, table):

    client = Client(host, password=clickhouse_pwd)
    client.execute(
        f"CREATE TABLE IF NOT EXISTS {db}.{table} ("
        f"dst_asn UInt32, dst_bgp_prefix_start UInt32, dst_bgp_prefix_end UInt32, dst_24 UInt32, " # Network metadata of dst
        f"rr_asn Nullable(UInt32), rr_bgp_prefix_start Nullable(UInt32), rr_bgp_prefix_end Nullable(UInt32), rr_24 UInt32, " # Newtork metadata of RR
        f"ping_id UInt64," 
        f"src UInt32, dst UInt32, reply_from UInt32, rr_hop UInt8, "
        f"rr_ip UInt32, created Datetime)"
        f"ENGINE=MergeTree() "
        f"ORDER BY (src, dst_bgp_prefix_start, dst_bgp_prefix_end) "
        )


class GlobalCoverRRVPSelector(RRVPSelector):

    def __init__(self, db_host, db_password, database, table, ipasnfile, vps_file):
        super(GlobalCoverRRVPSelector, self).__init__(db_host, db_password, database, ipasnfile)
        self.ranked_sites = []
        if not os.path.exists(vps_file):
            self.serializeRRVPs(table, vps_file)
        with open(vps_file) as f:
            for line in f:
                line = line.strip()
                vp = int(line)
                if vp in self.site_per_vp:
                    site = self.site_per_vp[vp]
                    self.ranked_sites.append(site)

    def insert(self, rr_ping_table, selection_table, tmp_file, is_dump):

        dropTable(self.db_host, self.db_password, self.database, selection_table)
        createRRPingsGlobalCoverTable(self.db_host, self.database, selection_table)

        if is_dump:
            if os.path.exists(tmp_file):
                os.remove(tmp_file)
            query = (
                f" SELECT "
                f"toUInt32(dst_asn) as dst_asn, toUInt32(dst_bgp_prefix_start), toUInt32(dst_bgp_prefix_end), dst_24, " # Network metadata of dst
                f"rr_asn, rr_bgp_prefix_start, rr_bgp_prefix_end , rr_24, " # Newtork metadata of RR
                f"ping_id," 
                f"src, dst, reply_from, rr_hop , "
                f"rr_ip, created "
                f" FROM {self.database}.{rr_ping_table} "
                f" WHERE isNotNull(dst_asn) AND isNotNull(dst_bgp_prefix_start) and isNotNull(dst_bgp_prefix_end) "
                # f" LIMIT 100000 "
                f" INTO OUTFILE '{tmp_file}' FORMAT CSV"
            )
            print(query)
            dump_query(self.db_host, self.db_password, query)

        insert_query = (
            f"INSERT INTO {self.database}.{selection_table} FORMAT CSV"
        )
        insert_from_file(self.db_host, self.db_password, tmp_file, insert_query)

    def serializeRRVPs(self, table, vps_file):

        group_by_columns = f"(src)"

        # Constrain the query due to memory issues...
        subquery = (
            f" SELECT * FROM {self.database}.{table} "
            f" LIMIT 100000 BY {group_by_columns}"
        )

        # Compute the distance from the vantage points to the prefix
        query = (
            f" WITH groupUniqArray((dst, rr_ip, rr_hop)) as covered_ips,"
            f" arrayFilter(x->x.3 < 8 and x.1 == x.2, covered_ips) as covered_ips_within_range "
            f" SELECT src, covered_ips_within_range "
            f" FROM ({subquery}) "
            f" GROUP BY {group_by_columns} "
        )

        query = (
            f" SELECT src, dst, rr_ip, rr_hop "
            f" FROM {self.database}.{table} "
            f" WHERE rr_hop < 7 AND rr_ip = dst "
            # f" LIMIT 10000"

        )
        covered_ips_within_range_by_vp = {}
        print(query)
        client = Client(self.db_host, password=self.db_password)
        settings = {'max_block_size': 10000}
        i = 0
        rows = client.execute(query, settings=settings)
        for src, dst, rr_ip, rr_hop in rows:
            i += 1
            if i % 100000 == 0:
                print(f"{i} rows read.")
            if src not in covered_ips_within_range_by_vp:
                covered_ips_within_range_by_vp[src] = set()
            covered_ips_within_range_by_vp[src].add(dst)

        start = time.time()
        covered_ips_within_range_by_vp = list(
            {vp: list(covered_ips_within_range_by_vp[vp]) for vp in covered_ips_within_range_by_vp}.items())
        selected_vps = self.greedy_set_cover_algorithm(covered_ips_within_range_by_vp, max_vp=250)
        elapsed = time.time() - start
        print(elapsed)
        client.disconnect()
        with open(vps_file, "w") as f:
            for vp in selected_vps:
                f.write(f"{vp}\n")

    def getRRVPs(self, destination, granularity, n_vp):

        '''
        This algorithm is the one used in the paper.
        :param destination: string representation of the IP address
        :return:
        '''
        ranked_vps = self.sites_to_vps(self.ranked_sites)
        if n_vp >= len(self.ranked_sites):
            return ranked_vps

        return ranked_vps[:n_vp]





def insert_global_cover():
    host = db_host
    database = "revtr"
    rr_ping_table = "RRPings"
    selection_table = "RRPingsGlobalCover"
    ipasnfile = 'resources/bgpdumps/latest'
    vps_file = "resources/global_cover_vps"
    selector = GlobalCoverRRVPSelector(db_host, clickhouse_pwd, database, ipasnfile, vps_file)
    tmp_file = f"algorithms/evaluation/resources/{rr_ping_table}.csv"
    selector.insert(rr_ping_table, selection_table, tmp_file, is_dump=False)
    vps = selector.getRRVPs("1.0.0.1", granularity=None, n_vp=250)

def compute_global_cover():
    database = "revtr"
    ipasnfile = 'resources/bgpdumps/latest'
    vps_file = "resources/global_cover_vps_evaluation"
    table = "RRPingsEvaluation"
    selector = GlobalCoverRRVPSelector(db_host, clickhouse_pwd, database, table, ipasnfile, vps_file)

def test():
    from network_utils import ipItoStr
    ipasnfile = 'resources/bgpdumps/latest'
    database = "revtr"
    vps_file = "resources/global_cover_vps"
    selector = GlobalCoverRRVPSelector(db_host, clickhouse_pwd, database, ipasnfile, vps_file)
    vps = selector.getRRVPs("36.68.72.1", granularity=None, n_vp=250)
    print([ipItoStr(vp) for vp in vps])

if __name__ == "__main__":
    compute_global_cover()





