import json
from time import time


def lambda_handler(event, context):
    inkey = event['input_object_key']
    r = context['r']
    
    data = r.get(inkey).decode("utf-8")
    ts1 = time()

    json_data = json.loads(data)
    str_json = json.dumps(json_data, indent=4)
    ts2 = time()

    return [ts1, ts2]
