import argparse
import ipaddress
from atlas.atlas_database import get_connection, load_midar_file_into_database

class Options(object):
    def __init__(self, ofile, label):
        self.ofile = ofile
        self.label = label

def extract_ips(options):
    '''
    Extract the IP addresses of the atlas to allow better intersection.
    :param options:
    :return:
    '''
    connection = get_connection("traceroute_atlas")
    cursor = connection.cursor()

    query = (
        f" SELECT distinct(hop) from atlas_traceroute_hops h "
        f" INNER JOIN atlas_traceroutes at on h.trace_id=at.id"
        f" WHERE platform='{options.label}'"
        f" UNION "
        f" SELECT distinct(rr_hop) from atlas_rr_pings rrp "
        f" inner join atlas_traceroutes at ON rrp.traceroute_id = at.id "
        f" WHERE platform='{options.label}'"
    )

    cursor.execute(query)
    rows = cursor.fetchall()

    ips = set()
    for hop in rows:
        ip = ipaddress.ip_address(hop[0])
        if ip.is_private:
            continue
        ips.add(str(ip))

    with open(options.ofile, "w") as f:
        for ip in ips:
            f.write(ip + "\n")

    cursor.close()
    connection.close()
if __name__ == "__main__":
    ######################################################################
    ## Parameters
    ######################################################################
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", help="Either load or extract. Load insert aliases in the DB, while extract gets "
                                       "alias candidates from the DB ", type=str)
    parser.add_argument("--ofile", help="Output file to write IP candidates for alias resolution (extract mode)", type=str)
    parser.add_argument("--ifile", help="Input file to load IP aliases in DB (load mode). Format is a midar-noconflicts.sets type",
                        type=str)
    parser.add_argument("--label",
                        help="Label of the platform used for the traceroute atlas",
                        type=str)
    # parser.add_argument("--id", help="traceroute id", type=int)
    args = parser.parse_args()

    if args.mode not in ["load", "extract"]:
        print("mode is mandatory. possible values are load, extract")
        exit(1)

    if args.mode == "extract":
        if not args.ofile or not args.label:
            print("ofile is mandatory")
            exit(1)
        extract_ips(Options(args.ofile, args.label))

    elif args.mode == "load":
        if not args.ifile:
            print("ifile is mandatory")
            exit(1)
        load_midar_file_into_database(args.ifile)
