from time import time
import re
import io
import subprocess
import sys

cleanup_re = re.compile('[^a-z]+')
tmp = '/dev/shm/'

def run_and_check(command, msg):
    ret, output = subprocess.getstatusoutput(command)
    print(output, file=sys.stdout)
    if ret != 0:
        raise Exception(msg)

def lambda_handler(event, context):
    input_object_key = event['input_object_key']
    output_object_key = event['output_object_key']  # example : lr_model.pk
    r = context['r']
    
    with open('%s/%s' % (tmp, input_object_key), 'wb') as f:
        f.write(r.get(input_object_key))

    ts1 = time()
    run_and_check('ffmpeg -y -i %s/%s -vf hflip %s/%s' % (tmp, input_object_key, tmp, output_object_key), 'ffmpeg')
    ts2 = time()
    with open('%s/%s' % (tmp, output_object_key), 'rb') as f:
        r.set(output_object_key, f.read())

    run_and_check('rm %s/%s %s/%s' % (tmp, input_object_key, tmp, output_object_key), 'rm')
    return [ts1, ts2]

if __name__ == '__main__':
    print(lambda_handler({'input_object_key': 'ml-dataset', 'output_object_key': 'ml-model'}, None))
