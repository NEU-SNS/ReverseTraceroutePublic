from mysql.connector import connect
from atlas.atlas_database import get_connection, get_stable_vps

import ipaddress

def extract_tr_rr_hops_from_db():
    connection = get_connection("traceroute_atlas")

    query = (
        f" SELECT rr_hop "
        f" FROM atlas_rr_pings arrp "
        # f" INNER JOIN atlas_traceroute_hop ath on ath.trace_id=arrp.traceroute_id "
        # f" WHERE traceroute_id={tr_id} "
        f" UNION ALL "
        f" SELECT hop "
        f" FROM atlas_traceroute_hops "
        # f" WHERE trace_id={tr_id}"
    )
    cursor = connection.cursor()
    cursor.execute(query, )
    # rows = cursor.fetchmany(batch_size)
    rows = cursor.fetchall()
    ips = set()
    for i, row in rows:
        ip = row[0]
        ip = ipaddress.ip_address(ip)
        if ip.is_private:
            continue
        ips.add(str(ip))
    ofile = "atlas/resources/midar.targets"
    with open(ofile, "w") as f:
        for ip in ips:
            f.write(ip +  "\n")


def available_sources():
    connection = get_connection("vpservice")
    query = (
        "SELECT ip, rec_spoof FROM vantage_points "
    )

    cursor = connection.cursor()
    cursor.execute(query)
    rows = cursor.fetchall()

    vps = set()
    for ip, can_rec_spoof in rows:
        if can_rec_spoof:
            vps.add(ip)
    print(len(vps))

    vps_used  = set()
    with open("algorithms/evaluation/resources/sources.csv") as f:
        for line in f:
            vps_used.add(int(ipaddress.ip_address(line.strip())))

    print(len(vps_used))
    print(len(vps_used.intersection(vps)))


def dump_stable_vps():

    stable_vps = get_stable_vps(1000)
    with open("algorithms/evaluation/resources/sources.csv", "w") as f:
        for vp in stable_vps:
            vp = str(ipaddress.ip_address(vp))
            f.write(vp + "\n")

if __name__ == "__main__":
    dump_stable_vps()
    # available_sources()
