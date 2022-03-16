import datetime
import io
import os
import shutil
import uuid
import zlib
from time import time

# from . import storage
# client = storage.storage.get_instance()

tmp = '/dev/shm/'

def parse_directory(directory):

    size = 0
    for root, dirs, files in os.walk(directory):
        for file in files:
            size += os.path.getsize(os.path.join(root, file))
    return size

def lambda_handler(event, ctx):
    r = ctx['r']
    input_object_key = event.get('input_object_key')
    output_object_key = event.get('output_object_key')

    download_path = '{}/{}'.format(tmp, 'compression')
    os.makedirs(download_path, exist_ok=True)

    s3_download_begin = datetime.datetime.now()
    with open('%s/%s' % (download_path, input_object_key), 'wb') as f:
        f.write(r.get(input_object_key))
    s3_download_stop = datetime.datetime.now()
    size = parse_directory(download_path)

    compress_begin = datetime.datetime.now()
    ts1 = time()
    shutil.make_archive(os.path.join(tmp, output_object_key), 'zip', root_dir=download_path, base_dir=download_path)
    ts2 = time()
    compress_end = datetime.datetime.now()

    s3_upload_begin = datetime.datetime.now()
    archive_name = '{}.zip'.format(input_object_key)
    with open('%s/%s.zip' % (tmp, output_object_key), 'rb') as f:
        r.set(output_object_key, f.read())
    s3_upload_stop = datetime.datetime.now()
    shutil.rmtree(download_path)
    os.remove('%s/%s.zip' % (tmp, output_object_key))
    download_time = (s3_download_stop - s3_download_begin) / datetime.timedelta(microseconds=1)
    upload_time = (s3_upload_stop - s3_upload_begin) / datetime.timedelta(microseconds=1)
    process_time = (compress_end - compress_begin) / datetime.timedelta(microseconds=1)
    return [ts1, ts2]