import socket
import struct
import MySQLdb

STOP_COUNT_THRESH = 10
TIME_MINUTE_THRESH = 10
SIGNIFICANT_THRESH = 10
db_host = ""
db_pwd = ""

def connectDB(type):
    if(type == "measurement"):
        db = MySQLdb.connect(host=db_host,
                         user="root",
                         passwd=db_pwd,
                         db="ccontroller")
    elif(type == "ranking"):
        """ (this is from AWS RDS, testing with dummy data)
        db = MySQLdb.connect(host="rtr-offline-aurora-db-inst-final-snapshot-reader.cdutp9pdmnzk.us-east-2.rds.amazonaws.com",
                             user="as5281",
                             passwd="datapants",
                             port=3306,
                             db="rankings")
        """
        db = MySQLdb.connect(host=db_host,
                             user="root",
                             passwd=db_pwd,
                             db="ranking")

    return db

def readDB(db, sql, args=None):
    """
    Returns the result of the <sql> query in the form of a Pandas dataframe.

    :param db: database connection to use.
    :param sql: sql query
    :param args: tuple, arguments for sql query.
    :return: the resulting rows of the <sql> query in the form of a Pandas dataframe.
    """
    cur = db.cursor()
    cur.execute(sql, args)

    """  sadly, Walter has an outdated python version and pandas cannot be installed.
    df = pd.DataFrame(cur.fetchall())
    if(not df.empty):
        fields = list(map(lambda x: x[0], cur.description))
        df.columns = fields
    """

    rows = cur.fetchall()
    rows_formatted = []

    if(rows):
        fields = list(map(lambda x: x[0], cur.description))
        for entry in rows:
            rows_formatted.append(dict(zip(fields, entry)))

    return rows_formatted

def getField(rows, key):
    """
    :param rows: array of dict with multiple keys.
    :return: array with only the value for the key of interest, for each row.
    """
    return [x[key] for x in rows]

def closeDB(db):
    db.close()

def ip2long(ip):
    packedIP = socket.inet_aton(ip)
    return struct.unpack("!L", packedIP)[0]

def long2ip(ipnum):
    return socket.inet_ntoa(struct.pack("!L", ipnum))

def destIsUnresponsive(dest, rows):
    return None # TODO

def keepVPsOfInterest(db, current_vpset, hist_measurements, m_ids):
    """

    :param db: ReverseTraceroute Skyler DB connection.
    :param current_vpset: Current set of VP sites.
    :param hist_measurements: Cached measurements.
    :param m_ids: List of ping_id's that have been provided by the user.
    :return: The updated set of VP sites that pass the filter.
    """

    # Discard usedVPs: Get vp site for provided ping ids (m_ids), if any.
    usedVPs_sites = set()
    if(m_ids):
        placeholder = '%s'
        placeholders = ", ".join(placeholder for x in m_ids)
        usedVPsSQL = """
        SELECT DISTINCT site 
        FROM pings
        INNER JOIN vpservice.vantage_points as vp
        ON vp.ip = pings.src
        WHERE pings.id IN (%s)
        """ % placeholders

        usedVPs_sites = readDB(db, usedVPsSQL, tuple(m_ids))
        usedVPs_sites = set([x["site"] for x in usedVPs_sites])
        #usedVPs_sites = usedVPs_sites[["site"]].drop_duplicates()

    # Get historically unresponsive vp sites.
    print("getting historically unresponsive vp sites")
    hist_unresponsive_sites = set([x["site"] for x in hist_measurements if x["loss"] == 1])
    #hist_unresponsive_sites = hist_measurements[hist_measurements["loss"] == 1][["site"]].drop_duplicates()

    # Filter out bad sites.
    print("filtering out bad sites")
    current_vpset = current_vpset.difference(usedVPs_sites)
    current_vpset = current_vpset.difference(hist_unresponsive_sites)
    # current_vpset = current_vpset.merge(usedVPs_sites, how="left", on="site", indicator=True)
    # current_vpset = [current_vpset["_merge"] == "left_only"][["site"]]
    # current_vpset = current_vpset.merge(hist_unresponsive_sites, on="site", how="left", indicator=True)
    # current_vpset = [current_vpset["_merge"] == "left_only"][["site"]]


    # Choose a VP per site.
    # Take an active VP within the site. If no active VP, discard site.
    vps = set()
    for site_id in current_vpset:
        print("check if %s has at least one active VP" % site_id)
        sql = """
        SELECT 
            VPS.site 
        FROM 
            vpservice.vantage_points VPS
            left outer join vpservice.quarantined_vps QVPS on VPS.hostname = QVPS.hostname
        WHERE QVPS.hostname is null AND VPS.site = %s
        LIMIT 1
        """
        vp_id_rows = readDB(db, sql, (site_id,))
        if(vp_id_rows):
            print("Yes it does! Keeping the site in the set.")
            vps.add(vp_id_rows[0]["site"])
        else:
            print("All VPs are quarantined, discarding the site.")

    return vps



