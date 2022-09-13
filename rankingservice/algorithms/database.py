from clickhouse_driver import Client
from mysql.connector import connect
from subprocess import Popen, PIPE
from network_utils import dnets_of, bgpDBFormat
from multiprocessing import Pool
import tqdm
import istarmap  # import to apply patch
import mysql


# clickhouse_pwd = mysql_pwd

db_host="localhost"
clickhouse_pwd = ""

mysql_host = ""
mysql_user = ""
mysql_pwd = ""  # Should be put in argument

ping_table = "pings"
ping_responses_table = "ping_responses"
record_route_table = "record_routes"

ping_table += "_debug"
ping_responses_table += "_debug"
record_route_table += "_debug"

def get_connection(database, autocommit=1):
    connection = connect(host=mysql_host,
                         user=mysql_user,
                         passwd=mysql_pwd,
                         db=database,
                         use_pure=False,
                         # connect_timeout=300,
                         raise_on_warnings=True,
                         autocommit=autocommit,
                         auth_plugin='mysql_native_password'
                         # max_allowed_packets=512 * 1024 * 1024
                         )

    return connection



def createPingsTable(host, db, table):

    client = Client(host, password=clickhouse_pwd)
    client.execute(
        f"CREATE TABLE IF NOT EXISTS {db}.{table} ("
        f"dst_asn Nullable(UInt32), dst_bgp_prefix_start Nullable(UInt32), dst_bgp_prefix_end Nullable(UInt32), dst_24 UInt32, " # Network metadata of dst
        f"ping_id UInt64," 
        f"src UInt32, dst UInt32, is_record_route UInt8, loss Float64, "
        f"created Datetime, vp_site String )"
        f"ENGINE=MergeTree() "
        f"ORDER BY (dst, src)"
        )

def createRRPingsTable(host, password, db, table, order_by):

    if password is None:
        client = Client(host)
    else:
        client = Client(host, password=password)

    client.execute(
        f"CREATE TABLE IF NOT EXISTS {db}.{table} ("
        f"dst_asn Nullable(UInt32), dst_bgp_prefix_start Nullable(UInt32), dst_bgp_prefix_end Nullable(UInt32), dst_24 UInt32, " # Network metadata of dst
        f"rr_asn Nullable(UInt32), rr_bgp_prefix_start Nullable(UInt32), rr_bgp_prefix_end Nullable(UInt32), rr_24 UInt32, " # Newtork metadata of RR
        f"ping_id UInt64," 
        f"src UInt32, dst UInt32, reply_from UInt32, rr_hop UInt8, "
        f"rr_ip UInt32, created Datetime, vp_site String )"
        f"ENGINE=MergeTree() "
        # f"ORDER BY (ping_id, src, dst, rr_hop)"
        f"ORDER BY {order_by} "
        )


def dropTable(host, password, db, table):
    if password is None:
        client  = Client(host)
    else:
        client = Client(host, password=password)
    client.execute(f"DROP TABLE IF EXISTS {db}.{table}")

def dump_query(host, pwd, query):
    cmd = f"clickhouse-client --host={host} "
    if pwd is not None and pwd != "":
        cmd += f" --password={pwd} "
    cmd += f" --query=\"{query}\""
    print(cmd)
    process = Popen(cmd, stdout=PIPE, stderr=PIPE, shell=True)
    out, err = process.communicate()
    out = [line.decode("utf-8") for line in out.splitlines()]
    err = [line.decode("utf-8") for line in err.splitlines()]
    for line in out:
        line = line.replace("\\n", " ")
        print(line)
    for line in err:
        print(line)

def insert_from_file(host, pwd, csv_file, query):

    cmd = f"cat {csv_file} | clickhouse-client --host={host} "
    if pwd is not None and pwd != "":
        cmd += f" --password={pwd} "
    cmd += f"--query=\"{query}\""
    print(cmd)
    process = Popen(cmd, stdout=PIPE, stderr=PIPE, shell=True)
    out, err = process.communicate()
    out = [line.decode("utf-8") for line in out.splitlines()]
    err = [line.decode("utf-8") for line in err.splitlines()]
    for line in out:
        print(line)
    for line in err:
        print(line)

def flush_batch(client, db, table, batch):

    client.execute(f"INSERT INTO {db}.{table} ("
                   f"dst_asn, dst_bgp_prefix_start, dst_bgp_prefix_end ,dst_24, "
                   f"rr_asn, rr_bgp_prefix_start, rr_bgp_prefix_end, rr_24,"
                   f" ping_id, src, dst, reply_from, rr_hop, rr_ip, created, vp_site) VALUES "
    ,
    batch)

import pyasn
ipasnfile = 'resources/bgpdumps/latest'
ip2asn = pyasn.pyasn(ipasnfile)

def dump_subspace(host, db, table, query, label_filter):
    import time, random
    # Sleep random time to be able to connect

    # time.sleep(random.randint(0, 5))
    while True:
        connection = connect(host=mysql_host,
                             user="root",
                             passwd=mysql_pwd,
                             db="ccontroller",
                             use_pure=False,
                             # connect_timeout=300,
                             raise_on_warnings=True,
                             autocommit=1,
                             # max_allowed_packets=512 * 1024 * 1024
                             )
        try:
            # print(query)
            # return
            # print(query)
            # try:
                # cursor = connection.cursor(buffered=True)
            cursor = connection.cursor()
            # cursor.execute("SET @@local.max_statement_time=60000;")
            # cursor.execute("SET @@local.net_read_timeout=300;")
            # cursor.execute("SET @@local.net_write_timeout=300;")
            cursor = connection.cursor()
            cursor.execute(query, )
            print(query)
            batch = []
            rows = cursor.fetchall()
            # print(f"Fetched {len(rows)} rows")
            i = 0
            for row in rows:
                i += 1
                dst_24, rr_24, ping_id, src, dst, dst_from, rr_hop, rr_ip, created, vp_site, label = row
                if label != label_filter:
                    continue
                # Get ASN and BGP information from dst
                dentry_dst = dnets_of(dst, ip2asn, ip_representation="uint32")
                bgp_dst_start, bgp_dst_end = bgpDBFormat(dentry_dst)
                # Get ASN and BGP information from rr_ip
                dentry_rr = dnets_of(rr_ip, ip2asn, ip_representation="uint32")
                bgp_rr_start, bgp_rr_end = bgpDBFormat(dentry_rr)
                # Labeled row with network infos
                labeled_row = dentry_dst.asn, bgp_dst_start, bgp_dst_end, dst_24, \
                              dentry_rr.asn, bgp_rr_start, bgp_rr_end, rr_24, ping_id, src, dst, dst_from, rr_hop, rr_ip, created, vp_site
                batch.append(labeled_row)

                # if len(batch) == insert_batch_size:
                    # Flush the batch into clickhouse
                    # flush_batch(client, db, table, batch)
                    # if not connection.is_connected():
                    #     print("error connection lost")
                    # batch.clear()
            # rows = cursor.fetchmany(batch_size)
            client = Client(host, password=clickhouse_pwd)
            flush_batch(client, db, table, batch)
            cursor.close()
            connection.close()

            # connection.ping()
            client.disconnect()

            break
        except (mysql.connector.errors.InterfaceError, mysql.connector.DatabaseError, Exception) as e:
            print(e)
            connection.close()
    # print("Done batch\n")

    # finally:
    #     print("Closing connection")


def setup_rr_table(host, db, table, label, is_create_new_table):
    # Setup the clickhouse database it in the clickhouse database
    if is_create_new_table:
        dropTable(host, clickhouse_pwd, db, table)
        createRRPingsTable(host, None, db, table, "(ping_id, src, dst, rr_hop)")
        # createRRPingsTable(host, None, db, table, order_by)

    n_process_pool = 32
    # First get the number of rows
    connection = connect(host=mysql_host,
                         user="root",
                         passwd=mysql_pwd,
                         db="ccontroller",
                         )

    cursor_settings = connection.cursor()
    # cursor_settings.execute('SET GLOBAL connect_timeout=default; '
    #                         'SET GLOBAL interactive_timeout=default;'
    #                         'SET GLOBAL wait_timeout=default;'
    #                         'SET GLOBAL net_write_timeout=70; ', multi=True)
    # cursor_settings.execute(f"SET GLOBAL max_allowed_packet={256 * 1024 * 1024} ;")

    cursor_settings.close()

    cursor = connection.cursor()
    query = (
        f" SELECT min(id), max(id) "
        f" FROM ccontroller.{ping_table} "
        f" WHERE label=\"{label}\""
    )
    cursor.execute(query, )
    rows = cursor.fetchall()
    assert (len(rows) == 1)
    min_id, max_id = rows[0][0], rows[0][1]
    cursor.close()
    connection.commit()
    connection.close()

    with Pool(n_process_pool) as p:
        n_rows = max_id - min_id + 1
        # Make the split so that n_rows < 50000
        # split = int((n_rows * 9) / 50000)
        split = 1024
        batch_args = []
        for j in range(0, split + 1):
            inf_born = int(j * ((n_rows) / split)) + min_id
            sup_born = int((j + 1) * ((n_rows) / split)) + min_id

            query = (
                f"SELECT p.dst & 0xFFFFFF00, rr.ip  & 0xFFFFFF00, p.id, p.src, p.dst, pr.from, rr.hop, rr.ip, p.created, vps.site, p.label "
                f"FROM  ccontroller.{ping_responses_table} as pr "
                f"INNER JOIN ccontroller.{ping_table} as p ON pr.ping_id = p.id "
                f"INNER JOIN ccontroller.{record_route_table} as rr on rr.response_id=pr.id  "
                f"INNER JOIN vpservice.vantage_points as vps on vps.ip = p.src "
                f"WHERE "
                f" p.id BETWEEN {inf_born} AND {sup_born} AND "
                # f"p.created >= {start_date} AND "
                f" p.record_route=1 AND pr.icmp_type in (0, 11)  "
                # f" LIMIT 1000"
                # f"AND p.label=\"{label}\" "
                # f"INTO OUTFILE 'algorithms/evaluation/resources/rr_ranking_evaluation_bgp_dump.csv' "
                # f"FIELDS TERMINATED BY ',' LINES TERMINATED BY '\n';"
            )
            batch_args.append((host, db, table, query, label))
            if len(batch_args) == n_process_pool:
                list(tqdm.tqdm(p.istarmap(dump_subspace, batch_args), total=len(batch_args)))
                batch_args.clear()
                print(f"Batch {j} done.")
        list(tqdm.tqdm(p.istarmap(dump_subspace, batch_args), total=len(batch_args)))
            # p.starmap(dump_subspace, args)

    connection.close()


def dump_table(host, db, pwd, table, tmp_file, format):
    print(table)
    query = f"SELECT * FROM {db}.{table} INTO OUTFILE '{tmp_file}' FORMAT {format}"
    dump_query(host, pwd, query)

def dump_database():

    host = db_host
    database = "revtr"
    client = Client(host, password=clickhouse_pwd, database=database)
    query = f"SHOW TABLES "
    tables = client.execute(query)
    # tables = ["revtr.RRPings"]
    client.disconnect()
    for table in tables:
        tmp_file = f"resources/{table}.native"
        dump_table(host, database, clickhouse_pwd, table[0], tmp_file, "Native")

def dump_schema():
    host = db_host
    database = "revtr"
    client = Client(host, password=clickhouse_pwd, database=database)
    query = f"SHOW TABLES "
    tables = client.execute(query)
    # tables = ["revtr.RRPings"]
    client.disconnect()
    for table in tables:
        print(table)
        table = table[0]
        query = f"SHOW CREATE TABLE {database}.{table}"
        dump_query(db_host, clickhouse_pwd, query)

def load_database():
    host = "localhost"
    database = "revtr"
    client = Client(host, database=database)
    query = f"SHOW TABLES "
    tables = client.execute(query)
    # tables = ["revtr.RRPings"]
    client.disconnect()
    for table in tables:
        print(table)
        table = table[0]
        tmp_file = f"resources/{table}.native"
        query = f"INSERT INTO {database}.{table} FORMAT Native"
        insert_from_file(host, None, tmp_file, query)

def setup_ranking_table():
    label = "ranking_evaluation_2021-10-28"
    table = label.replace("-", "_")
    database = "revtr"
    setup_rr_table(db_host, database, table, label, is_create_new_table=True)

def setup_ping_rr_table(table, label):
    database = "revtr"
    setup_rr_table(db_host, database, table, label, is_create_new_table=True)


def setup_ping_rr_table_assume_symmetry():
    table = "rr_ping_assume_symmetry_responsiveness"
    label = "assume_symmetry_snmp_30_rr_responsiveness"
    setup_ping_rr_table(table, label)

def setup_ping_rr_table_accuracy_forward_traceroute():
    table = "rr_ping_accuracy_forward_traceroutes"
    label = "accuracy_forward_traceroutes_3"
    setup_ping_rr_table(table, label)

def setup_ping_rr_table_destination_based_routing():
    table = "rr_ping_destination_based_routing_routers_violation"
    label = "bgp_survey_db_routing_no_cache_fix_routers_violation"
    setup_ping_rr_table(table, label)

if __name__ == "__main__":
    setup_ping_rr_table_destination_based_routing()
    # dump_schema()
    # setup_ping_rr_table_accuracy_forward_traceroute()
    # format = "Native"
    # tmp_file = f"algorithms/evaluation/resources/{table}.{format}"
    # insert_from_file("localhost", None, tmp_file, f"INSERT INTO {database}.{table} FORMAT Native ")