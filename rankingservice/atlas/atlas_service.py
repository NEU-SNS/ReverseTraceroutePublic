import argparse
import random
import requests
import time
import json
from datetime import datetime, date
from atlas.ripe import load_probes_from_file, fetch_traceroute_results_from_our_measurements
from atlas.atlas_database import get_connection


class Options(object):
    def __init__(self, ripe_account, ripe_key):
        self.ripe_account = ripe_account
        self.ripe_key = ripe_key

class TraceOptions(Options):

    def __init__(self, ripe_account, ripe_key, uuid, n_traceroutes_per_source, is_dry_run):
        super().__init__(ripe_account, ripe_key)
        self.uuid = uuid
        self.n_traceroutes_per_source = n_traceroutes_per_source
        self.is_dry_run = is_dry_run

class RefreshOptions(Options):
    def __init__(self, ripe_account, ripe_key, uuid, is_dry_run, description):
        super().__init__(ripe_account, ripe_key)
        self.uuid = uuid
        self.is_dry_run = is_dry_run
        self.description = description

class FetchOptions(Options):
    def __init__(self, ripe_account, ripe_key, uuid, dump_file):
        super().__init__(ripe_account, ripe_key)
        self.uuid = uuid
        self.dump_file = dump_file

def fetch_traceroutes(options):
    traceroutes = fetch_traceroute_results_from_our_measurements(options.uuid,
                                                                 int(datetime.combine(date.today(),
                                                                                      datetime.min.time()).timestamp()),
                                                                 options.ripe_key,
                                                                 measurement_limit=10000,
                                                                 is_most_recent=False)

    # Export traceroutes to json
    with open(options.dump_file, "w") as f:
        json.dump(traceroutes, f)

def traceroutes_to_sources(probes_per_source, options):
    measurement_ids = []
    max_probes_per_measurement = 1000
    for source, probe_ids in probes_per_source.items():
        for j in range(0, len(probe_ids), max_probes_per_measurement):
            while True:
                ids_str = f"".join([f",{i}" for i in probe_ids[j:j+max_probes_per_measurement]])[1:]
                if options.is_dry_run:
                    print(f"Dry run {source}, probes {probe_ids[j]} {probe_ids[j+max_probes_per_measurement-1]}")
                    break
                # description = f"Tracerouter atlas refresh to {source}, uuid {options.uuid}"
                # if options.is_evaluation:
                #     description += " evaluation"
                response = requests.post(
                    f"https://atlas.ripe.net/api/v2/measurements/?key={options.ripe_key}",
                    json={
                        "definitions": [
                            {
                                "target": source,
                                "af": 4,
                                "packets": 1,
                                "proto": "ICMP",
                                # "size": 48,
                                "description": options.description,
                                "skip_dns_check": True,
                                "type": "traceroute",
                            }
                        ],
                        "probes": [
                            {"value": ids_str, "type": "probes", "requested": len(ids_str)}
                        ],
                        "is_oneoff": True,
                        "bill_to": f"{options.ripe_account}",
                    },
                ).json()
                print(response)
                if "error" in response:
                    time.sleep(30)
                else:
                    measurement_ids.extend(response["measurements"])
                    break


def random_traceroutes_to_sources(sources, probes, anchors, options):

    max_probes_per_measurement = 1000
    # Limit of number of measurement is 1000
    anchor_ids = [p["id"] for p in anchors]
    probes_ids = [p["id"] for p in probes]

    if options.n_traceroutes_per_source < len(anchor_ids):
        random_probe_ids = random.sample(anchor_ids, options.n_traceroutes_per_source)
    else:
        # Select anchors and complete with standard probes
        random_probe_ids = random.sample(probes_ids, options.n_traceroutes_per_source - len(anchor_ids))
        random_probe_ids += anchor_ids
    for j in range(0, options.n_traceroutes_per_source, max_probes_per_measurement): # 1000 is the maximum number of probes allowed per measurement
        for i, source in enumerate(sources):
            # Select random anchors
            # print(f"{i} batch of probes, {j} batch of sources")
            while True:
                ids_str = f"".join([f",{i}" for i in random_probe_ids[j:j+max_probes_per_measurement]])[1:]
                if options.is_dry_run:
                    print(f"Dry run {source}, probes {random_probe_ids[j]} {random_probe_ids[j+max_probes_per_measurement-1]}")
                    break
                response = requests.post(
                    f"https://atlas.ripe.net/api/v2/measurements/?key={options.ripe_key}",
                    json={
                        "definitions": [
                            {
                                "target": source,
                                "af": 4,
                                "packets": 1,
                                "proto": "ICMP",
                                # "size": 48,
                                "description": f"Tracerouter atlas to {source}, uuid {options.uuid}",
                                "skip_dns_check": True,
                                "type": "traceroute",
                            }
                        ],
                        "probes": [
                            {"value": ids_str, "type": "probes", "requested": len(ids_str)}
                        ],
                        "is_oneoff": True,
                        "bill_to": f"{options.ripe_account}",
                    },
                ).json()
                # measurement_ids.extend(response["measurement_ids"])
                print(response)
                if "error" in response:
                    time.sleep(30)
                else:
                    break

    # print("Sent")


def compute_intersections():
    '''
    Order the traceroutes by their intersection
    :return:
    '''
    connection = get_connection(database="traceroute_atlas", autocommit=0)

    # Get sources
    cursor = connection.cursor()
    query = (
        f" SELECT distinct(dest) from atlas_traceroutes "
    )
    cursor.execute(query)
    rows = cursor.fetchall()
    sources = [r[0] for r in rows]

    for source in sources:
        print(source)
        # Get the tr intersections to the source
        query = (
            f" SELECT at.dst, at.id, at.platform, ath.hop, ath.ttl "
            f" FROM atlas_traceroutes at "
            f" INNER JOIN atlas_traceroute_hops ath ON ath.trace_id=at.id "
            f" WHERE dest={source}"
            f" ORDER BY at.id, ath.ttl "
        )
        cursor.execute(query)
        rows = cursor.fetchall()

        traceroute_id_per_platform = {}
        traceroutes_per_intersection_ip = {}
        hops_per_traceroute_id = {}
        distance_to_destination_per_traceroute_id = {}

        for src, id, platform, hop, ttl in rows:
            # src is the mlab vp
            if src == hop:
                # reached destination at that hop
                distance_to_destination_per_traceroute_id[id] = ttl
            else:
                hops_per_traceroute_id.setdefault(id, []).append((hop, ttl))
            traceroute_id_per_platform[id] = platform
            traceroutes_per_intersection_ip.setdefault(hop, set()).add(id)


        # Get the RR intersections to the source
        query = (
            f" SELECT at.dest, at.id, at.platform, ath.hop "
            f" FROM atlas_traceroutes at "
            f" INNER JOIN atlas_rr_intersection arri arri.traceroute_id=at.id "
            f" WHERE dest={source}"
        )

        cursor.execute(query)
        rows = cursor.fetchall()

        # Same as above, fill the data structure
        for id, platform, hop in rows:
            traceroutes_per_intersection_ip.setdefault(hop, set()).add(id)

        # Now for each traceroutes, compute the weight of the traceroute
        weight_per_traceroute_id = {}
        distance_to_source_per_intersection = {}
        for tr_id, hops in hops_per_traceroute_id.items():
            for hop, ttl in hops:
                # Get the distance to the src
                dist_to_src = distance_to_destination_per_traceroute_id[tr_id] - ttl
                distance_to_source_per_intersection[hop] = dist_to_src
                weight_per_traceroute_id.setdefault(tr_id, 0)
                # Weight is the number of traceroutes going through this hop * the distance to the source
                weight_per_traceroute_id[tr_id] += dist_to_src * len(traceroutes_per_intersection_ip[hop])

        intersection_per_best_traceroute = {}
        # Now for each intersection, select the traceroute having the biggest weight
        for intersection_ip, traceroute_ids in traceroutes_per_intersection_ip.items():
            sorted_traceroute_ids = sorted([(weight_per_traceroute_id[tr_id], tr_id) for tr_id in traceroute_ids],
                                           key=lambda x: x[0])
            best_traceroute_id = max([weight_per_traceroute_id[tr_id] for tr_id in traceroute_ids])
            intersection_per_best_traceroute[intersection_ip] = best_traceroute_id
            # Substract weight from the traceroute having this intersection
            for tr_id in traceroute_ids:
                weight_per_traceroute_id[tr_id] -= len(traceroute_ids) * distance_to_source_per_intersection[intersection_ip]

        # Insert into the db
        for intersection_ip, tr_id in intersection_per_best_traceroute.items():
            query = (
                f" INSERT INTO atlas_intersections (src, intersection_ip, traceroute_id) VALUES ({source}, {intersection_ip}, {tr_id}) "
            )
            cursor.execute(query)
            connection.commit()




if __name__ == "__main__":

    '''
    This script must be run from rankingservice directory
    '''
    ######################################################################
    ## Parameters
    ######################################################################
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", help="either trace, refresh or fetch", type=str)
    parser.add_argument("--sources-file", help="source file containing sources, one per line", type=str)
    parser.add_argument("--sources-probes-file", help="file containing source, probeID lines", type=str)
    parser.add_argument("--uuid", help="a uuid to put in the label of the RIPE measurement, or to retrieve it", type=str)
    parser.add_argument("--n-traceroutes-per-source", help="Number of traceroutes to run per source", type=int)
    parser.add_argument("--dry-run", help="Do not run the measurements, just for debug", action='store_true')
    parser.add_argument("--dump-file", help="Output file to dump when fetching traceroutes results (must be a json)", type=str)
    parser.add_argument("--ripe-account", help="RIPE account to run RIPE traceroutes",
                        type=str)
    parser.add_argument("--ripe-key", help="RIPE API key",
                        type=str)
    parser.add_argument("--description", help="Description of the traceroutes given to RIPE",
                        type=str)
    args = parser.parse_args()

    if not args.ripe_key or not args.ripe_account:
        print(parser.error("Please provide a ripe key and a ripe account"))
        exit(1)
    if args.action not in ["trace", "refresh", "fetch"]:
        print(parser.error("Please provide a valid action (trace, fetch)"))
        exit(1)
    if args.action == "trace":
        if not args.uuid or not args.sources_file or not args.n_traceroutes_per_source:
            print(parser.error("uuid, sources-file and n-traceroutes-per-source options are mandatory with trace action"))
            exit(1)

    if args.action == "refresh":
        if not args.uuid or not args.sources_probes_file:
            print(parser.error("uuid, sources-probes-file options are mandatory with refresh action"))
            exit(1)

    if args.action == "fetch":
        if not args.uuid or not args.dump_file:
            print(parser.error("uuid and dump-file options are mandatory with fetch action"))
            exit(1)

    if args.action == "trace":
        sources_file = args.sources_file
        sources = []
        with open(sources_file) as f:
            for line in f:
                sources.append(line.strip())
        anchors_file = "resources/ripe_anchors.json"
        probes_file = "resources/ripe_probes.json"
        probes = load_probes_from_file(probes_file)
        anchors = load_probes_from_file(anchors_file)
        options = TraceOptions(args.ripe_account, args.ripe_key, args.uuid, args.n_traceroutes_per_source, args.dry_run)
        random_traceroutes_to_sources(sources, probes, anchors, options)

    elif args.action == "refresh":
        sources_probes_file = args.sources_probes_file
        sources_probes = {}
        with open(sources_probes_file) as f:
            for line in f:
                source, probe_id = line.split(",")
                sources_probes.setdefault(source, set()).add(probe_id)
        sources_probes = {s : list(sources_probes[s]) for s in sources_probes}
        options = RefreshOptions(args.ripe_account, args.ripe_key, args.uuid, args.dry_run, args.description)
        traceroutes_to_sources(sources_probes, options)

    elif args.action == "fetch":
        # Fetch the results associated with a uuid
        options = FetchOptions(args.ripe_account, args.ripe_key, args.uuid, args.dump_file)
        fetch_traceroutes(options)
