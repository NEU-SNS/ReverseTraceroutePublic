from clickhouse_driver import Client
import time
import pyasn
import numpy as np
import functools
import ipaddress
import random
from algorithms.utils import find, generalized_jaccard
from algorithms.rr_heuristics import compute_heuristics
from algorithms.database import dropTable, createRRPingsTable, clickhouse_pwd, db_host, dump_query, insert_from_file, ip2asn
from algorithms.selection_technique import RRVPSelector
from network_utils import dnets_of, bgpDBFormat, prefix_24_from_ip, ipv4_index_of, ipItoStr
from multiprocessing import Pool

import scipy.spatial.distance
import logging
logging.basicConfig()
logger = logging.getLogger("ic")
logger.setLevel(logging.INFO)
# is_debug = True
# is_debug = False

def createIngressCoverTable(host, password, db, table):
    if password is None:
        client = Client(host)
    else:
        client = Client(host, password=password)

    client.execute(
        f"CREATE TABLE IF NOT EXISTS {db}.{table} ("
        f"dst_bgp_prefix_start UInt32, dst_bgp_prefix_end UInt32, "  # Network metadata of dst
        f"site String, src UInt32, ingress UInt32, distance Float64, rank UInt32 "
        # f"vp_site String "
        f")"
        f"ENGINE=MergeTree() "
        f"ORDER BY (dst_bgp_prefix_start, dst_bgp_prefix_end, rank)"

    )

def greedy_set_cover(ingresses_metrics, uncovered_vps):
    selected_ingress_search = set()
    selected_ingress = []
    while len(uncovered_vps) > 0:
        # Select the set with the biggest intersection
        best_current_ingress = None
        best_weight = -1
        best_additional_covered_vps = set()

        for ingress, metrics in ingresses_metrics.items():
            if ingress in selected_ingress_search:
                continue
            ingress_consistency, ingress_vps, ingress_ratio = metrics
            additional_covered_vps = uncovered_vps.intersection(ingress_vps)
            weight = len(additional_covered_vps) * ingress_ratio * (1 - ingress_consistency)
            if best_weight < weight and len(additional_covered_vps) > 0:
                best_current_ingress = ingress
                best_additional_covered_vps = additional_covered_vps
                best_weight = weight

        selected_ingress_search.add(best_current_ingress)
        selected_ingress.append((best_current_ingress, best_additional_covered_vps))
        uncovered_vps = uncovered_vps - best_additional_covered_vps

    return selected_ingress


def greedy_minimum_set_cover(vps_per_ingress, uncovered_vps):
    selected_ingress_search = set()
    selected_ingress = []
    while len(uncovered_vps) > 0:
        # Select the set with the biggest intersection
        best_current_ingress = None
        best_additional_covered_vps = set()

        for ingress, vps in vps_per_ingress.items():
            if ingress in selected_ingress_search:
                continue
            additional_covered_vps = uncovered_vps.intersection(vps)
            if len(additional_covered_vps) > len(best_additional_covered_vps):
                best_current_ingress = ingress
                best_additional_covered_vps = vps

        selected_ingress_search.add(best_current_ingress)
        selected_ingress.append((best_current_ingress, best_additional_covered_vps))
        uncovered_vps = uncovered_vps - best_additional_covered_vps

    return selected_ingress


def backward_ingress(ingress,
                     group,
                     rr_sub_paths_per_src,
                     potential_ingresses,
                     ):

    # Look until where we can backward with two conditions:
    # (1) No convergence point anymore backward
    # (2) The last IPs set should always be a subset of the potential ingresses
    ingress_indexes = {src: find([x[0] for x in rr_sub_paths_per_src[src]], ingress)[0] for src in group}
    i = 1
    n_last_ip_per_backward = {}
    while True:

        if any([len(rr_sub_paths_per_src[src][:ingress_indexes[src]-i]) == 0 for src in group]):
            break
        # Backward until first hop so that there is no convergence points before that hop
        last_ips = set(rr_sub_paths_per_src[src][ingress_indexes[src]-i][0] for src in group)
        if not last_ips.issubset(potential_ingresses):
            break
        n_last_ip_per_backward[i] = len(last_ips)
        i += 1


    # At this step, look how many hops we can backward without losing ingress infos.
    # We care here about finding LB patterns and not take it for different ingresses.
    is_load_balancing = False
    is_maybe_in_load_balancing = False
    for i, n_paths in sorted(n_last_ip_per_backward.items()):
        if n_paths > 1:
            is_maybe_in_load_balancing = True
        elif n_paths == 1:
            if is_maybe_in_load_balancing:
                is_load_balancing = True
                break

    if is_load_balancing:
        # We have a load balanced paths, so we can skip backward until the divergence point
        backward = i - 1
    else:
        backward = 1

    ingress_to_re_add = set()
    new_ingress_groups = {}
    for src in group:
        # rr_sub_paths_per_src[src] = rr_sub_paths_per_src[src][:-backward]
        # Proceed backward, the new ingress candidates are the last IPs of each rr path
        if len(rr_sub_paths_per_src[src]) == 0:
            # If there is none, the current is the minimal ingress
            return {}
        new_ingress = rr_sub_paths_per_src[src][ingress_indexes[src]-backward][0]
        if new_ingress not in potential_ingresses:
            continue
        new_ingress_groups.setdefault(new_ingress, set()).add(src)
        ingress_to_re_add.add(new_ingress)
    return ingress_to_re_add, new_ingress_groups


def compute_ingresses(dst_src_rr_paths_training, dst_src_rr_paths_evaluation, ip2asn):
    '''
    Ingress can be anywhere between the first hop before the AS and the destination (included)
    Conditions for an ingress to be well defined, in that priority order:
    (1) An ingress should be consistent (i.e., VPs at the same distance of the ingress should either all
    fail or all succeed)
    (2) An ingress should cover the biggest number of VPs
    (3) An ingress should (ideally) be in both training and RR path
    :param dst_src_rr_paths_training:
    :param src_dst_rr_paths_evaluation:
    :param ip2asn:
    :return:
    '''

    # First remove the subpaths ]d, ] from all the subpaths
    if len(dst_src_rr_paths_training) == 0:
        return [], {}, {}, None

    technique = "ingress"

    # rr_sub_paths_per_src = {x[0]: x[2] for x in src_dst_rr_paths}
    # rr_sub_paths_per_src_debug = {x[0]: (x[1], x[2]) for x in src_dst_rr_paths}
    dst_training, src_rr_paths_training = dst_src_rr_paths_training
    dst_evaluation, src_rr_paths_evaluation = dst_src_rr_paths_evaluation
    rr_sub_paths_per_src_training = {x[0]: (dst_training, x[1]) for x in src_rr_paths_training}
    rr_sub_paths_per_src_evaluation = {x[0]: (dst_evaluation, x[1]) for x in src_rr_paths_evaluation}



    # Initialize the groups by their first IP in the BGP prefix
    # (or the first IP before the destination (could be found via heuristics) if none in the BGP prefix)
    # (or the last  IP if no destination found either by normal or heuristics
    dentry_dst     = dnets_of(dst_training, ip2asn, ip_representation="uint32")
    dst_asn        = dentry_dst.asn
    dst_bgp_prefix = bgpDBFormat(dentry_dst)

    # ingress previous is a structure where each potential ingress has a parent

    ingress_groups            = {}
    # Potential ingresses correspond to the first hop before the first IP in the dst AS until the destination
    # they could all be ingress. Before that, no.
    potential_ingresses   = {}

    vp_reached_destination = {}
    is_fall_back_on_destination = False
    vp_reached_as = {}
    # The ingress can be anywhere between the first hop before entering in the AS and the destination.
    for src_rr_paths in src_rr_paths_training:
        src, rr_path_training = src_rr_paths
        dst_src_rr_path = (src, dst_training, rr_path_training)
        if src not in rr_sub_paths_per_src_evaluation:
            continue
        _, rr_path_evaluation = rr_sub_paths_per_src_evaluation[src]
        rr_path_evaluation_ip = [x[0] for x in rr_path_evaluation]
        rr_path_heuristics = compute_heuristics(dst_src_rr_path, ip2asn)
        use_heuristic = False
        if rr_path_heuristics is not None:
            use_heuristic = True
            _, _, _, new_index, technique = rr_path_heuristics

        # new_dst because of the heuristics
        last_hop_in_as_before_dst = None
        # logger.debug(src_dst_rr_path_to_string(src, (dst, rr_path)))
        for i, (rr_ip, rr_hop) in enumerate(rr_path_training):

            dentry_rr     = dnets_of(rr_ip, ip2asn, ip_representation="uint32")
            rr_asn        = dentry_rr.asn
            rr_bgp_prefix = bgpDBFormat(dentry_rr)
            if dst_asn == rr_asn:
                vp_reached_as.setdefault(dst_training, set()).add(src)
                # Add the hop before entering the asn
                # Look if the ingress is also in the evaluation path.
                potential_ingress_before = rr_path_training[i - 1][0]
                potential_ingress = rr_ip
                if potential_ingress_before in rr_path_evaluation_ip:
                    potential_ingresses.setdefault(potential_ingress_before, set()).add(src)
                else:
                    # If private IP address, go back until we have a non private IP address.
                    k = i
                    while ipaddress.ip_address(potential_ingress_before).is_private:
                        k -=  1
                        if k - 1 < 0:
                            break
                        potential_ingress_before = rr_path_training[k - 1][0]
                        if potential_ingress_before in rr_path_evaluation_ip:
                            potential_ingresses.setdefault(potential_ingress_before, set()).add(src)
                            break

                if potential_ingress in rr_path_evaluation_ip:
                    potential_ingresses.setdefault(potential_ingress, set()).add(src)
                # if new_dst != rr_ip:
                #     potential_ingresses_minus_dest.add(rr_sub_path[i-1][0])
                #     potential_ingresses_minus_dest.add(rr_ip)
                last_hop_in_as_before_dst = (i, rr_ip)
            if use_heuristic:
                if i == new_index:
                    potential_ingress_before = rr_path_training[i - 1][0]
                    potential_ingress = rr_ip
                    if potential_ingress_before in rr_path_evaluation_ip:
                        potential_ingresses.setdefault(potential_ingress_before, set()).add(src)
                    if potential_ingress in rr_path_evaluation_ip:
                        potential_ingresses.setdefault(potential_ingress, set()).add(src)
                    if technique == "ip_loop":
                        last_hop_in_as_before_dst = (i-1, rr_path_training[i - 1][0])
                        # In the case of ip loop potential ingress are considered all along the loop.
                        for rr_ip_loop, rr_hop_loop in reversed(rr_path_training[:new_index]):
                            if rr_ip_loop ==  rr_path_training[i][0]:
                                break
                            potential_ingress = rr_ip_loop
                            if potential_ingress in rr_path_evaluation_ip:
                                potential_ingresses.setdefault(potential_ingress, set()).add(src)

                    else:
                        last_hop_in_as_before_dst = (i, rr_ip)
                    break
            if rr_ip == dst_training:
                potential_ingress_before = rr_path_training[i - 1][0]
                potential_ingress = rr_ip
                if potential_ingress_before in rr_path_evaluation_ip:
                    potential_ingresses.setdefault(potential_ingress_before, set()).add(src)
                if potential_ingress in rr_path_evaluation_ip:
                    potential_ingresses.setdefault(potential_ingress, set()).add(src)
                # Special case, if the destination is one hop away,
                # relax the constraint
                if i == 0:
                    potential_ingresses.setdefault(potential_ingress, set()).add(src)
                # Second special case, if the destination is one hop away,
                # relax the constraint

                last_hop_in_as_before_dst = (i, rr_ip)
                vp_reached_destination.setdefault(dst_training, set()).add(src)
                break
            # if rr_bgp_prefix == dst_bgp_prefix:
            #     # First define the ingress as the first hop in the dst bgp prefix
            #     ingress_groups.setdefault(rr_ip, set()).add(src)
            #     # Keep the subpath until the last ingress candidate
            #     rr_sub_paths_per_src[src] = rr_sub_paths_per_src[src][:i+1]
            #     break
            # else:
            #     if rr_ip == new_dst:
            #         ingress_groups.setdefault(rr_sub_path[i-1][0], set()).add(src)
            #         rr_sub_paths_per_src[src] = rr_sub_paths_per_src[src][:i+1]
            #         break
        if last_hop_in_as_before_dst is not None:
            i, rr_ip = last_hop_in_as_before_dst
            ingress_groups.setdefault(rr_ip, set()).add(src)
            # Keep the subpath until the last ingress candidate
            # rr_sub_paths_per_src[src] = rr_sub_paths_per_src[src][:i+1]


    # If we have no ingress, set the ingress as the destination.
    if len(potential_ingresses) == 0:
        is_fall_back_on_destination = True
        potential_ingresses = vp_reached_destination
        if len(vp_reached_destination) > 0:
            # for _, vps in vp_reached_destination.items():
            #     for vp in vps:
            #         logger.debug(rr_sub_paths_per_src_training[vp])
            #         logger.debug(rr_sub_paths_per_src_evaluation[vp])
            # logger.info("Falling back on destination cover, no ingress found, but reached destination")
            technique = "destination"

    # Look if we had only one possible ingress
    src_dst_rr_paths_d = {x[0]: (dst_training, x[1]) for x in src_rr_paths_training}
    evaluation_src_dst_rr_paths_d = {x[0]: (dst_evaluation, x[1]) for x in src_rr_paths_evaluation}
    # If we have no ingress to test, we are done
    for vp in src_dst_rr_paths_d:

        if vp in evaluation_src_dst_rr_paths_d:
            logger.debug(src_dst_rr_path_to_string(vp, src_dst_rr_paths_d[vp]))
            logger.debug(src_dst_rr_path_to_string(vp, evaluation_src_dst_rr_paths_d[vp]))
            logger.debug("\n")

    if len(potential_ingresses) == 0:
        logger.debug("No ingress")
        for vp in src_dst_rr_paths_d:

            if vp in evaluation_src_dst_rr_paths_d:
                logger.debug(src_dst_rr_path_to_string(vp, src_dst_rr_paths_d[vp]))
                logger.debug(src_dst_rr_path_to_string(vp, evaluation_src_dst_rr_paths_d[vp]))
                logger.debug("\n")
        if len(vp_reached_as) == 0:
            technique = "no_reach_as"
        else:
            technique = "no_ingress"
        return [], {}, {}, technique

    '''
    Apply a greedy set cover on the ingress to find the ingresses.
    '''
    uncovered_vps = set.union(*potential_ingresses.values())
    selected_ingresses = greedy_minimum_set_cover(potential_ingresses, uncovered_vps)
    sorted_vps_by_distance_per_ingress = {}
    closest_distance_per_ingress = {}
    ingresses = []
    ingress_per_vp = {}
    for ingress, vps in selected_ingresses:

        logger.debug(("Ingress", str(ipaddress.ip_address(ingress)), ingress,
                      len(vps)
                      ))
        ingresses.append(ingress)
        for i, vp in enumerate(vps):
            # Compute the distance to the ingress in both training and evaluation
            dst, rr_path_training = src_dst_rr_paths_d[vp]
            rr_path_training = [r[0] for r in rr_path_training]
            ingress_index_training, _ = find(rr_path_training, ingress)
            distances_to_ingress = []
            if dst == ingress and ingress_index_training == 0:
                # Special case when dst is only one hop away:
                ingress_index_training = -1
                logger.debug(src_dst_rr_path_to_string(vp, src_dst_rr_paths_d[vp]))
                logger.debug(src_dst_rr_path_to_string(vp, evaluation_src_dst_rr_paths_d[vp]))
                logger.debug("\n")
            else:
                dst, rr_path_evaluation = evaluation_src_dst_rr_paths_d[vp]
                rr_path_evaluation = [r[0] for r in rr_path_evaluation]
                if not is_fall_back_on_destination:
                    ingress_index_evaluation, _ = find(rr_path_evaluation, ingress)
                    distances_to_ingress.append(ingress_index_evaluation)


            distances_to_ingress.append(ingress_index_training)
            ingress_per_vp[vp] = ingress
            sorted_vps_by_distance_per_ingress.setdefault(ingress, []).append((vp, np.mean(distances_to_ingress)))

        sorted_vps_by_distance_per_ingress[ingress].sort(key=lambda x: x[1])
        closest_distance_per_ingress[ingress] = sorted_vps_by_distance_per_ingress[ingress][0][1]
    # Sort the ingress by the distance to their closest VP.
    ingresses = sorted(closest_distance_per_ingress.items(), key=lambda x: x[1])
    ingresses = [x[0] for x in ingresses]
    return ingresses, ingress_per_vp, sorted_vps_by_distance_per_ingress, technique


    # Metrics for selecting the ingresses
    consistency_per_ingress = {}
    vps_per_ingress = {}
    in_both_training_and_evaluation_ratio_per_ingress = {}

    ingress_per_vp = {}
    # if len(potential_ingresses_minus_dest) <= 1:
    #     print("Only one ingress", ipaddress.ip_address(list(ingress_groups.keys())[0]))
    #     for ingress, vps in ingress_groups.items():
    #         for vp in vps:
    #             # Distance VP - Ingress
    #             rr_sub_path = src_dst_rr_paths_d[vp][1]
    #             # Compute the distance to the ingress
    #             rr_sub_path_ip = [r[0] for r in rr_sub_path]
    #             ingress_index, _ = find(rr_sub_path_ip, ingress)
    #             distance = ingress_index
    #             ingress_per_vp[vp] = ingress
    #             vps_per_ingress.setdefault(ingress,[]).append((vp, distance))
    #
    #             print_src_dst_rr_path(src_dst_rr_paths_d[vp])
    #             print_src_dst_rr_path(evaluation_src_dst_rr_paths_d[vp])
    #             print()
    #
    #     return ingress_per_vp, vps_per_ingress


    ingress_done = set()
    while True:
        # Now that ingress groups have been initialized, try to define ingress backward
        # and see if it produces better results
        new_ingress_groups = {}
        ingress_to_re_add = set()
        # First check (1)
        for ingress, group in ingress_groups.items():
            logger.debug(f"Evaluating {str(ipaddress.ip_address(ingress))}, {len(group)}")
            if ingress in ingress_done:
                new_ingress_groups[ingress] = group
                continue

            # Check two things, consistency BEFORE ingress
            # (distance from the VP to the ingress in both training and
            # evaluation) and consistency AFTER ingress (distance from the ingress to the destination) by distance.
            distance_to_ingress_per_src  = {}
            distances_from_ingress_to_dst_training = []
            distances_from_ingress_to_dst_evaluation = []

            # For debug, VPs that are at the same distance to their ingress,
            # but not at the same distance
            # from the destination
            distance_ingress_dst_per_distance_vp_debug = {}

            for src in group:
                dst_training, rr_path_training_ip_hop = rr_sub_paths_per_src_debug[src]
                dst_evaluation, rr_path_evaluation_ip_hop = rr_sub_paths_per_src_evaluation[src]
                # logger.debug(src_dst_rr_path_to_string(src, rr_sub_paths_per_src_debug[src]))
                # Compute the distance to the ingress
                rr_path_training = [r[0] for r in rr_path_training_ip_hop]

                rr_path_evaluation = [r[0] for r in rr_path_evaluation_ip_hop]
                debug_index = find(rr_path_training, ingress)
                ingress_index_training, _   = find(rr_path_training, ingress)
                distance_to_ingress_per_src.setdefault(src, []).append(ingress_index_training)

                # Distance vp ingress in the evaluation path
                ingress_index_evaluation = find(rr_path_evaluation, ingress)
                is_ingress_in_evaluation_path = True
                if ingress_index_evaluation is None:
                    is_ingress_in_evaluation_path = False
                    # If the ingress is not in the evaluation path, ignore it as this will be counted in the
                    # other constraint
                    ingress_index_evaluation = ingress_index_training
                else:
                    ingress_index_evaluation = ingress_index_evaluation[0]

                distance_to_ingress_per_src.setdefault(src, []).append(ingress_index_evaluation)

                dst_index_training   = find(rr_path_training, dst_training)
                dst_index_evaluation = find(rr_path_evaluation, dst_evaluation)
                if dst_index_training is not None:
                    distance_from_ingress_to_dst_training = dst_index_training[0] - ingress_index_training
                else:
                    distance_from_ingress_to_dst_training = 8 - ingress_index_training
                distances_from_ingress_to_dst_training.append(distance_from_ingress_to_dst_training)
                if dst_index_evaluation is not None:
                    distance_from_ingress_to_dst_evaluation = dst_index_evaluation[0] - ingress_index_evaluation
                else:
                    distance_from_ingress_to_dst_evaluation = 8 - ingress_index_evaluation
                distances_from_ingress_to_dst_evaluation.append(distance_from_ingress_to_dst_evaluation)

                if is_ingress_in_evaluation_path:
                    distance_ingress_dst_per_distance_vp_debug.setdefault(
                        distance_from_ingress_to_dst_evaluation, []).append(src)
                    # distance_ingress_dst_per_distance_vp_debug.setdefault(
                    #     distance_from_ingress_to_dst_training, []).append(src)

            # Two types of inconsistencies: dist(vp, i) et dist(i, d)
            error_dist_vp_ingress = sum([(distance_to_ingress_per_src[src][0] - distance_to_ingress_per_src[src][1])**2
                                   for src in distance_to_ingress_per_src])
            error_dist_ingress_dst_training = np.std(distances_from_ingress_to_dst_training)
            error_dist_ingress_dst_evaluation = np.std(distances_from_ingress_to_dst_evaluation)

            if error_dist_vp_ingress > 0 or error_dist_ingress_dst_training > 0 or error_dist_ingress_dst_evaluation > 0:
                logger.debug(f"Ingress {str(ipaddress.ip_address(ingress))}")
                if error_dist_vp_ingress > 0:
                    for src in distance_to_ingress_per_src:
                        distances = distance_to_ingress_per_src[src]
                        if distances[0] != distances[1]:
                            logger.debug(f"Distance vp ingress changes")
                            dst_training, rr_path_training = rr_sub_paths_per_src_debug[src]
                            dst_evaluation, rr_path_evaluation = rr_sub_paths_per_src_evaluation[src]
                            logger.debug(src_dst_rr_path_to_string(src, (dst_training, rr_path_training)))
                            logger.debug(src_dst_rr_path_to_string(src, (dst_evaluation, rr_path_evaluation)))
                if  error_dist_ingress_dst_evaluation > 0:
                    for distance_ingress_dst, vps in distance_ingress_dst_per_distance_vp_debug.items():
                        logger.debug(f"Distance ingress destination {distance_ingress_dst}")
                        src = vps[0]
                        dst_training, rr_path_training = rr_sub_paths_per_src_debug[src]
                        dst_evaluation, rr_path_evaluation = rr_sub_paths_per_src_evaluation[src]
                        logger.debug(src_dst_rr_path_to_string(src, (dst_training, rr_path_training)))
                        logger.debug(src_dst_rr_path_to_string(src, (dst_evaluation, rr_path_evaluation)))
            print()
            print()

            # If the ratio for each hop is not very high or very low or the variance of distance to destination is high
            # , the ingress is probably wrong
            # The ideal definition of ingress would be that the vector success_ratio_per_distance is only 0 or 1

            # optimal_success_ratio_per_distance = [0 if x < 0.5 else 1 for x in success_ratio_per_distance]
            # consistency = scipy.spatial.distance.euclidean(optimal_success_ratio_per_distance, success_ratio_per_distance)
            consistency = error_dist_vp_ingress + error_dist_ingress_dst_training + error_dist_ingress_dst_evaluation
            # if len(variance_distance_to_dst_per_distance) > 0:
            consistency_per_ingress[ingress] = consistency
            if consistency > 0:
                logger.debug("Non zero consistency")
                for src in group:
                    rr_path_training = rr_sub_paths_per_src[src]
                    # logger.debug(src_dst_rr_path_to_string(src, (dst, rr_path)))
            else:
                consistency_per_ingress[ingress] = 100

            # Check (2), number of VPs per ingress
            vps_per_ingress[ingress] = group

            # Check (3), is the ingress in both training and evaluation?
            in_both_training_and_evaluation_ratio = 0
            for src in group:
                # Check that the ingress is in both training and evaluation rr paths
                training_rr_sub_path = rr_sub_paths_per_src[src]
                dst_evaluation, evaluation_rr_sub_path = rr_sub_paths_per_src_evaluation[src]
                # training_rr_sub_path = [x[0] for x in training_rr_sub_path]
                evaluation_rr_sub_path = [x[0] for x in evaluation_rr_sub_path]
                if ingress in evaluation_rr_sub_path:
                    # The ingress was not in both training and the evaluation
                    in_both_training_and_evaluation_ratio += 1
            in_both_training_and_evaluation_ratio /= len(group)
            in_both_training_and_evaluation_ratio_per_ingress[ingress] = in_both_training_and_evaluation_ratio
            # We are done with this ingress
            for vp in group:
                new_ingress_groups.setdefault(ingress, set()).add(vp)
            # Check if we can mark the ingress as done.
            ingress_done.add(ingress)
            # if consistency == 0 and in_both_training_and_evaluation_ratio == 1:
            #     # This means that we have found an optimal ingress, so just continue, no need to backward
            #     continue
            # else:
            ingress_not_done, new_ingress_groups_to_add = \
                backward_ingress(
                    ingress,
                    group,
                    rr_sub_paths_per_src,
                    potential_ingresses
                    )
            ingress_to_re_add.update(ingress_not_done)
            for ingress, vps in new_ingress_groups_to_add.items():
                new_ingress_groups.setdefault(ingress, set()).update(vps)
        # Re-add the ingress that still need to go backward because they have been found
        # by backwarding on other ingresses

        for ingress in ingress_to_re_add:
            if ingress in ingress_done:
                ingress_done.remove(ingress)
        logger.debug(f"Ingress done {[str(ipaddress.ip_address(x)) for x in ingress_done]}")
        # Stopping condition is that we have no new ingress to test.
        if set(ingress_groups.keys()) == set(new_ingress_groups.keys()) or new_ingress_groups == {}:
            # We are done backwarding
            break
        ingress_groups = new_ingress_groups



            # rr_sub_paths_per_src_to_ingress = {}
            # rr_sub_paths_per_src_ingress_destination_evaluation = {}
            # for src, rr_sub_path in rr_sub_paths_per_src.items():
            #     rr_sub_path_ip = [r[0] for r in rr_sub_path]
            #     ingress_index, _ = find(rr_sub_path_ip, ingress)
            #     rr_sub_paths_per_src_to_ingress[src] = rr_sub_path_ip[:ingress_index]
            #
            #     # Same for evaluation
            #     rr_sub_path_ip = [r[0] for _ , r in evaluation_src_dst_rr_paths[src]]
            #     ingress_index, _ = find(rr_sub_path_ip, ingress)
            #     dst_index, _ = find(rr_sub_path_ip, dst)
            #     rr_sub_paths_per_src_ingress_destination_evaluation[src] = \
            #         rr_sub_path_ip[src][ingress_index:dst]
            #
            # # Compute similarity between the subpaths [:ingress]
            # similarity_src_ingress = generalized_jaccard(rr_sub_paths_per_src_to_ingress.values())
            # # Compute similarity between the subpaths [ingress:]
            # similarity_ingress_dst = generalized_jaccard(rr_sub_paths_per_src_ingress_destination_evaluation.values())

    # Based on the different scoring, define the ingresses
    metrics_per_ingress = {x : (
        consistency_per_ingress[x],
        vps_per_ingress[x],
        in_both_training_and_evaluation_ratio_per_ingress[x]
        )
        for x in ingress_groups
    }
    uncovered_vps = set()

    for _, vps in ingress_groups.items():
        uncovered_vps.update(vps)

    # Greedy set cover algorithm to select the ingress
    selected_ingress = greedy_set_cover(metrics_per_ingress, uncovered_vps)

    sorted_vps_by_distance_per_ingress = {}
    closest_distance_per_ingress = {}
    ingresses = []
    for ingress, vps in selected_ingress:

        logger.debug(("Ingress", str(ipaddress.ip_address(ingress)), ingress,
              consistency_per_ingress[ingress],
              in_both_training_and_evaluation_ratio_per_ingress[ingress],
              len(vps)
              ))
        ingresses.append(ingress)
        for i, vp in enumerate(vps):
            # Distance VP - Ingress
            dst, rr_path_training = src_dst_rr_paths_d[vp]
            # Compute the distance to the ingress
            rr_path_training = [r[0] for r in rr_path_training]
            debug_index = find(rr_path_training, ingress)
            ingress_index_training, _ = find(rr_path_training, ingress)

            if dst == ingress and ingress_index_training == 0:
                # Special case when dst is only one hop away:
                ingress_index_training = -1
            # ingress_index, _ = ingress_index
            distance_to_ingress = ingress_index_training
            ingress_per_vp[vp] = ingress
            sorted_vps_by_distance_per_ingress.setdefault(ingress, []).append((vp, distance_to_ingress))
            # logger.debug(src_dst_rr_path_to_string(vp, src_dst_rr_paths_d[vp]))
            # logger.debug(src_dst_rr_path_to_string(vp, evaluation_src_dst_rr_paths_d[vp]))
            # logger.debug("\n")
        # print()
        sorted_vps_by_distance_per_ingress[ingress].sort(key=lambda x:x[1])
        closest_distance_per_ingress[ingress] = sorted_vps_by_distance_per_ingress[ingress][0][1]
    # Sort the ingress by the distance to their closest VP.
    ingresses = sorted(closest_distance_per_ingress.items(), key=lambda x:x[1])
    ingresses = [x[0] for x in ingresses]
    return ingresses, ingress_per_vp, sorted_vps_by_distance_per_ingress

def src_dst_rr_path_to_string(vp, dst_rr_path):
    dst_rr_path = [str(ipaddress.ip_address(dst_rr_path[0])), [str(ipaddress.ip_address(ip)) for ip, hop in dst_rr_path[1]]]
    return f"{vp} {dst_rr_path}"

def compute_vps_in_range(src_dst_rr_paths, ip2asn, with_heuristics):
    rr_sub_paths = []
    for src_dst_rr_path in src_dst_rr_paths:
        src, dst, rr_hop_path = src_dst_rr_path
        # First build rr subpath between source and destination
        rr_path = [x[0] for x in rr_hop_path]
        index_rr_dst = find(rr_path, dst)
        # Check if there exists a hop after the destination
        # (to see if we are able to uncover reverse hops)
        if index_rr_dst is not None:
            index, _ = index_rr_dst
            distance = index
            rr_sub_paths.append((src, dst, rr_hop_path, distance, "ingress"))
            continue
        if with_heuristics:
            rr_path_heuristics = compute_heuristics(src_dst_rr_path, ip2asn)
            if rr_path_heuristics is not None:
                rr_sub_paths.append(rr_path_heuristics)
    return rr_sub_paths

def index_subpaths_biggest_common_suffix(rr_sub_paths):

    '''
    Modify the subpaths in place to remove their longest common suffix
    :param rr_sub_paths: a list of tuple (rr_ip, hop)
    :return:
    '''
    remove_index = 1
    while True:
        last_ips = set(x[1][0][-remove_index][0] for x in rr_sub_paths if len(x[1][0]) >= remove_index)
        if len(last_ips) == 1:
            remove_index += 1
        else:
            break
    return remove_index

def compute_convergence_points(src_dst_rr_paths, ip2asn):
    '''
    This algorithm finds convergence points between paths leading to the destination.
    :param src_dst_rr_paths:
    :param ip2asn:
    :return:
    '''
    if len(src_dst_rr_paths) == 0:
        return [] , {}
    rr_sub_paths_l = compute_vps_in_range(src_dst_rr_paths, ip2asn, with_heuristics=True, is_include_last_hop_dst=True)
    if len(rr_sub_paths_l) == 0:
        return [], {}
    rr_sub_paths_debug = {x[0]: (x[1], x[2], x[3]) for x in rr_sub_paths_l}
    rr_sub_paths = {x[0]: (x[1], x[2], x[3]) for x in rr_sub_paths_l}
    # Now take the set of last IP of each subpath to create the groups.
    rr_sub_paths_groups_per_recurse = {}
    # rr_sub_paths_groups_per_vp = {}
    # Maintain a structure with the groups per ingress
    recurse_level = 0

    # Put all the VPs in the same first group
    rr_sub_paths_groups_per_recurse.setdefault(0, {}).setdefault(src_dst_rr_paths[0][1], []).extend([(r[0], r[2], r[3]) for r in rr_sub_paths_l])

    while True:
        # Stopping condition
        if recurse_level not in rr_sub_paths_groups_per_recurse:
            break
        rr_sub_paths_groups_recurse_level = rr_sub_paths_groups_per_recurse[recurse_level]
        for convergence_point in rr_sub_paths_groups_recurse_level:
            rr_path_per_vp = rr_sub_paths_groups_recurse_level[convergence_point]
            # Filter the empty RR paths
            rr_sub_paths_cp = [(src, rr_sub_paths[src]) for src, _, _ in rr_path_per_vp if len(rr_sub_paths[src][0]) > 0]
            remove_index = index_subpaths_biggest_common_suffix([rr_sub_paths_cp[i] for i in range(len(rr_sub_paths_cp))])
            if remove_index > 1:
                for src, _ in rr_sub_paths_cp:
                    remove_index_min = min(remove_index, len(rr_sub_paths[src][0]))
                    rr_sub_paths[src] = (
                    rr_sub_paths[src][0][:-remove_index_min], rr_sub_paths[src][1], rr_sub_paths[src][2])


            last_ips = {}
            vps_per_last_ip = {}

            # Search for more convergence point
            for vp, _ , _ in rr_path_per_vp:
                rr_sub_path = rr_sub_paths[vp]
                if len(rr_sub_path[0]) == 0:
                    continue
                # The vantage point
                last_ip = rr_sub_path[0][-1][0]
                last_ips.setdefault(last_ip, 0)
                last_ips[last_ip] += 1
                vps_per_last_ip.setdefault(last_ip, []).append(vp)

            for last_ip, occ in last_ips.items():

                if occ > 1 or (occ == 1 and recurse_level == 0):
                    # There is a convergence point
                    vps = vps_per_last_ip[last_ip]
                    # This is the distance from the convergence point

                    for vp in vps:
                        distance_to_cp = len(rr_sub_paths[vp][0]) - 1
                        technique = rr_sub_paths[vp][2]
                        rr_sub_paths_groups_per_recurse.setdefault(recurse_level + 1, {}) \
                            .setdefault(last_ip, []) \
                            .append((vp, distance_to_cp, technique))
                        # rr_sub_paths_groups_per_vp.setdefault(vp, []).append((last_ip, recurse_level, distance_to_cp))

        recurse_level += 1

    # Remove the 0 recurse level which corresponds to subpaths without any computation
    # del rr_sub_paths_groups_per_recurse[0]

    # Compute the ingresses, they correspond to the convergence points
    convergence_points = [list(convergence_points.keys()) for recurse_level, convergence_points in rr_sub_paths_groups_per_recurse.items()]
    convergence_points = [cp for cps in convergence_points for cp in cps]

    return convergence_points, rr_sub_paths_groups_per_recurse

def compare_vp(vp1, vp2):
    techniques_order = {"ingress": 1, "double_stamp": 2, "ip_loop": 3, "enter_exit_prefix": 4, "private_ip": 5}
    src1, d1, t1 = vp1
    src2, d2, t2 = vp2

    if techniques_order[t1] < techniques_order[t2]:
        return -1
    elif techniques_order[t1] == techniques_order[t2]:
        if d1 < d2:
            return -1
        elif d1 > d2:
            return 1
        elif d1 == d2:
            return 0
    else:
        return 1

def precompute_ranking_impl(site_per_vp, d_training_per_dst_prefix, d_evaluation_per_dst_prefix,
                            host, password, database, ingress_table):

    # import pyasn
    # ipasnfile = '/home/kevin/rankingservice/resources/bgpdumps/latest'
    # ip2asn = pyasn.pyasn(ipasnfile)
    rows = []
    for i, (dst_bgp_prefix, dst_src_rr_paths) in enumerate(d_training_per_dst_prefix):
        if i % 10000 == 0:
            print(f"{i} BGP prefixes done on {len(d_training_per_dst_prefix)}")
        dst_src_rr_paths_training = dst_src_rr_paths[0]
        dst_src_rr_paths_evaluation = d_evaluation_per_dst_prefix[dst_bgp_prefix][0]
        # src_dst_rr_paths_training = training_set_multi_per_bgp_prefix[dst_bgp_prefix]
        ingresses, ingress_per_vp, sorted_vps_by_distance_per_ingress, _ = compute_ingresses(
            dst_src_rr_paths_training,
            dst_src_rr_paths_evaluation,
            ip2asn
        )
        ranked_vps = rank(ingresses, sorted_vps_by_distance_per_ingress)
        rows.extend(vps_to_rows(site_per_vp, dst_bgp_prefix, ranked_vps))

    client = Client(host, password=password)

    query = (
        f"INSERT INTO {database}.{ingress_table} (dst_bgp_prefix_start, dst_bgp_prefix_end, "  # Network metadata of dst
        f"site, src, ingress, distance, rank) VALUES "
    )
    client.execute(query, rows)

    return rows

def vps_to_rows(site_per_vp, dst_bgp_prefix, ranked_vps):
    rows = []
    for i, (src, ingress, distance) in enumerate(ranked_vps):
        if src in site_per_vp:
            rows.append((dst_bgp_prefix[0], dst_bgp_prefix[1], site_per_vp[src], src, ingress, distance, i))
    return rows

def rank(
        sorted_ingresses,
        sorted_vps_by_distance_per_ingress,
        responsive_vps = None
        ):

    if len(sorted_ingresses) == 0:
        return []

    ranked_vps = []
    ranked_vps_s = set()

    # Compute the maximum number of VPs per ingress
    max_vps_per_ingress = max(len(sorted_vps_by_distance_per_ingress[x]) for x in sorted_ingresses)
    for i in range(0, max_vps_per_ingress):
        for ingress in sorted_ingresses:
            vps = sorted_vps_by_distance_per_ingress[ingress]
            if len(vps) > i:
                vp = vps[i]
                if vp not in ranked_vps_s:
                    if responsive_vps is not None:
                        if vp[0] not in responsive_vps:
                            continue
                    ranked_vps.append((vp[0], ingress, vp[1]))
                    ranked_vps_s.add(vp[0])
    return ranked_vps


def rank_convergence_points(
        rr_sub_paths_groups_per_recurse,
        responsive_vps = None
        ):
    '''
    Ranking is done as follows: for each level of recursion, take
    :param training_vps:
    :return:
    '''
    ranked_vps = []
    ranked_vps_s = set()
    for recurse_level, rr_path_per_vp_per_convergence_point in sorted(rr_sub_paths_groups_per_recurse.items(), key=lambda x:x[0]):

        add_to_ranking = []
        for convergence_point, rr_path_per_vp in rr_path_per_vp_per_convergence_point.items():
            # Sort VPs by their and distance to the convergence_point.
            rr_path_per_vp_sorted = sorted(rr_path_per_vp, key=lambda x: x[1])
            for vp, distance, technique in rr_path_per_vp_sorted:
                if vp not in ranked_vps_s:
                    if responsive_vps is not None and vp in responsive_vps:
                        add_to_ranking.append((vp, distance))
                        ranked_vps_s.add(vp)
                        break
                    else:
                        add_to_ranking.append((vp, distance))
                        ranked_vps_s.add(vp)
                        break


        # Sort vps by their min distance to their convergence point
        add_to_ranking_sorted = sorted(add_to_ranking, key=lambda x: x[1])
        ranked_vps.extend([x[0] for x in add_to_ranking_sorted])

    return ranked_vps


class IngressCoverRRVPSelector(RRVPSelector):

    def __init__(self, db_host, db_password, database, table, ipasnfile, is_in_memory):
        super(IngressCoverRRVPSelector, self).__init__(db_host, db_password, database, ipasnfile)
        self.ingress_table = table
        self.in_memory_ranking = {}
        self.is_in_memory = is_in_memory
        if self.is_in_memory:
            self.load_ranking_in_memory()


    @classmethod
    def get_rr_ip_hops_query(cls, db, table, filter_bgp_prefixes):

        query = (
            f" WITH groupUniqArray((src, rr_ip, rr_hop)) as rr_ip_hops, "
            f" arrayDistinct(arrayMap(x->x.1, rr_ip_hops)) as srcs, "
            f" arrayMap(x->(x, arrayFilter(y->y.1=x, rr_ip_hops)), srcs) as rr_ip_hops_by_src, "
            f" arrayMap(x->(x.1, arrayMap(y->(y.2, y.3), x.2)), rr_ip_hops_by_src) as rr_ip_hops_by_dst_no_src, "
            f" arrayMap(x->(x.1, arraySort(y->y.2, x.2)), rr_ip_hops_by_dst_no_src) as sorted_rr_ip_hops_by_dst_no_src "
            # f" arrayFilter(x->x.1 == dst, rr_ip_hops) as rr_ip_dst, "
            # f" arrayReduce('max', arrayMap(x->x.2, rr_ip_dst)) as max_dst_rr_hop "
            f" SELECT dst_bgp_prefix_start, dst_bgp_prefix_end, dst, sorted_rr_ip_hops_by_dst_no_src "
            f" FROM {db}.{table}"
        )
        if len(filter_bgp_prefixes) > 0:
            query += f" WHERE (dst_bgp_prefix_start, dst_bgp_prefix_end) in {filter_bgp_prefixes} "
            # f" WHERE ping_id < 1969691651 + 200000"
        # query += f" WHERE dst_bgp_prefix_end < 106777471 "
        query += (
            f" GROUP BY dst_bgp_prefix_start, dst_bgp_prefix_end, dst "
            f" HAVING length(rr_ip_hops) > 0 "
            # f" ORDER BY dst_bgp_prefix_start, dst_bgp_prefix_end "
            # f" LIMIT 1 BY (dst_bgp_prefix_start, dst_bgp_prefix_end, src, dst)"
            # f" LIMIT 2 BY (dst_bgp_prefix_start, dst_bgp_prefix_end, dst)"
            # f" LIMIT 500000 "
            # f" HAVING max_dst_rr_hop < 8 AND "

        )
        return query

    def precompute_ranking(self, rr_pings_table, ingress_table, is_insert):
        print("Precomputing ingresses, getting rr paths")
        # (16777216, 16777471) for debug
        filter_bgp_prefix = []
        # filter_bgp_prefix = [(16777216, 16777471)]
        query = self.get_rr_ip_hops_query(self.database, rr_pings_table, filter_bgp_prefixes=filter_bgp_prefix)
        print(query)
        client = Client(self.db_host, password=self.db_password)
        start = time.time()
        settings = {'max_block_size': 100000, 'max_query_size': 1000000000, "max_ast_elements": 500000}
        rows = client.execute_iter(query, settings=settings)
        d_training_per_dst_prefix = {}
        d_training = {}

        d_evaluation_per_dst_prefix = {}
        d_evaluation = {}

        dst_src_rr_ip_hops = {}


        i = 0
        for row in rows:
            i += 1
            if i % 10000 == 0:
                print(i)
                # break
            dst_bgp_prefix_start, dst_bgp_prefix_end, dst, src_rr_ip_hops = row
            dst_bgp_prefix = dst_bgp_prefix_start, dst_bgp_prefix_end
            dst_src_rr_ip_hops.setdefault(dst_bgp_prefix, {})[dst] = src_rr_ip_hops

        i = 0
        for dst_bgp_prefix, dst_src_rr_ip_hops in dst_src_rr_ip_hops.items():
            i += 1
            if i % 10000 == 0:
                print(i, "prefixes split training and evaluation.")
            if len(dst_src_rr_ip_hops) < 2:
                # logger.info(f"Skipping BGP prefix {dst_bgp_prefix}, only {len(dst_src_rr_ip_hops)} destination")
                dst, src_rr_ip_hops = list(dst_src_rr_ip_hops.items())[0]
                d_training_per_dst_prefix.setdefault(dst_bgp_prefix, []).append((dst, src_rr_ip_hops))
                d_evaluation_per_dst_prefix.setdefault(dst_bgp_prefix, []).append((dst, src_rr_ip_hops))
                continue

            for k, (dst, src_rr_ip_hops) in enumerate(dst_src_rr_ip_hops.items()):
                if k >= 2:
                    break
                if k < 2:
                    if k == 0:
                        d_training_per_dst_prefix.setdefault(dst_bgp_prefix, []).append((dst, src_rr_ip_hops))
                    elif k == 1:
                        d_evaluation_per_dst_prefix.setdefault(dst_bgp_prefix, []).append((dst, src_rr_ip_hops))


        elapsed = time.time() - start
        print(f"Query took {elapsed} seconds")
        rows = []

        print("Inserting into ingress table...")
        dropTable(self.db_host, self.db_password, self.database, ingress_table)
        createIngressCoverTable(self.db_host, self.db_password, self.database, ingress_table)

        # TODO Multiprocess this code
        p_arg_size = 10000
        args = [list(d_training_per_dst_prefix.items())[i * p_arg_size: (i + 1) * p_arg_size]
                for i in range(0, int(len(d_training_per_dst_prefix) / p_arg_size) + 1)]
        print(f"{len(d_training_per_dst_prefix)} BGP prefixes to compute.")
        # Create args for multiprocessing
        args_mp_process = []
        for i, d_training_per_dst_prefix_process in enumerate(args):
            print("Creating args for multiprocess...")
            d_evaluation_per_dst_prefix_process = {dst_prefix[0]: d_evaluation_per_dst_prefix[dst_prefix[0]]
                                                   for dst_prefix in d_training_per_dst_prefix_process}
            args_mp_process.append((self.site_per_vp, d_training_per_dst_prefix_process, d_evaluation_per_dst_prefix_process,
                                    self.db_host, self.db_password, self.database, ingress_table))

        with Pool() as p:
            print("Starting to compute ingresses...")
            p.starmap(precompute_ranking_impl, args_mp_process)
            # for row_process in row_processes:
            #     rows.extend(row_process)


    def load_ranking_in_memory(self):
        print("Starting loading ranking in memory for ingress cover...")
        query = (
            f" WITH groupUniqArray((site, ingress, distance, rank)) as site_rank, "
            f" arraySort(x->x.4, site_rank) as site_rank_sorted "
            f" SELECT dst_bgp_prefix_start, dst_bgp_prefix_end, site_rank_sorted FROM {self.database}.{self.ingress_table} "
            f" GROUP BY dst_bgp_prefix_start, dst_bgp_prefix_end "
        )
        print(query)
        client = Client(self.db_host, password=self.db_password)
        rows = client.execute(query)

        for row in rows:
            ranked_vps = []
            dst_bgp_prefix_start, dst_bgp_prefix_end, src_ingress_distance_rank_s = row
            for site, ingress, distance, rank in src_ingress_distance_rank_s:
                ranked_vps.append((site, ingress, distance))
            self.in_memory_ranking[(dst_bgp_prefix_start, dst_bgp_prefix_end)] = ranked_vps
        print("Finished loading ranking in memory for ingress cover...")

    def getRRVPs(self, destination, granularity, n_vp):
        '''
           Return the vantage points that have min distance from the different ingresses.
           :param destination:
           :param granularity:
           :param n_vp:
           :return:
        '''
        if type(destination) == int:
            # Change the representation of the destination to string
            destination = ipItoStr(destination)

        dentry_dst = dnets_of(destination, self.ip2asn)
        bgp_dst_start, bgp_dst_end = bgpDBFormat(dentry_dst)

        if not self.is_in_memory:
            query = (
                f" SELECT site, rank, ingress, distance FROM {self.database}.{self.ingress_table} "
                f" WHERE  dst_bgp_prefix_start={bgp_dst_start} AND dst_bgp_prefix_start= {bgp_dst_end} "
                f" ORDER BY rank "
            )

            client = Client(self.db_host, password=self.db_password)

            rows = client.execute(query)
            ranked_sites = []
            # ranked_sites_s = set()
            for row in rows:
                site, rank, ingress, distance = row
                ranked_sites.append((site, ingress, distance))
                # ranked_sites_s.add((site, ingress))
            ranked_sites = self.add_global_ranking(ranked_sites)
            ranked_sites = ranked_sites[:n_vp]
            ranked_vps = self.sites_to_vps(ranked_sites)
            return ranked_vps
        else:
            ranked_sites = []
            if (bgp_dst_start, bgp_dst_end) in self.in_memory_ranking:
                ranked_sites = self.in_memory_ranking[(bgp_dst_start, bgp_dst_end)]
            ranked_sites = self.add_global_ranking(ranked_sites)
            ranked_sites = ranked_sites[:n_vp]
            ranked_vps   = self.sites_to_vps(ranked_sites)

            return ranked_vps



def insert_ingress_computation(rr_pings_table, ingress_table):
    host = db_host
    database = "revtr"
    # rr_pings_table = "RRPings"
    # rr_pings_table = "RRPingsEvaluationBGPAll"
    # ingress_table = "RRPingsIngress"
    ipasnfile = 'resources/bgpdumps/latest'
    global_ranking_file = "resources/global_cover_vps"
    selector = IngressCoverRRVPSelector(host, clickhouse_pwd, database, ingress_table,
                                         ipasnfile, is_in_memory=False)
    selector.precompute_ranking(rr_pings_table, ingress_table, is_insert=True)

def test():
    host = db_host
    database = "revtr"
    ingress_table = "RRPingsIngress"
    ipasnfile = 'resources/bgpdumps/latest'
    global_ranking_file = "resources/global_cover_vps"
    selector = IngressCoverRRVPSelector(host, clickhouse_pwd, database, ingress_table, ipasnfile, is_in_memory=True)
    vps = selector.getRRVPs("36.68.72.1", granularity="BGP", n_vp=100)
    print(vps)
    vps = selector.getRRVPs("36.68.72.1", granularity="BGP", n_vp=100)
    print(vps)


if __name__ == "__main__":
    rr_pings_table = "ranking_evaluation_2021_10_28"
    ingress_table = "ranking_2021_10_28"
    insert_ingress_computation(rr_pings_table, ingress_table)
    # test()
