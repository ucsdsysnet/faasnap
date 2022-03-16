import datetime
import igraph
from time import time

def lambda_handler(event, ctx):

    size = int(event.get('size'))

    graph = igraph.Graph.Barabasi(size, 10)

    ts1 = time()
    result = graph.pagerank()
    ts2 = time()

    return [ts1, ts2]
