import ipaddress
import argparse
import sys
from mysql.connector import connect
from atlas.atlas_database import get_connection, get_select_alias_query, get_alias_sets


def get_vps():
    database = "plcontroller"
    query = (
        f" SELECT ip, site FROM {database}.vantage_point "
    )
    # print(query)

    # Get the connection to
    connection = get_connection(database)
    cursor = connection.cursor()
    cursor.execute(query, )
    rows = cursor.fetchall()
    vps = []
    for row in rows:
        ip, site = row
        vps.append(ip)
    cursor.close()
    connection.close()
    return vps


def check_destination_based_routing(tr_hops, tr_hop_ping_id_rr_hops):
    '''
    Perform a sanity check that next rr hop of current tr_hop is contained in the next tr_hop RR
    :param tr_hop_ping_id_rr_hops:
    :return:
    '''
    for i, tr_hop in enumerate(tr_hops):
        if tr_hop not in tr_hop_ping_id_rr_hops:
            continue
        # Is this hop in a tunnel?
        # Look at next hop

    # for dst_tr_hop, ping_id_rr_hops in :


def flatten_connected(successors_per_hop):
    for hop in successors_per_hop:
        successors = successors_per_hop[hop]
        prev_successors = set()
        while len(successors) != len(prev_successors):
            prev_successors = set(successors)
            new_successors = set(successors)
            for successor in successors:
                if successor in successors_per_hop:
                    new_successors.update(successors_per_hop[successor])
            successors = new_successors
        successors_per_hop[hop] = successors

def compute_pred_succ_constraints(tr_hop_ping_id_rr_hops):
    '''
    Ugly implementation to find successors and predecessors of hops in RR
    :param tr_hop_ping_id_rr_hops:
    :return:
    '''
    predecessors_per_hop = {}
    successors_per_hop = {}
    for tr_hop, ping_id_rr_hops in tr_hop_ping_id_rr_hops.items():
        for _, rr_hops in ping_id_rr_hops.items():
            rr_hops = sorted(rr_hops, key=lambda x: x[1])
            for i in range(len(rr_hops)):
                rr_hop_i, _ = rr_hops[i]
                for j in range(i+1, len(rr_hops)):
                    rr_hop_j, _ = rr_hops[j]
                    predecessors_per_hop.setdefault(rr_hop_j, set()).add(rr_hop_i)
                    successors_per_hop.setdefault(rr_hop_i, set()).add(rr_hop_j)

    # Flatten the predecessors so that every hop has the full set of predecessors/successors.
    flatten_connected(successors_per_hop)
    flatten_connected(predecessors_per_hop)

    return predecessors_per_hop, successors_per_hop


def match_rr_tr_hops(cursor, ips_for_alias_set, rr_hops_per_tr_id, tr_hops_per_tr_id, with_p2p):
    # Get the alias of all the IPs seen in the traceroutes
    alias_set_by_ip = get_alias_sets(tuple(ips_for_alias_set))

    # For each rr hop, a subpath corresponding to the potential intersection
    rr_intersection_rows_to_insert = []
    for tr_id, tr_hop_ping_id_rr_hops in rr_hops_per_tr_id.items():
        # if tr_id != 2467:
        #     continue
        predecessors_per_hop, successors_per_hop = compute_pred_succ_constraints(tr_hop_ping_id_rr_hops)
        # print(tr_id)
        intersects_per_rr_hop = {}
        # Index per tr hop for faster retrieve
        tr_hops = tr_hops_per_tr_id[tr_id]
        tr_hops.sort(key=lambda x: x[1])
        index_per_tr_hop = {tr_hop[0]: i for i, tr_hop in enumerate(tr_hops)}

        aliases_tr = {alias_set_by_ip[hop[0]]: hop[0] for hop in tr_hops if hop[0] in alias_set_by_ip}
        p2p_tr = {str(ipaddress.ip_network(str(ipaddress.ip_address(hop[0])) + "/30", strict=False)): tr_hops[i - 1][0]
                  for i, hop in enumerate(tr_hops)
                  if i > 0
                  }
        if with_p2p:
            aliases_tr.update(p2p_tr)
        # Two types of constraints
        # Alias and p2p link
        for dst_tr_hop, ping_id_rr_hops in tr_hop_ping_id_rr_hops.items():
            for ping_id, rr_hops in ping_id_rr_hops.items():
                # Sort the rr hops by their index
                rr_hops = sorted(rr_hops, key=lambda x: x[1])
                # Get the aliases in rr and tr hops
                matches_rr = [(i, hop[0], aliases_tr[alias_set_by_ip[hop[0]]], "alias")
                              for i, hop in enumerate(rr_hops)
                              if hop[0] in alias_set_by_ip
                              and alias_set_by_ip[hop[
                        0]] in aliases_tr  # Case where the rr hop is in another alias set not present in tr
                              and hop[0] not in index_per_tr_hop
                              ]
                if with_p2p:
                    matches_p2p_rr = [(i, hop[0],
                                       aliases_tr[str(ipaddress.ip_network(str(ipaddress.ip_address(hop[0])) + "/30",
                                                                           strict=False))],
                                       "p2p")
                                      for i, hop in enumerate(rr_hops)
                                      if str(
                            ipaddress.ip_network(str(ipaddress.ip_address(hop[0])) + "/30", strict=False)) in p2p_tr
                                      and hop[0] not in index_per_tr_hop]
                    matches_rr.extend(matches_p2p_rr)
                    matches_rr.sort(key=lambda x: x[0])

                for i, rr_hop in enumerate(rr_hops):
                    if rr_hop[0] in index_per_tr_hop:
                        continue
                    previous_tr_hop = index_per_tr_hop[dst_tr_hop]
                    next_hop_rr_alias = None
                    for j, _, alias_tr, from_ in matches_rr:
                        # Find the closest tr hops before and after this rr hop.
                        if i == j:
                            previous_tr_hop = index_per_tr_hop[alias_tr]
                            next_hop_rr_alias = previous_tr_hop
                            break
                        elif i > j:
                            candidate_previous_tr_hop_index = index_per_tr_hop[alias_tr]
                            if candidate_previous_tr_hop_index > previous_tr_hop:
                                previous_tr_hop = candidate_previous_tr_hop_index
                        elif i < j:
                            # It is a candidate after the RR hop, so the closest after the RR hop
                            next_hop_rr_alias = index_per_tr_hop[alias_tr]
                            break

                    if next_hop_rr_alias is None:
                        next_hop_rr_alias = max(index_per_tr_hop.values())
                    if rr_hop[0] not in intersects_per_rr_hop:
                        intersects_per_rr_hop[rr_hop[0]] = previous_tr_hop, next_hop_rr_alias
                    else:
                        # Look if we found a smaller subpath
                        current_previous_tr_hop, current_next_tr_hop = intersects_per_rr_hop[rr_hop[0]]
                        if previous_tr_hop > current_previous_tr_hop:
                            current_previous_tr_hop = previous_tr_hop
                        if next_hop_rr_alias < current_next_tr_hop:
                            current_next_tr_hop = next_hop_rr_alias
                        intersects_per_rr_hop[rr_hop[0]] = current_previous_tr_hop, current_next_tr_hop

        # Use the constraints found with the successors, predecessors of the rr_hop to see
        # if we can do better
        intersects_per_rr_hop_with_pred_succ = {}
        for rr_hop in intersects_per_rr_hop:
            previous_tr_hop, next_tr_hop = intersects_per_rr_hop[rr_hop]
            if rr_hop in successors_per_hop:
                succ_tr_hop = [intersects_per_rr_hop[successor][1] for successor in successors_per_hop[rr_hop] if
                               successor in intersects_per_rr_hop]
                if len(succ_tr_hop) > 0:
                    min_succ_tr_hop = min(succ_tr_hop)
                    if min_succ_tr_hop < next_tr_hop:
                        next_tr_hop = min_succ_tr_hop
            if rr_hop in predecessors_per_hop:
                pred_tr_hop = [intersects_per_rr_hop[predecessor][0] for predecessor in predecessors_per_hop[rr_hop] if
                               predecessor in intersects_per_rr_hop]
                if len(pred_tr_hop) > 0:
                    max_pred_tr_hop = max(pred_tr_hop)
                    if max_pred_tr_hop > previous_tr_hop:
                        previous_tr_hop = max_pred_tr_hop
            intersects_per_rr_hop_with_pred_succ[rr_hop] = previous_tr_hop, next_tr_hop

        # print(tr_id)
        # Now insert the subpaths intersection into the database

        for rr_hop, (tr_hop_start_index, tr_hop_end_index) in intersects_per_rr_hop_with_pred_succ.items():
            tr_ttl_start = tr_hops[tr_hop_start_index][1]
            tr_ttl_end = tr_hops[tr_hop_end_index][1]
            rr_intersection_rows_to_insert.append((tr_id, rr_hop, tr_ttl_start, tr_ttl_end))

    query = (
        f" INSERT INTO atlas_rr_intersection (traceroute_id, rr_hop, tr_ttl_start, tr_ttl_end) "
        f" VALUES (%s, %s, %s, %s) "
    )
    cursor.executemany(query, rr_intersection_rows_to_insert)

def match_rr_tr_hops_traceroute_ids(traceroute_ids, with_p2p):
    connection = get_connection("traceroute_atlas", autocommit=0)
    traceroute_ids = list(traceroute_ids)
    for i  in range(0, len(traceroute_ids), 1000):
        max_index = i + 1000
        if i + 1000 > len(traceroute_ids):
            max_index = len(traceroute_ids)
        in_clause= f"".join([f",{traceroute_id}" for traceroute_id in traceroute_ids[i:max_index]])[1:]

        query = (
            f" SELECT traceroute_id, ping_id, tr_hop, rr_hop, rr_hop_index "
            f" FROM atlas_rr_pings arrp "
            f" INNER JOIN atlas_traceroutes at ON at.id = arrp.traceroute_id "
            f" WHERE at.id in ({in_clause}) "
            f" ORDER BY traceroute_id, ping_id, rr_hop_index "
        )
        cursor = connection.cursor()
        cursor.execute(query, )
        # rows = cursor.fetchmany(batch_size)
        rows = cursor.fetchall()
        if len(rows) == 0:
            return
        # print(f"Fetched {len(rows)} rows")
        i = 0
        rr_hops_per_tr_id = {}
        ips_for_alias_set = set()
        for row in rows:
            i += 1
            tr_id, ping_id, tr_hop, rr_hop, rr_hop_index = row
            if ipaddress.ip_address(rr_hop).is_private:
                continue
            ips_for_alias_set.add(rr_hop)
            # rr hops are ordered
            rr_hops_per_tr_id.setdefault(tr_id, {}).setdefault(tr_hop, {}).setdefault(ping_id, []).append(
                (rr_hop, rr_hop_index))

        query = (
            f" SELECT src, dest, trace_id, ath.hop, ath.ttl"
            f" FROM atlas_traceroute_hops ath"
            f" INNER JOIN atlas_traceroutes at ON at.id = ath.trace_id "
            f" WHERE at.id in ({in_clause}) "
            f" ORDER BY trace_id, ttl"
        )

        cursor.execute(query, )
        # rows = cursor.fetchmany(batch_size)
        rows = cursor.fetchall()
        src_dst_per_tr_id = {}
        tr_hops_per_tr_id = {}
        for row in rows:
            src, dest, trace_id, hop, ttl = row
            tr_hops_per_tr_id.setdefault(trace_id, []).append((hop, ttl))
            src_dst_per_tr_id[trace_id] = (src, dest)

        match_rr_tr_hops(cursor, ips_for_alias_set, rr_hops_per_tr_id, tr_hops_per_tr_id, with_p2p)

        connection.commit()
        cursor.close()
    connection.close()

def compute_tr_hops_by_tr_id(cursor, src, label):
    query = (
        f" SELECT src, dest, trace_id, ath.hop, ath.ttl"
        f" FROM atlas_traceroute_hops ath"
        f" INNER JOIN atlas_traceroutes at ON at.id = ath.trace_id "
        f" WHERE at.dest={src} "
    )
    if label is not None:
        query += f" AND at.platform='{label}' "
    query += (
        f" ORDER BY trace_id, ttl"
    )

    cursor.execute(query, )
    # rows = cursor.fetchmany(batch_size)
    rows = cursor.fetchall()
    src_dst_per_tr_id = {}
    tr_hops_per_tr_id = {}
    for row in rows:
        src, dest, trace_id, hop, ttl = row
        tr_hops_per_tr_id.setdefault(trace_id, []).append((hop, ttl))
        src_dst_per_tr_id[trace_id] = (src, dest)

    return tr_hops_per_tr_id, src_dst_per_tr_id

def compute_rr_hops_by_tr_id(cursor, src, label):
    query = (
        f" SELECT traceroute_id, ping_id, tr_hop, rr_hop, rr_hop_index "
        f" FROM atlas_rr_pings arrp "
        f" INNER JOIN atlas_traceroutes at ON at.id = arrp.traceroute_id "
        f" WHERE at.dest={src} "
    )
    if label is not None:
        query += f" AND at.platform='{label}' "
    query += (
        f" ORDER BY traceroute_id, ping_id, rr_hop_index "

    )

    cursor.execute(query, )
    # rows = cursor.fetchmany(batch_size)
    rows = cursor.fetchall()
    if len(rows) == 0:
        return
    # print(f"Fetched {len(rows)} rows")
    i = 0
    rr_hops_per_tr_id = {}
    ips_for_alias_set = set()
    for row in rows:
        i += 1
        tr_id, ping_id, tr_hop, rr_hop, rr_hop_index = row
        if ipaddress.ip_address(rr_hop).is_private:
            continue
        ips_for_alias_set.add(rr_hop)
        # rr hops are ordered
        rr_hops_per_tr_id.setdefault(tr_id, {}).setdefault(tr_hop, {}).setdefault(ping_id, []).append(
            (rr_hop, rr_hop_index))
    return rr_hops_per_tr_id, ips_for_alias_set

def match_rr_tr_hops_all(with_p2p, label):
    vps = get_vps()

    for src in vps:
        print(src)
        connection = get_connection("traceroute_atlas", autocommit=0)
        cursor = connection.cursor()

        rr_hops_per_tr_id_ips_for_alias_set = compute_rr_hops_by_tr_id(cursor, src, label)
        if rr_hops_per_tr_id_ips_for_alias_set is not None:
            rr_hops_per_tr_id, ips_for_alias_set = rr_hops_per_tr_id_ips_for_alias_set
            tr_hops_per_tr_id, src_dst_per_tr_id = compute_tr_hops_by_tr_id(cursor, src, label)
            match_rr_tr_hops(cursor, ips_for_alias_set, rr_hops_per_tr_id, tr_hops_per_tr_id, with_p2p)

        connection.commit()
        connection.close()


if __name__ == "__main__":

    '''
    Run the intersection algorithm to find with RR hop corresponds to which TrHop in traceroutes. 
    '''

    ######################################################################
    ## Parameters
    ######################################################################
    parser = argparse.ArgumentParser()
    parser.add_argument("-m", "--mode", help="mode, either all, label or selection", type=str)
    parser.add_argument("-l", "--label", help="label of the traceroutes in the database", type=str)
    parser.add_argument("--id", help="traceroute id", type=int)
    parser.add_argument("--ids-file", help="file containing traceroute ids", type=str)


    args = parser.parse_args()

    if not args.mode or args.mode not in ["all", "selection", "label"]:
        print(parser.error("mode option is mandatory, values are either all or single"))
        exit(1)

    if args.mode == "all":
        match_rr_tr_hops_all(with_p2p=True, label=args.label)
    elif args.mode == "selection":
        traceroute_ids = set()
        with open(args.ids_file) as f:
            for line in f:
                traceroute_ids.add(int(line.strip("\n")))

        match_rr_tr_hops_traceroute_ids(traceroute_ids, with_p2p=True)
    elif args.mode == "label":
        match_rr_tr_hops_all(with_p2p=True, label=args.label)
