from mrtparse import Reader, MRT_T, TD_V2_ST, TD_ST
import json

def generate_as_paths(mrt_file_path, ofile, filter_asn):
    r = Reader(mrt_file_path)
    n = 0
    as_paths = {}
    while True:
        # if n > 10:
        #     break
        try:
            m = r.next()
        except StopIteration:
            break
        # print(m.data["type"], m.data["subtype"])
        if m.err:
            continue
        n += 1
        if n % 1000 == 0:
            print(n)
            # print(".", end="", flush=True)
            # if n == 2000:
            #     break

        # prefix = "0.0.0.0/0"
        if (
                (m.data["type"][0] == MRT_T["TABLE_DUMP_V2"]
            and m.data["subtype"][0] == TD_V2_ST["RIB_IPV4_UNICAST"]) or
                (m.data["type"][0] == MRT_T["TABLE_DUMP"]
            and m.data["subtype"][0] == TD_ST["AFI_IPv4"])
        ):
            prefix = m.data["prefix"] + "/" + str(m.data["prefix_length"])
            if prefix not in filter_asn:
                continue
            as_paths_to_add = [
                v["path_attributes"][1]["value"][0]["value"]
                for v in m.data["rib_entries"]
            ]
            as_paths_to_add = (tuple(x) for x in as_paths_to_add)
            for as_path in as_paths_to_add:
                as_paths.setdefault(prefix, set()).add(as_path)
                print(prefix, as_path)

    as_paths = {p: list(as_paths[p]) for p in as_paths}
    with open(ofile, "w") as f:
        json.dump(as_paths, f)
        return as_paths

def main():
    mrt_file_path = "resources/bgpdumps/rib.20201201.1200"
    ofile = "atlas/evaluation/resources/as_paths.json"
    filter_prefix_file = "asns_source_destination.json"

    with open(filter_prefix_file) as f:
        filter_prefixes = json.load(f)
        filter_prefixes_sources = filter_prefixes["sources"]
        filter_prefixes_destinations = filter_prefixes["destinations"]
        filter_prefixes = []
        filter_prefixes.extend(filter_prefixes_sources)
        filter_prefixes.extend(filter_prefixes_destinations)
        filter_prefixes = {x[1] for x in filter_prefixes}
        generate_as_paths(mrt_file_path, ofile, filter_asn=filter_prefixes)

def dummy():
    import socket
    import ipaddress
    i = 3194134067
    staging_nodes_reboot = ["revtr-mlab1-lis02.mlab-oti.measurement-lab.org",
    "revtr-mlab1-yyc02.mlab-oti.measurement-lab.org"    ,
    "revtr-mlab2-yyc02.mlab-oti.measurement-lab.org"    ,
    "revtr-mlab3-lis02.mlab-oti.measurement-lab.org"    ,
    "revtr-mlab3-yyc02.mlab-oti.measurement-lab.org"    ,
    "revtr-mlab4-beg01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-bru03.mlab-staging.measurement-lab.org",
    "revtr-mlab4-bru05.mlab-staging.measurement-lab.org",
    "revtr-mlab4-del01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-del02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-dfw02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-fra01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-gru04.mlab-staging.measurement-lab.org",
    "revtr-mlab4-ham02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-iad04.mlab-staging.measurement-lab.org",
    "revtr-mlab4-lax06.mlab-staging.measurement-lab.org",
    "revtr-mlab4-lhr04.mlab-staging.measurement-lab.org",
    "revtr-mlab4-lis02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-maa02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-mad02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-mia04.mlab-staging.measurement-lab.org",
    "revtr-mlab4-mil06.mlab-staging.measurement-lab.org",
    "revtr-mlab4-nuq02.mlab-staging.measurement-lab.org",
    "revtr-mlab4-svg01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-tnr01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-tun01.mlab-staging.measurement-lab.org",
    "revtr-mlab4-yyc02.mlab-staging.measurement-lab.org"]



    for node in staging_nodes_reboot:
        ip = socket.gethostbyname( "revtr-mlab4-gru04.mlab-staging.measurement-lab.org")
        print(int(ipaddress.ip_address(ip)))


    with open("/Users/kevinvermeulen/Downloads/revtrvp.log") as f:
        get_probes = 0
        sending_back = 0
        in_sending = False
        in_receiving = True
        for line in f:
            if "getProbe" in line:
                if in_sending:
                    print("Sending back", sending_back)
                in_receiving = True
                in_sending = False

                get_probes += 1
                sending_back = 0

            if "Sending back" in line:
                if in_receiving:
                    print("Received", get_probes)
                in_receiving = False
                in_sending = True
                sending_back += 1
                get_probes = 0


if __name__ == "__main__":
    dummy()