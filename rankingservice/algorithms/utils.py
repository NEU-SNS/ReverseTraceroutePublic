import os
import re
import json
import ipaddress

def generalized_jaccard(S):
    '''
    S is an iterable of sets
    :param S:
    :return:
    '''
    union = set.union(*S)
    intersection = set.intersection(*S)
    return len(intersection) / len(union)


def find(l, t):
    for i, e in enumerate(l):
        if e == t:
            return (i, e)
    return None

def adjust_size(ds):
    for i in range(len(ds)):
        d_i = ds[i]
        for j in range(len(ds)):
            if i == j:
                continue
            d_j = ds[j]
            for k in d_i:
                if k not in d_j:
                    d_j[k] = 0

def connected(list_of_lists):
    # Apply transitive closure
    old_len_connected = len(list_of_lists)
    new_len = 0
    # End condition
    while old_len_connected != new_len:
        old_len_connected = new_len
        connected_lists = []
        # Now apply transitive closure
        merged = set()
        for i in range(0, len(list_of_lists)):
            if list_of_lists[i] in merged:
                continue
            for j in range(i + 1, len(list_of_lists)):
                if len(list_of_lists[i].intersection(list_of_lists[j])) > 0:
                    list_of_lists[i].update(list_of_lists[j])
                    if frozenset(list_of_lists[j]) not in merged:
                        merged.add(frozenset(list_of_lists[j]))
            connected_lists.append(list_of_lists[i])

        list_of_lists = connected_lists
        new_len = len(connected_lists)
    final_sets = connected_lists
    return final_sets

def any_intersection(ts, ss):
    '''
    This function return the intersection between
    :param s: s is a set
    :param ss: ss is a list of non overlapping sets
    :return:
    '''

    for s in ss:
        inter_ts_s  = ts.intersection(s)
        if len(inter_ts_s) > 0:
            return s, inter_ts_s
    return None, set()

def extract_midar_routers(midar_file):
    """Parsing midar-noconflicts.sets file(s)

    Returns a list of sets
    Each set corresponds to the IPs of one router
    """
    list_routers = []
    router = set()
    with open(midar_file) as f:
        for line in f:
            if re.match("# end", line) is None:
                if re.match("# set", line):
                    if len(router) > 0:
                        list_routers.append(router)
                    router = set()
                if re.match("^[0-9]", line):
                    router.add(line.strip())
            else:
                list_routers.append(router)
    return list_routers

def extract_apple_routers(apple_file):
    with open(apple_file) as f:
        routers = json.load(f)
        for i, r in enumerate(routers):
            routers[i] = set(str(ipaddress.ip_address((ip))) for ip in r)
    return routers