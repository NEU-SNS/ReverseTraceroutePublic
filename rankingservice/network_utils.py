'''
    Some helper functions on networking and IP addresses
'''

# import socket, pyasn
import socket
import sys
from collections import defaultdict, namedtuple
import struct


# import timeit
# mysetup = "import socket"
# mycode = '''
# def long2ip():
#     return socket.inet_ntoa(struct.pack("!L", 20000))
# '''
#
# print(timeit.timeit(setup=mysetup, stmt=mycode, number=1000000))
#
#
# mycode = '''
# def ipItoStr():
#     ip=20000
#     return f"{str((ip & 0xFF000000) >> 24)}.{str((ip & 0x00FF0000) >> 16)}.{str((ip & 0x0000FF00) >> 8)}.{str(ip & 0x000000FF)}"
# '''
#
# print(timeit.timeit(setup=mysetup, stmt=mycode, number=1000000))

def prefix_24_from_ip(ip):
    q = (ip & 0xFFFFFF00)
    return q


def s24_of(ip):
    octets = ip.split('.')
    return ".".join(octets[:3]) + ".0/24"

def ipItoStr(ip):
    '''
    Use this function rather than python ipaddress.ip_address, way faster
    :param ip:
    :return:
    '''
    return f"{str((ip & 0xFF000000) >> 24)}.{str((ip & 0x00FF0000) >> 16)}.{str((ip & 0x0000FF00) >> 8)}.{str(ip & 0x000000FF)}"

def network_from_ip(ip, mask):
    bits_0 = 32 - mask
    return ip & ((2**32 - 1) - (2** bits_0 - 1))

def ipv4_index_of( addr ):
    try:
        as_bytes = socket.inet_aton(addr)
    except OSError as e:
        sys.exit("on trying socket.inet_aton({}), received error {}".format(addr, e))
    return int.from_bytes(as_bytes, byteorder='big', signed=False)

DnetEntry = namedtuple('DnetEntry',['asn','bgp'])

def dnets_of(ip, ipasn, ip_representation = None):
    if ip_representation == "uint32":
        ip = ipItoStr(ip)
    try:
        return DnetEntry(*(ipasn.lookup(ip)))
    except:
        return DnetEntry('','',)


def bgpDBFormat(dentry):
    bgp_prefix_start = ipv4_index_of(dentry.bgp.split('/')[0]) if dentry.bgp else 0
    bgp_prefix_length = int(dentry.bgp.split('/')[1]) if dentry.bgp else 0
    if bgp_prefix_start == 0:
        bgp_prefix_end = 0
    else:
        bgp_prefix_end = bgp_prefix_start + (2**(32 - bgp_prefix_length) - 1)
    return bgp_prefix_start, bgp_prefix_end


def isValidIP(ip):
    return len(ip.split('.')) == 4

if __name__ == "__main__":

    ip = "132.216.0.1"
    import ipaddress
    ipI = int(ipaddress.ip_address(ip))
    ipStr = ipItoStr(ipI)
    assert(ipStr == ip)

    dnetEntry = DnetEntry(0, "132.0.0.128/25")
    bgp_start, bgp_end = bgpDBFormat(dnetEntry)
    assert(bgp_end == int(ipaddress.ip_address("132.0.0.255")))

    address = ipaddress.ip_address("8.8.8.1")
    network = network_from_ip(int(address), 30)
    assert(network == int(ipaddress.ip_network(str(address) + "/30", strict=False).network_address))
