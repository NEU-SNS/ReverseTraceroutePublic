import ipaddress
from datetime import date, datetime
from mysql.connector import connect
from algorithms.utils import extract_midar_routers
from algorithms.database import get_connection

def get_select_alias_query(ips):
    query = (
        f" SELECT cluster_id, ip_address from "
        f" ip_aliases "
        f" WHERE cluster_id in ("
        f" SELECT cluster_id from ip_aliases  WHERE ip_address in {ips} "
        f")  "
    )
    return query

def get_insert_query(id, ips):
    query = "INSERT INTO ip_aliases(cluster_id, ip_address) VALUES "
    for i, ip in enumerate(ips):
        if i > 0:
            query += ","
        query += f"({id}, {ip})"

    return query
def load_midar_file_into_database(midar_file):



    alias_sets = extract_midar_routers(midar_file)


    connection = get_connection("traceroute_atlas")

    '''
    For each alias set, look if there exists an alias set containing one ip of the alias set
    if it is the case, insert with the cluster id, otherwise
    insert a new alias set.
    '''
    updated_alias_sets = set()
    new_alias_sets = set()
    cursor = connection.cursor()
    for alias_set in alias_sets:
        alias_set_i = sorted([int(ipaddress.ip_address(ip)) for ip in alias_set])
        print(alias_set_i)
        alias_set_sql = tuple(alias_set_i)
        query = get_select_alias_query(alias_set_sql)
        cursor.execute(query, )
        rows = cursor.fetchall()
        # print(f"Fetched {len(rows)} rows")
        i = 0
        cluster_ids_to_merge = {}
        for row in rows:
            i += 1
            cluster_id, ip = row
            cluster_ids_to_merge.setdefault(cluster_id, set()).add(ip)

        if len(rows) == 0:
            # Insert a new cluster ID
            cursor.execute("SELECT max(cluster_id) from ip_aliases")
            max_id = cursor.fetchone()[0]
            query = get_insert_query(max_id + 1, alias_set_i)
            cursor.execute(query)
            new_alias_sets.add(max_id + 1)

        else:
            # Insert in the corresponding cluster id, and merge if necessary
            ips_in_cluster = set.union(*[c for c in cluster_ids_to_merge.values()])
            ips_to_insert = set(alias_set_i) - ips_in_cluster
            if len(ips_to_insert) == 0:
                # IP addresses already in the db
                # Case where 2 clusters: continue, do not update the current clusters
                continue

            new_cluster_id = min(cluster_ids_to_merge)
            updated_alias_sets.add(new_cluster_id)
            # Insert new IPs
            query = get_insert_query(new_cluster_id, ips_to_insert)
            cursor.execute(query)

            if len(cluster_ids_to_merge) > 1:
                # Update existing ones
                for cluster_id, ips in cluster_ids_to_merge.items():
                    if cluster_id == new_cluster_id:
                        continue
                    else:
                        query = f"UPDATE ip_aliases SET cluster_id = {new_cluster_id} where cluster_id = {cluster_id} "
                        cursor.execute(query)
    # connection.commit()
    cursor.close()
    connection.close()

    print(f"Inserted {len(new_alias_sets)} new alias sets.")
    print(f"Updated {len(updated_alias_sets)} existing alias sets.")

def get_unstable_vps():
    '''
    return VPs that have been quarantined in the last 2 months.
    :return:
    '''

    connection = get_connection("vpservice")
    cursor = connection.cursor()
    query = (f" select vp.ip, vpe.hostname, vpe.type, vpe.time as c "
             f" from vp_events vpe "
             f" INNER JOIN vantage_points vp ON vp.hostname=vpe.hostname "
             f" where vpe.site like '%MLab%' and vpe.hostname like 'revtr%' and time >= DATE_SUB(NOW(), interval 90 day)"
             # f" GROUP BY ip"
             f" ORDER BY vpe.time "
             f"")

    cursor.execute(query)
    rows = cursor.fetchall()
    # How much time the vp have been offline in the last 3 months?
    last_time_online_by_vp = {}
    status_by_vp = {}
    offline_by_ip = {}
    for ip, hostname, type, time in rows:
        if ip not in status_by_vp:
            status_by_vp[ip] = "ONLINE"
        if type == "OFFLINE":
            # Start offline count
            status_by_vp[ip] = type
            last_time_online_by_vp[ip] = time
        else:
            if status_by_vp[ip] == "OFFLINE":
                status_by_vp[ip] = "ONLINE"
                # VP is back, so count the number of time it was offline
                offline_by_ip.setdefault(ip, 0)
                # Time the vp stayed offline
                offline_by_ip[ip] += (time - last_time_online_by_vp[ip]).seconds


    cursor.close()
    connection.close()

    return offline_by_ip

def get_stable_vps(num_vp):

    # print(query)
    query = (
        f" SELECT ip, site, rec_spoof FROM vantage_points "
    )
    # Get the connection to
    connection = get_connection("vpservice")
    cursor = connection.cursor()
    cursor.execute(query, )
    rows = cursor.fetchall()
    sites_by_ip = {}
    for row in rows:
        ip, site, can_record_route = row
        if can_record_route:
            sites_by_ip[ip] = site

    cursor.close()
    connection.close()

    offline_by_ip = get_unstable_vps()
    stable_vps = sorted(offline_by_ip.items(), key=lambda x:x[1])
    stable_vps_with_site = [(x[0], x[1], sites_by_ip[x[0]]) for x in stable_vps if x[0] in sites_by_ip]
    selected_cities = set()
    # Only select one per site
    selected_vps = []
    for ip, offline, site in sorted(stable_vps_with_site, key=lambda x:x[2]):
        tokens = site.split("MLab - ")
        site = tokens[1]
        city = site
        if city in selected_cities:
            continue
        selected_vps.append(ip)
        selected_cities.add(city)
        if len(selected_vps) >= num_vp:
            break
    return selected_vps

def get_alias_sets(ips, with_single = False):
    '''
    Return all the alias sets containing an IP address in a traceroute to the src
    :param ips: tuple of ip int
    :return:
    '''

    connection = get_connection("traceroute_atlas")
    cursor = connection.cursor()

    # Divide the ips in batch to have a not too long query
    batch = 10000
    alias_set_by_ip = {}
    for i in range(0, len(ips), batch):
        query = get_select_alias_query(ips[i:i+batch])
        # print(query)
        cursor.execute(query)
        rows = cursor.fetchall()
        for row in rows:
            cluster_id, ip = row
            alias_set_by_ip[ip] = cluster_id

    return alias_set_by_ip

if __name__ == "__main__":
    midar_file = "atlas/resources/midar/midar-noconflicts.sets"
    load_midar_file_into_database(midar_file)

