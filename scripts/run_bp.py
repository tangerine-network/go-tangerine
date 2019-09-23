#!/usr/bin/env python3
#
# Copyright 2019 The go-tangerine Authors
# This file is part of the go-tangerine library.
#
# The go-tangerine library is free software: you can redistribute it
# and/or modify it under the terms of the GNU Lesser General Public License as
# published by the Free Software Foundation, either version 3 of the License,
# or (at your option) any later version.
#
# The go-tangerine library is distributed in the hope that it will be
# useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
# General Public License for more details.
#
# You should have received a copy of the GNU Lesser General Public License
# along with the go-tangerine library. If not, see
# <http://www.gnu.org/licenses/>.


import argparse
import hashlib
import json
import os
import platform
import random
import re
import socket
import subprocess
import sys
import time


try:
    import ntplib
except Exception:
    print('Please run `pip3 install ntplib\'')
    sys.exit(1)


CONTAINER_NAME = 'tangerine'
NUM_SLOTS = 5
POLLING_INTERVAL = 30
SLEEP_RAND_RANGE = 1800
TANGERINE_IMAGE_TMPL = 'byzantinelab/go-tangerine:latest-%s-%d'
TOOLS_IMAGE = 'byzantinelab/tangerine-tools'
WHITELISTED_FILES = [
    os.path.basename(sys.argv[0]),
    'datadir',
    'node.key'
]


def get_shard_id(nodekey):
    """Return shard ID.

    Shard ID is calculate as (first byte of sha3 of Node Key address) mode NUM_SLOTS.
    """
    wd = os.getcwd()
    p = subprocess.Popen(['docker', 'run', '-v', '%s:/mnt' % wd, '-t', TOOLS_IMAGE,
                          'nodekey', 'inspect', '/mnt/%s' % nodekey],
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()

    address = re.search('^Node Address: (0x[0-9a-fA-F]{40})', stdout.decode('utf-8')).group(1)
    return int(hashlib.sha256(address.encode('utf-8')).hexdigest()[0], base=16) % NUM_SLOTS


def get_tangerine_image(args):
    """Return the tangerine image by shard-ID."""
    return TANGERINE_IMAGE_TMPL % ('testnet' if args.testnet else 'mainnet',
                                   get_shard_id(args.nodekey))


def get_time_delta():
    """Compare time with NTP and return time delta."""
    c = ntplib.NTPClient()
    response = c.request('tw.pool.ntp.org', version=3)
    return abs(response.offset)


def generate_node_key(nodekey):
    """Generate a new node key."""
    if os.path.exists(nodekey):
        raise RuntimeError('node.key already exists, abort.')

    wd = os.getcwd()
    subprocess.Popen(['docker', 'run', '-v', '%s:/mnt' % wd, '-t', TOOLS_IMAGE,
                      'nodekey', 'generate', '/mnt/%s' % nodekey]).wait()

    print('Node key generated')
    print('\n\033[5;91;49mPlease backup node.key in a secure place!\033[0m')
    print('\n\033[0;91mSend at least 1 ETH (use Rinkeby ETH for testnet) to node key address!\033[0m')
    print('\033[0;91mThese ETHs are used for then Tangerine network recovery mechanism.\033[0m\n\n')


def get_image_version(image):
    """Get a docker image's version."""
    p = subprocess.Popen(['docker', 'inspect', '-f', '{{.Id}}', image],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()
    return stdout


def update_image(image):
    """Update a given docker image."""
    subprocess.Popen(['docker', 'pull', image],
                     stdout=subprocess.PIPE, stderr=subprocess.PIPE).wait()


def check_environment():
    """Check execution environment."""
    for f in os.listdir(os.getcwd()):
        if f not in WHITELISTED_FILES:
            raise RuntimeError('please execute this script in an empty directory, abort')

    if get_time_delta() > 0.05:
        raise RuntimeError('please sync your network time by installing a NTP client')

    if platform.system() == 'Linux':
        p1 = subprocess.Popen('ps aux | grep -q "[n]tp"', shell=True).wait()
        p2 = subprocess.Popen('ps aux | grep -q "[c]hrony"', shell=True).wait()
        if p1 != 0 and p2 != 0:
            raise RuntimeError('please install ntpd or chrony to synchronize system time')


def start(args, force=False):
    """Start the docker container."""
    p = subprocess.Popen(['docker', 'inspect', '-f', '{{.State.Running}}', CONTAINER_NAME],
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()

    if stdout.strip() == b'true':
        if force:
            print('Stopping old container ...')
            subprocess.Popen(['docker', 'rm', '-f', CONTAINER_NAME]).wait()
        else:
            print('Container already running.')
            return
    elif stdout.strip() == b'false':
        print('Stopping old container ...')
        subprocess.Popen(['docker', 'rm', '-f', CONTAINER_NAME]).wait()

    tangerine_image = get_tangerine_image(args)

    wd = os.getcwd()
    cmd = ['docker', 'run',
           '-d', '--name=%s' % CONTAINER_NAME,
           '-v', '%s:/mnt' % wd,
           '-p', '30303:30303/tcp',
           '-p', '30303:30303/udp',
           '--restart', 'always',
           '-t', tangerine_image,
           '--identity=%s' % args.identity,
           '--bp',
           '--nodekey=/mnt/%s' % args.nodekey,
           '--datadir=/mnt/datadir',
           '--syncmode=fast',
           '--cache=1024',
           '--verbosity=3',
           '--gcmode=archive']

    if args.testnet:
        cmd.append('--testnet')

    print('Starting container ...')
    update_image(tangerine_image)
    subprocess.Popen(cmd, stdout=subprocess.PIPE).wait()

    print('\nContainer running, check logs with `docker logs -f %s\'\n' % CONTAINER_NAME)


def monitor(args):
    """Monitor if there are newer image."""
    tangerine_image = get_tangerine_image(args)
    old_image = get_image_version(tangerine_image)

    while True:
        update_image(tangerine_image)
        new_image = get_image_version(tangerine_image)

        if new_image != old_image:
            sleep_time = random.randint(0, SLEEP_RAND_RANGE)
            print('New image found, sleeping for %s seconds before updating ...' % sleep_time)
            time.sleep(sleep_time)

            # Check update again.
            update_image(tangerine_image)
            new_image = get_image_version(tangerine_image)
            start(args, True)

            old_image = new_image

        time.sleep(POLLING_INTERVAL)


def main():
    """Main."""
    parser = argparse.ArgumentParser(description='Script for launching a Tangerine Node')
    parser.add_argument('--nodekey', default='node.key', dest='nodekey',
                        help='Path to nodekey, default to `node.key\'')
    parser.add_argument('--identity', default=socket.gethostname(),
                        dest='identity', help='Name of the node, e.g. ByzantineLab')
    parser.add_argument('--testnet', default=False, action='store_true', dest='testnet',
                        help='Whether or not to run a testnet node')

    args = parser.parse_args()

    check_environment()

    if not os.path.exists(args.nodekey):
        res = input('No node key found, generate a new one? [y/N] ')
        if res == 'y':
            generate_node_key(args.nodekey)
        else:
            print('Abort.')
            sys.exit(1)

    start(args)
    monitor(args)


if __name__ == '__main__':
    try:
        main()
    except RuntimeError as e:
        print('Error: %s' % str(e))
    except KeyboardInterrupt:
        print('Got interrupt, quitting ...')
