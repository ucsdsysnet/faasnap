#!/usr/bin/env python3

import subprocess

from flask import Flask
app = Flask(__name__)

@app.route('/')
def hello_world():
    return 'Hello, World!'

@app.route('/dmesg')
def dmesg():
    ret, output = subprocess.getstatusoutput('dmesg')
    return output

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
