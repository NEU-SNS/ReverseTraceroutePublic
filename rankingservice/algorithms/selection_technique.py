import pyasn
import random
from mysql.connector import connect
from algorithms.database import mysql_host, mysql_user, mysql_pwd

class RRVPSelector(object):



    def __init__(self, db_host, db_password, database, ipasnfile):
        self.db_host = db_host
        self.db_password = db_password
        self.database = database
        self.ip2asn = pyasn.pyasn(ipasnfile)

        self.site_per_vp, self.vp_per_site, _ = RRVPSelector.get_vp_sites()
        # Fill the two structure aboves

        self.global_ranking = []
        global_ranking_file = "resources/global_cover_vps"
        with open(global_ranking_file) as f:
            for line in f:
                self.global_ranking.append(int(line))
    def insert(self, rr_ping_table, selection_table, tmp_file, is_dump):
        pass

    def getRRVPs(self, destination, granularity, n_vp):
        pass

    def sites_to_vps(self, ranked_sites):
        ranked_vps = []
        for site, ingress, distance in ranked_sites:
            if site in self.vp_per_site:
                random_vp = random.choice(self.vp_per_site[site])
                if random_vp is not None:
                    ranked_vps.append((random_vp, ingress, distance, site))
        return ranked_vps

    @classmethod
    def get_vp_sites(cls):
        host = mysql_host
        user = mysql_user
        pwd = mysql_pwd
        database = "vpservice"
        query = (
            f" SELECT ip, hostname, site, record_route, spoof FROM {database}.vantage_points "
        )
        # print(query)

        # Get the connection to
        connection = connect(host=host,
                             user=user,
                             passwd=pwd,
                             db=database
                             )
        cursor = connection.cursor()
        cursor.execute(query, )
        rows = cursor.fetchall()
        site_per_vp = {}
        vps_per_site = {}
        hostname_per_ip = {}
        for row in rows:
            ip, hostname, site, can_record_route, can_spoof = row
            # if "staging" in hostname:
            #     continue
            if can_record_route == 1 and can_spoof == 1:
                site_per_vp[ip] = site
                vps_per_site.setdefault(site, []).append(ip)
                hostname_per_ip[ip] = hostname
        cursor.close()
        connection.close()

        return site_per_vp, vps_per_site, hostname_per_ip

    @classmethod
    def greedy_set_cover_algorithm(cls, rows, max_vp):

        # Compute the union of all sets
        space_ips = set()
        for i, (_, covered_ips_within_range_vp) in enumerate(rows):
            space_ips.update(covered_ips_within_range_vp)

        selected_vps_search = set()
        selected_vps = []
        while len(space_ips) != 0:
            # Select the set with the biggest intersection
            best_current_vp = None
            best_additional_covered_ip_addresses = set()

            for vp, covered_ips_within_range_vp in rows:
                if vp in selected_vps_search:
                    continue
                additional_covered_ip_addresses = space_ips.intersection(covered_ips_within_range_vp)
                if len(additional_covered_ip_addresses) > len(best_additional_covered_ip_addresses):
                    best_current_vp = vp
                    best_additional_covered_ip_addresses = additional_covered_ip_addresses

            selected_vps_search.add(best_current_vp)
            selected_vps.append(best_current_vp)
            space_ips = space_ips - best_additional_covered_ip_addresses
            if len(selected_vps) == max_vp:
                break

        return selected_vps

    def add_global_ranking(self, ranked_sites):
        ranked_sites = list(ranked_sites)
        ranked_sites_s = set([r[0] for r in ranked_sites])

        for vp in self.global_ranking:
            if vp in self.site_per_vp:
                site = self.site_per_vp[vp]
                if site in ranked_sites_s:
                    continue
                ranked_sites.append((site, 0, 0))
        return ranked_sites