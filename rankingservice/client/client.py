import logging
import grpc
import os
import sys
import rankingservice_pb2
import rankingservice_pb2_grpc
import rankingservice_utils as rs_utils


def run():
    ip = "160.39.30.232"
    port = "49491"
    channel = ip + ":" + port
    with grpc.insecure_channel(channel) as channel:

        k = int(sys.argv[1])
        print("Connecting to " + ip + " through port " + port)
        stub = rankingservice_pb2_grpc.RakingStub(channel)

        print("------Get VPs------")

        req = rankingservice_pb2.getVPsReq()

        req.ip = "104.245.114.100"
        req.numVPs = k
        req.rankingTechnique = "TK"
        req.executedMeasurements.extend(["1495083626"])

        resp = stub.getVPs(req)
        print(resp)


if __name__ == '__main__':
    logging.basicConfig()
    run()
