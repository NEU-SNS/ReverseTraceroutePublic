import unittest
import protos.rankingservice_pb2 as rankingservice_pb2
from rankingservice_server import RankingServicer

from algorithms.database import db_host, clickhouse_pwd


class TestGetTargetsFromHitlistRequest():
    def __init__(self):
        self.is_initial = True
        self.is_only_addresses = True
        self.ip_addresses = ["8.8.8.8"]


class FakeGRPCContext(object):

    def set_code(self, code):
        pass
    def set_details(self, details):
        pass

class TestRankingServiceMethods(unittest.TestCase):


    def setUp(self):
        ipasnfile = 'resources/bgpdumps/latest'
        self.ranking_servicer = RankingServicer(db_host, clickhouse_pwd, "revtr", ipasnfile)

    # def test_getTargetsFromHitlist(self):
    #     res = self.ranking_servicer.getTargetsFromHitlist(TestGetTargetsFromHitlistRequest(), None)
    #     print(res.targets)

    def test_getVP(self):

        req = rankingservice_pb2.GetVPsReq()
        req.ip = "36.68.72.1"
        req.numVPs = 5
        ranking_techniques = ["random_cover",
                              "ingress_cover",
                              "global_cover",
                              "destination_cover"]

        for technique in ranking_techniques:
            req.rankingTechnique = technique
            res = self.ranking_servicer.getVPs(req, context=FakeGRPCContext())
            res = res.vps
            print(req.rankingTechnique, res)
            self.assertTrue(0 < len(res) <= req.numVPs)


if __name__ == '__main__':
    unittest.main()