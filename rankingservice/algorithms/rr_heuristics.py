import ipaddress
from network_utils import dnets_of, bgpDBFormat

def compute_double_stamp(rr_ip_hops):
    for i, (rr_ip, rr_hop) in enumerate(rr_ip_hops):
        if i == 0:
            continue
        # Find two consecutive same IP addresses
        if rr_ip == rr_ip_hops[i-1][0] and rr_hop < 8:
            # if rr_ip != dst:
            #     return rr_ip_hops[i]
            # else:
            if i == 1:
                return i - 1, rr_ip_hops[i-1]
            else:
                return i - 2, rr_ip_hops[i-2]

def compute_ip_loop(rr_ip_hops):
    count_by_ip = {}
    for i, (rr_ip, rr_hop) in enumerate(rr_ip_hops):
        if rr_ip not in count_by_ip:
            count_by_ip[rr_ip] = i
        else:
            if count_by_ip[rr_ip] == i - 1:
                # This is not a loop, just a double stamp
                continue
            # Be conservative on the index loop, take the first IP
            # so that there is no loop in the subpath
            return i, rr_ip_hops[i]
    return None

def compute_enter_exit_prefix(rr_ip_hops, dst, ip2asn):
    '''
    If we go into the BGP prefix and leave it, we probably reached the destination
    :param rr_ip_hops:
    :param dst:
    :return:
    '''
    # bgp_prefix = prefix_24_from_ip(dst)
    dentry_dst = dnets_of(dst, ip2asn, ip_representation="uint32")
    bgp_dst_start, bgp_dst_end = bgpDBFormat(dentry_dst)
    in_bgp_prefix = None
    for i, (rr_ip, rr_hop) in enumerate(rr_ip_hops):
        if bgp_dst_start <= rr_ip <= bgp_dst_end:
            in_bgp_prefix = i, rr_ip_hops[i]
        else:
            if in_bgp_prefix is not None:
                # We left the destination /24, so it's probably reverse hop
                return in_bgp_prefix
def compute_private_ip_address(rr_ip_hops):
    '''
    Find the first IP address before the private IP address
    :param rr_ip_hops:
    :return:
    '''
    in_private = None
    for i, (rr_ip, rr_hop) in enumerate(rr_ip_hops):
        if ipaddress.ip_address(rr_ip).is_private:
            if in_private is None:
                # We are still in the first private hops
                continue
            else:
                # We are entering again in a private network, probably close to the destination
                return i, rr_ip_hops[i-1]
        else:
            if in_private is None:
                # Leaving private network
                in_private = False
    return None


def compute_heuristics(src_dst_rr_path, ip2asn):
    src, dst, rr_hop_path = src_dst_rr_path
    # Try double stamp technique
    rr_path = [x[0] for x in rr_hop_path]
    if dst in rr_path:
        # Destination was reached, no need for heuristics
        return None
    index_rr_dst = compute_double_stamp(rr_hop_path)
    if index_rr_dst is not None and rr_path[-1] != dst:
        index, _ = index_rr_dst
        # Change the destination
        new_dst = rr_hop_path[index][0]
        distance = index
        return (src, new_dst, rr_hop_path, distance, "double_stamp")
    # Try the ip loop technique
    index_rr_dst = compute_ip_loop(rr_hop_path)
    if index_rr_dst is not None and rr_path[-1] != dst:
        index, _ = index_rr_dst
        new_dst = rr_hop_path[index][0]
        distance = index
        return (src, new_dst, rr_hop_path, distance, "ip_loop")
    # Try the enter_exit_prefix technique
    index_rr_dst = compute_enter_exit_prefix(rr_hop_path, dst, ip2asn)
    if index_rr_dst is not None and rr_path[-1] != dst:
        index, _ = index_rr_dst
        new_dst = rr_hop_path[index][0]
        distance = index
        return (src, new_dst, rr_hop_path, distance, "enter_exit_prefix")
    # Try the private IP address technique
    # index_rr_dst = compute_private_ip_address(rr_hop_path)
    # if index_rr_dst is not None and rr_path[-1] != dst:
    #     index, _ = index_rr_dst
    #     if index == 0:
    #         rr_sub_paths.append((src, [(None, 0)], index, "private_ip"))
    #     else:
    #         rr_sub_paths.append((src, rr_hop_path[:index], index, "private_ip"))
    #     continue
    return None