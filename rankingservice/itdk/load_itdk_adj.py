import bz2
import re
import ipaddress
import json
from network_utils import prefix_24_from_ip
from clickhouse_driver import Client
from atlas.atlas_database import get_connection
ipv4_regex = re.compile("(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)")




def string_to_link(s):
    src, dst = s.split("_")
    return (src, dst)

def get_ts_responsive_nodes():
    host = "localhost"
    database = "revtr"
    table_hitlist = "ts_survey_hitlist"
    table_ark = "ts_survey_ark_routers"
    client = Client(host)
    query = (
        f" SELECT distinct(dst) from {database}.{table_hitlist} "
        f" UNION ALL "
        f" SELECT distinct(dst) from {database}.{table_ark} "
    )


    rows = client.execute(query)
    responsive_nodes = set()
    for row in rows:
        responsive_nodes.add(str(ipaddress.ip_address(row[0])))

    return responsive_nodes

def parse_links(itdk_nodes_file, itdk_links_file, responsive_nodes):
    start_nodes_index = 3

    ips_per_node_id = {}
    i = 0
    with open(itdk_nodes_file) as f:
        for line in f:
            # line = line.decode("utf-8")
            i += 1
            if i % 100000 == 0:
                print(i)
                # break
            if line.startswith("#"):
                continue
            line = line.strip("\n")

            s = line.split(" ")
            router_id = s[1].split(":")[0]
            router = s[start_nodes_index:-1]
            if len(set(router).intersection(responsive_nodes)) > 0:
                ips_per_node_id[router_id] = router
            else:
                ips_per_node_id[router_id] = []
            # ips_per_node_id[router_id] = set(router).intersection(responsive_nodes)
            # for ip in router:
            #     ips_per_node_id[ip] = router_id

    i = 0
    links = set()
    links_line = re.compile("(N[0-9]+)+")
    with open(itdk_links_file) as f:
        for line in f:
            # line = line.decode("utf-8")
            i += 1
            if i % 100000 == 0:
                print(i)
            if line.startswith("#"):
                continue
            line = line.strip("\n")
            nodes_id = re.findall(links_line, line)
            nodes = list(set().union(*[ips_per_node_id[node_id] for node_id in nodes_id]))

            for i in range(0, len(nodes)):
                for j in range(i+1, len(nodes)):
                    links.add((nodes[i], nodes[j]))

    print(len(links))
    link_file = "resources/data.caida.org/itdk.links"
    with open(link_file, "w") as f:
        for link in links:
            f.write(f"{link[0]},{link[1]}\n")



def get_links(links_file):

    links = set()
    with open(links_file) as f:
        for line in f:
            line = line.strip("\n")
            src, dst = line.split(",")

            links.add((src, dst))

    return links

def get_links_from_json(links_file):
    with open(links_file) as f:
        links_per_dst_prefix = json.load(f)
        print(len(links_per_dst_prefix))
        links_per_dst_prefix = {p :
                                    { string_to_link(l) : links_per_dst_prefix[p][l]
                                     for l in links_per_dst_prefix[p]
                                     }
                                for p in links_per_dst_prefix
                                }
        print(len(links_per_dst_prefix))

        return links_per_dst_prefix

def get_links_from_public_ripe(ripe_public_file):


def load_adjacencies(links):
    connection = get_connection("revtr", autocommit=0)
    query = (
        f" INSERT INTO adjacencies(ip1, ip2) VALUES "
    )
    cursor = connection.cursor()
    for i, (ip1, ip2) in enumerate(list(links)):
        if ip1 == ip2:
            continue
        ip1 = int(ipaddress.ip_address(ip1))
        ip2 = int(ipaddress.ip_address(ip2))
        if i > 0:
            query += ","
        query += f"({ip1}, {ip2})"
    cursor.execute(query)
    connection.commit()
    cursor.close()
    connection.close()

def load_adjacencies_destinations(links_by_dst_prefix, is_only_source_prefixes):

    prefix_filter = set()
    if is_only_source_prefixes:
        connection_vp = get_connection("vpservice")
        query = (
            f" SELECT ip FROM vantage_points "
        )
        cursor = connection_vp.cursor()
        cursor.execute(query)
        rows = cursor.fetchall()
        for row in rows:
            prefix_24 = prefix_24_from_ip(row[0])
            prefix_filter.add(prefix_24)




    connection = get_connection("revtr", autocommit=0)
    query = (
        f" INSERT INTO adjacencies_to_dest(dest24, address, adjacent, cnt) VALUES "
    )
    cursor = connection.cursor()

    batch_query = query
    insert_index = 0
    for k, (dst_24, links) in enumerate(links_by_dst_prefix.items()):
        if is_only_source_prefixes:
            if dst_24 not in prefix_filter:
                continue
        for i, (link, count) in enumerate(links.items()):
            ip1, ip2 = link
            if ip1 == ip2:
                continue
            ip1 = int(ipaddress.ip_address(ip1))
            ip2 = int(ipaddress.ip_address(ip2))
            # Get the /24 of IP2
            if insert_index > 0:
                batch_query += ","
            batch_query += f"({dst_24}, {ip1}, {ip2}, {count})"
            insert_index += 1
        # if k % 10000 == 0:
        # print(k)
    cursor.execute(batch_query)
    connection.commit()
    batch_query = query
    insert_index = 0

    cursor.close()
    connection.close()



if __name__ ==  "__main__":

    # responsive_nodes = get_ts_responsive_nodes()
    links_file = "algorithms/evaluation/resources/ark.links"
    links_destination_file = "algorithms/evaluation/resources/ark_links_destination_to_sources.json"
    # links = get_links(links_file)
    # load_adjacencies(links)
    links_destination = get_links_from_json(links_destination_file)

    load_adjacencies_destinations(links_destination, is_only_source_prefixes=False)

