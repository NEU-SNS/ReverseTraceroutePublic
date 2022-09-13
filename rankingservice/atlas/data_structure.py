try:
    import ujson as json
except ImportError:
    import json
from collections import defaultdict
from ast import literal_eval as make_tuple
import math
from bisect import bisect_left
import ipaddress
from network_utils import dnets_of
import socket
import random
from multiprocessing import Pool
import tqdm
from atlas.ripe import fetch_traceroute_results_from_our_measurements
from atlas.atlas_database import get_connection, get_alias_sets


def json_result_by_ttl(replies):
    # 3 is the index of the TTL.
    # ttls_after_destination =  [int(reply[3]) for reply in replies if int(reply[0]) == destination]
    # ttls_raw = [reply[3] for reply in replies if reply[3] <= max_sent_ttl]
    # if len(ttls_after_destination) > 0:
    #     max_ttl_after_destination = min(ttls_after_destination)
    #     if len(ttls_raw) == 0:
    #         return None
    #     max_ttl_raw = int(max(ttls_raw))
    #     max_ttl = min(max_ttl_raw, max_ttl_after_destination)
    # else:
    #     max_ttl = int(max_sent_ttl)
    max_ttl = max(r[1] for r in replies)
    result = [{"hop": i, "result":[]} for i in range (1, max_ttl + 1)]

    results_per_hop = {i : 0 for i in range(1, max_ttl + 1)}

    for reply_ip, ttl in replies:
        if ttl == 0:
            continue
        results_per_hop[ttl] += 1
        result[ttl-1]["result"].append(
            {
                "from": str(ipaddress.IPv4Address(reply_ip)),
            })

    for hop in range(1, max_ttl + 1):
        if results_per_hop[hop] == 0:
            result[hop - 1]["result"].append({"x":"*"
                                              })


    return result

def convert_mysql_traceroute_to_ripe_format(source, destination, replies):
    af = "4"  # Only IPv4 atm
    result = json_result_by_ttl(replies)

    # algorithm = "d-miner"

    output = {
        "af": int(af),
        "dst_addr": str(ipaddress.IPv4Address(destination)),
        "dst_name": str(ipaddress.IPv4Address(destination)),
        # "endtime": end_time.timestamp(),
        "from": str(ipaddress.IPv4Address(source)),
        "prb_id": source,
        "fw": 4790,
        "lts": -1,
        "msm_id": -1,
        # "msm_name": algorithm,
        "paris_id": 0,
        # To be changed when other protocols will be available
        "proto": "ICMP",
        "result": result,
        # TCP payload
        "size": "Unknown",
        "src_addr": str(ipaddress.IPv4Address(source)),
        # "timestamp": start_time.timestamp(),
        "type": "traceroute"
    }

    return output

class Atlas(object):

    def __init__(self):
        self.sources = set()
        self.weights_by_ip = {}
        self.ips_per_destination_source = {}
        self.links_per_destination_source = {}
        self.traceroutes_by_ip = {}
        self.raw_traceroutes = {}
        self.alias_sets = {}
        self.date_per_traceroute = {}


    def recompute(self, weight_type):
        '''
        Alias sets must be set before calling this function
        :param weight_type:
        :return:
        '''
        self.sources = set()
        self.weights_by_ip = {}
        self.ips_per_destination_source = {}
        self.traceroutes_by_ip = {}
        raw_traceroutes = self.raw_traceroutes.items()
        self.raw_traceroutes = {}
        for _, traceroute in raw_traceroutes:
            self.add_traceroute(traceroute, weight_type)

    def intersect(self, traceroute, is_include_self, is_include_source_asn, ip2asn):
        destination = traceroute["from"]  # From where we probed to the source
        source = traceroute["dst_addr"]
        results = traceroute["result"]
        probe_id = traceroute["prb_id"]

        for result in results:
            hop = result["hop"]
            replies = result["result"]
            for reply in replies:
                if "x" in reply or reply == {}:
                    continue
                reply_ip = reply["from"]
                if reply_ip == source:
                    continue
                if ipaddress.ip_address(reply_ip).is_private:
                    continue
                reply_ip = self.alias_sets[reply_ip]
                if reply_ip in self.traceroutes_by_ip:
                    intersect_traceroutes_candidates = self.traceroutes_by_ip[reply_ip]
                    # Look for one that has the same destination
                    intersect_traceroutes = []
                    for intersect_traceroute in intersect_traceroutes_candidates:
                        if intersect_traceroute[3] == source:
                            intersect_traceroutes.append(intersect_traceroute)
                    for intersect_traceroute in intersect_traceroutes:
                        # Prioritize a traceroute from the same destination
                        if intersect_traceroute[2] == destination and intersect_traceroute[2] == source:
                            if is_include_self:
                                return reply_ip, intersect_traceroute
                            else:
                                continue
                    asn_reply_ip = dnets_of(reply_ip, ip2asn).asn
                    for intersect_traceroute in intersect_traceroutes:
                        if intersect_traceroute[2] == destination and intersect_traceroute[2] == source:
                            continue
                        elif intersect_traceroute[3] == source:
                            asn_source = dnets_of(source, ip2asn).asn
                            if asn_source == asn_reply_ip:
                                if is_include_source_asn:
                                    return reply_ip, intersect_traceroute
                                else:
                                    continue
                            else:
                                return reply_ip, intersect_traceroute

    def add_traceroute(self, traceroute, weight_type):
        assert (weight_type in ["reverse_hops", "absolute"])
        destination = traceroute["from"] # From where we probed to the source
        source      = traceroute["dst_addr"]
        results     = traceroute["result"]
        probe_id    = traceroute["prb_id"]
        ripe_measurement_id  = traceroute["msm_id"]
        date        = traceroute["endtime"]
        distance_to_destination = None
        reply_ips = []
        ttl_by_ip = {}
        ip_by_ttl = {}
        has_reach_destination = False
        key = (ripe_measurement_id, probe_id, destination, source)
        if key in self.raw_traceroutes:
            return
        self.sources.add(source)
        for result in results:
            hop = result["hop"]
            if "error" in result:
                continue
            replies = result["result"]
            for reply in replies:
                if "x" in reply or reply == {} or "error" in reply:
                    continue
                reply_ip = reply["from"]
                if ipaddress.ip_address(reply_ip).is_private:
                    continue
                if reply_ip == source:
                    distance_to_destination = hop
                    ttl_by_ip[reply_ip] = hop
                    reply_ips.append(reply_ip)
                    has_reach_destination = True
                    continue
                reply_ips.append(reply_ip)
                if weight_type  == "reverse_hops":
                    ttl_by_ip[reply_ip] = hop
                elif weight_type == "absolute":
                    # ttl_by_ip.setdefault(reply_ip, 0)
                    ttl_by_ip[reply_ip] = hop
                ip_by_ttl[hop] = reply_ip

        if not has_reach_destination:
            return

        for ip, ttl in ttl_by_ip.items():
            self.weights_by_ip.setdefault(self.alias_sets[ip], 0)
            if weight_type == "reverse_hops":
                self.weights_by_ip[self.alias_sets[ip]] += distance_to_destination - ttl
            elif weight_type == "absolute":
                self.weights_by_ip[self.alias_sets[ip]] += 1
            self.traceroutes_by_ip.setdefault(self.alias_sets[ip], []).append(key)

        # Extracts the traceroute links
        # traceroute_links_by_ttl = {}
        # sorted_ttl_ip = sorted(ip_by_ttl.items(), key=lambda x:x[0])
        # for i, (ttl, ip) in enumerate(sorted_ttl_ip):
        #     next_ttl, next_ip = sorted_ttl_ip[i+1]
        #     if next_ttl == ttl


        self.ips_per_destination_source[key] = [self.alias_sets[ip] for ip in reply_ips]
        self.raw_traceroutes[key] = traceroute
        self.date_per_traceroute[key] = date
        # print(self.ips_per_destination)
        return self

    @classmethod
    def extract_ips_from_raw_traceroutes(cls, traceroutes):
        ips = set()
        for traceroute in traceroutes:
            results = traceroute["result"]
            for result in results:
                hop = result["hop"]
                if "error" in result:
                    continue
                replies = result["result"]
                for reply in replies:
                    if "x" in reply or reply == {} or "error" in reply:
                        continue
                    reply_ip = reply["from"]
                    if ipaddress.ip_address(reply_ip).is_private:
                        continue
                    ips.add(reply_ip)
        return ips

    def serialize(self, ofile):
        d = dict(self.__dict__)
        d["sources"] = list(d["sources"])
        d["ips_per_destination_source"] = {str(k): d["ips_per_destination_source"][k] for k in
                                           d["ips_per_destination_source"]}
        d["raw_traceroutes"] = {str(k): d["raw_traceroutes"][k] for k in
                                           d["raw_traceroutes"]}
        with open(ofile, "w") as f:
            json.dump(d, f)

    @classmethod
    def deserialize(cls, ifile):
        atlas = Atlas()
        with open(ifile) as f:
            d = json.load(f)
            atlas.weights_by_ip = d["weights_by_ip"]
            ips_per_destination_source = d["ips_per_destination_source"]
            atlas.ips_per_destination_source = {make_tuple(k): ips_per_destination_source[k] for k in
                                                ips_per_destination_source}
            raw_traceroutes = d["raw_traceroutes"]
            atlas.raw_traceroutes = {make_tuple(k): raw_traceroutes[k] for k in
                                     raw_traceroutes}
            atlas.sources = set(d["sources"])
            traceroutes_by_ip = d["traceroutes_by_ip"]
            for ip, traceroute_meta in traceroutes_by_ip.items():
                for k, t in enumerate(traceroute_meta):
                    traceroute_meta[k] = tuple(t)
            atlas.traceroutes_by_ip = traceroutes_by_ip
            if "date_per_traceroute" in d:
                atlas.date_per_traceroute = d["date_per_traceroute"]

        return atlas

    @classmethod
    def build_atlas_from_ripe_traceroutes(cls, traceroutes, alias_sets):
        atlas = Atlas()
        atlas.alias_sets = alias_sets
        for traceroute in traceroutes:
            atlas.add_traceroute(traceroute, weight_type="absolute")
        return atlas

    @classmethod
    def fetch_and_build_atlas_from_ripe_traceroutes(cls, label, timestamp):
        traceroutes = fetch_traceroute_results_from_our_measurements(label,
                                                       timestamp,
                                                       measurement_limit=150,
                                                       is_most_recent=False)
        atlas = Atlas()
        atlas.raw_traceroutes = traceroutes
        ips = Atlas.extract_ips_from_raw_traceroutes(traceroutes)
        alias_sets_int = get_alias_sets(tuple(ips))
        alias_sets = {}
        for ip in ips:
            if ip not in alias_sets_int:
                alias_sets[str(ipaddress.ip_address(ip))] = str(ipaddress.ip_address(ip))
            else:
                alias_sets[str(ipaddress.ip_address(ip))] = alias_sets_int[ip]

        atlas = Atlas.build_atlas_from_ripe_traceroutes(traceroutes, alias_sets)
        return atlas

    @classmethod
    def build_atlas_from_ripe_traceroutes_without_alias_sets(cls, traceroutes):
        ips = Atlas.extract_ips_from_raw_traceroutes(traceroutes)
        alias_sets = {}
        atlas = Atlas()
        for ip in ips:
            alias_sets[ip] = ip
        atlas.alias_sets = alias_sets
        for traceroute in traceroutes:
            atlas.add_traceroute(traceroute, weight_type="absolute")
        return atlas


    @classmethod
    def ips_from_ripe_traceroute(cls, batch_index, traceroutes, filter):
        print(f"Loading batch index {batch_index}")
        ips_per_traceroute = {}
        for traceroute in traceroutes:
            destination = traceroute["from"]  # From where we probed to the source
            source = traceroute["dst_addr"]
            results = traceroute["result"]
            probe_id = traceroute["prb_id"]
            ripe_measurement_id = traceroute["msm_id"]
            date = traceroute["endtime"]
            reply_ips = [(destination, 0)]
            key = (ripe_measurement_id, probe_id, destination, source)
            if filter is not None and filter != (destination, source):
                continue
            for result in results:
                hop = result["hop"]
                if "error" in result:
                    continue
                replies = result["result"]
                for reply in replies:
                    if "x" in reply or reply == {} or "error" in reply:
                        continue
                    reply_ip = reply["from"]
                    if ipaddress.ip_address(reply_ip).is_private:
                        continue
                    if reply_ip == source:
                        reply_ips.append((reply_ip, hop))
                        has_reach_destination = True
                        continue
                    reply_ips.append((reply_ip, hop))
            ips_per_traceroute.update({(key, date): reply_ips})
        return ips_per_traceroute


    @classmethod
    def fast_ips_per_traceroute(cls, traceroutes, filter):
        ips_per_traceroute = {}
        batch_traceroutes = [(i, traceroutes[i:i + 1000], filter) for i in range(0, len(traceroutes), 1000)]
        n_process = 32
        if filter is not None:
            n_process = 1
        with Pool(n_process) as p:
            # results = tqdm.tqdm(p.istarmap(Atlas.ips_from_ripe_traceroute, batch_traceroutes), total=len(batch_traceroutes))
            results = p.starmap(Atlas.ips_from_ripe_traceroute, batch_traceroutes)

        for result in results:
            ips_per_traceroute.update(result)
        return ips_per_traceroute

    @classmethod
    def build_atlas_from_mysql_db(cls, datetime, source):
        connection = get_connection("traceroute_atlas")

        cursor = connection.cursor()
        query = (
            f" SELECT src, dest, hop, ttl "
            f" FROM atlas_traceroutes at "
            f" INNER JOIN atlas_traceroute_hops ath ON at.id = ath.trace_id "
            f" WHERE at.date > '{datetime}' and at.platform='mlab'"
        )
        if source is not None:
            srcI = int(ipaddress.ip_address(source))
            query += f" and dest={srcI} "
        query += (
            f" ORDER BY ttl"
        )

        cursor.execute(query)

        rows = cursor.fetchall()
        traceroutes = {}
        ips = set()
        for src, dest, hop, ttl in rows:
            traceroutes.setdefault((src, dest), []).append((hop, ttl))
            ips.add(hop)
            ips.add(src)
            ips.add(dest)

        if len(ips) > 0:
            alias_sets_int = get_alias_sets(tuple(ips))
        else:
            alias_sets_int = {}
        alias_sets = {}
        for ip in ips:
            if ip not in alias_sets_int:
                alias_sets[str(ipaddress.ip_address(ip))] = str(ipaddress.ip_address(ip))
            else:
                alias_sets[str(ipaddress.ip_address(ip))] = alias_sets_int[ip]

        ripe_traceroutes = []
        for (src, dest), replies in traceroutes.items():
            ripe_traceroutes.append(convert_mysql_traceroute_to_ripe_format(src, dest, replies))


        atlas = Atlas.build_atlas_from_ripe_traceroutes(ripe_traceroutes, alias_sets)

        return atlas




    def distance(self, ip, traceroute):
        _ = traceroute["from"]  # From where we probed to the source
        source = traceroute["dst_addr"]
        results = traceroute["result"]
        _ = traceroute["prb_id"]
        distance_to_destination = None
        has_reach_destination = False
        ip_distance = None
        private_ip_addresses = 0
        for i, result in enumerate(results):
            hop = result["hop"]
            replies = result["result"]
            for reply in replies:
                if "x" in reply or reply == {}:
                    private_ip_addresses += 1
                    continue
                reply_ip = reply["from"]
                is_private = ipaddress.ip_address(reply_ip).is_private
                if is_private:
                    private_ip_addresses += 1
                    continue
                reply_ip = self.alias_sets[reply_ip]
                if reply_ip == source:
                    distance_to_destination = hop
                    has_reach_destination = True
                    if ip == source:
                        # if hop > 64:
                        ip_distance = i + 1
                    continue
                if reply_ip == ip:
                    ip_distance = i + 1

        if not has_reach_destination:
            return None, None

        distance_to_destination = distance_to_destination
        saved_distance = distance_to_destination - ip_distance
        return saved_distance, distance_to_destination, private_ip_addresses





    def weight(self, s, weights):
        return sum([weights[e] for e in s])

    def greedy_weighted_set_cover_algorithm(self, maximum_set_number, sets, weights, with_analyze):
        assert(type(sets) == dict)
        # Defensive copies
        sets = dict(sets)
        weights = dict(weights)
        # Compute the union of all sets
        space = set()
        for _, s in self.ips_per_destination_source.items():
            space.update(s)

        if maximum_set_number is None or maximum_set_number > len(self.ips_per_destination_source):
            maximum_set_number = len(self.ips_per_destination_source)

        if with_analyze:
            cumulative_weights = []
            cumulative_ips     = []

        selected_sets = []
        while len(selected_sets) < maximum_set_number and len(space) > 0 :
            print(len(selected_sets))
            # Select the set with the biggest intersection
            best_set = None
            best_additional_weight = -1
            best_set_elements = None
            for s, elements in sets.items():
                # The score of the a set if the score of the weights
                additional_elements = space.intersection(elements)
                additional_weight   = self.weight(additional_elements, weights)
                if additional_weight > best_additional_weight:
                    best_set = s
                    best_additional_weight = additional_weight
                    best_set_elements = additional_elements

            selected_sets.append(best_set)
            space = space - best_set_elements
            if with_analyze:
                if len(cumulative_weights) == 0:
                    cumulative_weights.append(best_additional_weight)
                    cumulative_ips.append(len(best_set_elements))
                else:
                    cumulative_weights.append(best_additional_weight + cumulative_weights[-1])
                    cumulative_ips.append(len(best_set_elements) + cumulative_ips[-1])
            del sets[best_set]

        if with_analyze:
            # Normalize cumulative weights
            total_weight = sum(weights.values())
            cumulative_weights = [x/total_weight for x in cumulative_weights]
            cumulative_ips = [x / len(weights) for x in cumulative_ips]

            return selected_sets, cumulative_weights, cumulative_ips
        return selected_sets


    def binary_search(self, v, w):
        '''
        Look for k such that p^k < w <= p^(k+1)
        :param p:
        :param w:
        :return:
        '''

        i = bisect_left(v, w)
        return i


    def dfg(self, maximum_set_number, sets, weights,  p=1.05, partial_cover=1):
        """Compute the best set of traceroutes."""
        assert (type(sets) == dict)
        # Defensive copies
        sets = dict(sets)
        weights = dict(weights)
        # Compute the union of all sets
        space = set()
        for _, s in self.ips_per_destination_source.items():
            space.update(s)

        if maximum_set_number is None or maximum_set_number > len(self.ips_per_destination_source):
            maximum_set_number = len(self.ips_per_destination_source)


        # Get the total discoveries
        # for agent, prefixes in discoveries_per_agent.items():
        #     ranks_per_agent[agent] = []
        #     for prefix, reply_ips in prefixes.items():
        #         subsets[(agent, prefix)] = set(reply_ips)
        #         total_discoveries.update(reply_ips)

        # Shuffle the subsets
        # subsets_list = list(subsets.items())
        # random.shuffle(subsets_list)
        # subsets = dict(subsets_list)

        selected_sets = []
        # Populate the subcollections
        subcollections = defaultdict(list)
        max_weight = max(self.weight(s, weights) for _, s in sets.items())
        k_max = int(math.log(max_weight) / math.log(p))
        v = [p**k for k in range(0, k_max + 1)]
        for meta, s in sets.items():

            s = set(s)
            k = self.binary_search(v, self.weight(s, weights))
            subcollections[k-1].append((meta, s))

        covered = set()
        # Adjust k_max so we start with the element with the maximum weight
        # k_max += 1
        # k = k_max ... 1
        remaining = []
        is_adding_only_weight_one_tr = False
        has_reached_max_set_number = False
        for k in range(k_max, -1, -1):
            for meta, s in subcollections[k]:
                w = self.weight(s - covered, weights)
                max_tr_weight = max(weights[i] for i in s)
                if max_tr_weight == 1 and not is_adding_only_weight_one_tr:
                    print("Number of traceroutes needed to cover all intersections", len(selected_sets), len(sets))
                    is_adding_only_weight_one_tr = True
                if w >= v[k]:
                    selected_sets.append((meta, s))
                    if len(selected_sets) == maximum_set_number:
                        has_reached_max_set_number = True
                        break
                    covered.update(s)
                    if len(covered) / len(space) >= partial_cover:
                        break
                else:
                    s -= covered
                    w_prime = self.weight(s, weights)
                    k_prime = self.binary_search(v, w_prime)
                    if k_prime == 0 and w_prime > 0:
                        subcollections[k_prime].append((meta, s))
                    else:
                        subcollections[k_prime - 1].append((meta, s))
            if has_reached_max_set_number:
                break

        # if not has_reached_max_set_number:
        #     # k = 0
        #     for meta, s in subcollections[0]:
        #         w = self.weight(s - covered, weights)
        #         if w > 0:
        #             selected_sets.append((meta, s))
        #             covered.update(s)
        #             if len(selected_sets) == maximum_set_number:
        #                 break

        print(f"Covered: {len(covered)/ len(space)}")
        return selected_sets


    def random(self, ip2asn, probes, n_traceroutes):
        '''
        Select random traceroutes not authorizing same traceroute in the same AS/country
        :param ip2asn:
        :param fraction_traceroutes:
        :return:
        '''

        # Shuffle the traceroutes
        random_traceroutes = list(dict(self.raw_traceroutes).items())

        traceroutes = []
        next_traceroutes  = []
        # Take only anchors?
        for meta, tr in random_traceroutes:
            if meta[0] in probes:
                if probes[meta[0]]["is_anchor"]:
                    traceroutes.append((meta, tr))
                else:
                    next_traceroutes.append((meta, tr))
        traceroutes.extend(next_traceroutes)
        return traceroutes[:n_traceroutes]
        random.shuffle(random_traceroutes)
        selected_as_country_pair = set()
        redundant_traceroutes = []
        selected_traceroutes = []
        for meta, traceroute in random_traceroutes:
            asn = dnets_of(meta[1], ip2asn).asn
            if meta[0] in probes:
                country = probes[meta[0]]["country_code"]
            else:
                country = "Unknown"
            if (asn, country) in selected_as_country_pair:
                redundant_traceroutes.append((meta, traceroute))
            else:
                selected_traceroutes.append((meta, traceroute))
                selected_as_country_pair.add((asn, country))

        selected_traceroutes.extend(redundant_traceroutes)
        return selected_traceroutes[:n_traceroutes]


    def ttl_distance_distribution(self):
        distance_by_ip = {}
        for meta, traceroute in self.raw_traceroutes.items():
            destination = traceroute["from"]  # From where we probed to the source
            source = traceroute["dst_addr"]
            results = traceroute["result"]
            probe_id = traceroute["prb_id"]
            distance_to_destination = None
            ips_per_destination_source = {}
            ttl_by_ip = {}
            has_reach_destination = False
            key = (probe_id, destination, source)
            self.sources.add(source)
            for result in results:
                hop = result["hop"]
                if "error" in result:
                    continue
                replies = result["result"]
                for reply in replies:
                    if "x" in reply or reply == {} or "error" in reply:
                        continue
                    reply_ip = reply["from"]
                    if ipaddress.ip_address(reply_ip).is_private:
                        continue
                    if reply_ip == source:
                        distance_to_destination = hop
                        has_reach_destination = True
                        continue
                    ips_per_destination_source.setdefault(key, []).append(reply_ip)
                    ttl_by_ip[reply_ip] = hop

            if not has_reach_destination:
                return
            for ip, ttl in ttl_by_ip.items():
                distance_by_ip.setdefault(ip, []).append(distance_to_destination - ttl)
        return distance_by_ip

    @classmethod
    def dump_traceroute_txt(cls, traceroute, ip2asn):
        '''
        Dump a ripe traceroute in a text format with ASN and ptr informations and extensions
        :param traceroute:
        :return:
        '''
        results = traceroute["result"]
        for result in results:
            hop = result["hop"]
            if "error" in result:
                continue
            replies = result["result"]
            for reply in replies:
                if "x" in reply or reply == {} or "error" in reply:
                    print("x")
                    continue
                reply_ip = reply["from"]
                try:
                    ptr = socket.gethostbyaddr(reply_ip)[0]
                except socket.herror:
                    ptr = ""
                    pass
                icmp_ext  = ""
                if "icmpext" in reply:
                    # icmp_ext = reply["icmpext"]
                    if "mpls" in reply["icmpext"]["obj"][0]:
                        icmp_ext = "mpls"
                print(reply_ip, ptr, dnets_of(reply_ip, ip2asn).asn, icmp_ext)

    @classmethod
    def hops_in_tunnels(cls, traceroute):
        hops_in_tunnel = []
        results = traceroute["result"]
        for result in results:
            hop = result["hop"]
            if "error" in result:
                continue
            replies = result["result"]
            for reply in replies:
                if "x" in reply or reply == {} or "error" in reply:
                    continue
                reply_ip = reply["from"]
                try:
                    ptr = socket.gethostbyaddr(reply_ip)[0]
                except socket.herror:
                    ptr = ""
                    pass
                icmp_ext = ""
                if "icmpext" in reply:
                    # icmp_ext = reply["icmpext"]
                    if "mpls" in reply["icmpext"]["obj"][0]:
                        hops_in_tunnel.append(reply_ip)
        return hops_in_tunnel

'''
This is the ripe atlas format
        {'fw': 4790, 'lts': 46, 'endtime': 1608163058, 'dst_name': '4.15.166.25',
                        'dst_addr': '4.15.166.25', 'src_addr': '192.168.178.26', 'proto': 'ICMP', 'af': 4, 'size': 48,
                        'paris_id': 1, 'result': [{'hop': 1, 'result': [{'x': '*'}]}, {'hop': 2, 'result': [
                {'from': '194.109.5.177', 'ttl': 254, 'size': 28, 'rtt': 13.846}]}, {'hop': 3, 'result': [
                {'from': '194.109.7.73', 'ttl': 253, 'size': 28, 'rtt': 15.665}]}, {'hop': 4, 'result': [
                {'from': '194.109.5.7', 'ttl': 252, 'size': 28, 'rtt': 9.261}]}, {'hop': 5, 'result': [
                {'from': '134.222.94.216', 'ttl': 248, 'size': 28, 'rtt': 7.848}]}, {'hop': 6, 'result': [
                {'from': '194.122.122.102', 'ttl': 246, 'size': 28, 'rtt': 12.015}]}, {'hop': 7, 'result': [
                {'from': '89.149.181.54', 'ttl': 245, 'size': 28, 'rtt': 12.332}]}, {'hop': 8, 'result': [
                {'from': '4.68.38.121', 'ttl': 244, 'size': 28, 'rtt': 10.81}]}, {'hop': 9, 'result': [
                {'from': '4.69.153.121', 'ttl': 237, 'size': 28, 'rtt': 145.389}]}, {'hop': 10, 'result': [
                {'from': '4.15.166.25', 'ttl': 53, 'size': 48, 'rtt': 142.965}]}], 'msm_id': 28393854, 'prb_id': 1,
                        'timestamp': 1608163055, 'msm_name': 'Traceroute', 'from': '82.95.114.207',
                        'type': 'traceroute', 'group_id': 28393845, 'stored_timestamp': 1608163077}
'''



