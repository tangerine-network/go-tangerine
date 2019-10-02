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
import urllib.request


try:
    import ntplib
except Exception:
    print('Please run `pip3 install ntplib\'')
    sys.exit(1)


_SCRIPT_SRC = ('https://raw.githubusercontent.com/'
               'tangerine-network/go-tangerine/master/scripts/run_bp.py')

_REQUEST_TIMEOUT = 5
_CONTAINER_NAME_BASE = 'tangerine'
_NUM_SLOTS = 5
_POLLING_INTERVAL = 60
_SLEEP_RAND_RANGE = 1800
_TANGERINE_IMAGE_TMPL = 'byzantinelab/go-tangerine:latest-%s-%d'
_TOOLS_IMAGE = 'byzantinelab/tangerine-tools'
_WHITELISTED_FILES = [
    os.path.basename(sys.argv[0]),
    'datadir',
    'node.key'
]

# Current executing script sha1sum
sha1sum = None


def get_container_name(testnet):
    return (_CONTAINER_NAME_BASE
            if not testnet else _CONTAINER_NAME_BASE + '-testnet')


def get_shard_id(nodekey):
    """Return shard ID.

    Shard ID is calculate as (first byte of sha3 of Node Key address) mode
    _NUM_SLOTS.
    """
    wd = os.getcwd()
    p = subprocess.Popen(['docker', 'run', '-v', '%s:/mnt' % wd, '--rm',
                          '-t', _TOOLS_IMAGE,
                          'nodekey', 'inspect', '/mnt/%s' % nodekey],
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()

    address = re.search('^Node Address: (0x[0-9a-fA-F]{40})',
                        stdout.decode('utf-8')).group(1)
    return int(hashlib.sha256(
        address.encode('utf-8')).hexdigest()[0], base=16) % _NUM_SLOTS


def get_tangerine_image(args):
    """Return the tangerine image by shard-ID."""
    return _TANGERINE_IMAGE_TMPL % ('testnet' if args.testnet else 'mainnet',
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
    subprocess.Popen(['docker', 'run', '-v', '%s:/mnt' % wd, '--rm',
                      '-t', _TOOLS_IMAGE,
                      'nodekey', 'generate', '/mnt/%s' % nodekey]).wait()

    print('Node key generated')
    print('\n\033[5;91;49mPlease backup node.key in a secure place!\033[0m')
    print('\n\033[0;91mSend at least 1 ETH (use Rinkeby ETH for testnet) to '
          'node key address!\033[0m')
    print('\033[0;91mThese ETHs are used for then Tangerine network recovery '
          'mechanism.\033[0m\n\n')


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
        if f not in _WHITELISTED_FILES:
            raise RuntimeError('please execute this script in an empty '
                               'directory, abort')

    for i in range(5):
        try:
            if get_time_delta() > 0.05:
                raise RuntimeError('please sync your network time by '
                                   'installing a NTP client')
        except ntplib.NTPException as e:
            print('Error: %s' % e)
            print('Retrying in 10 seconds ...')
            time.sleep(10)
        else:
            break

    if platform.system() == 'Linux':
        p1 = subprocess.Popen('ps aux | grep -q "[n]tp"', shell=True).wait()
        p2 = subprocess.Popen('ps aux | grep -q "[c]hrony"', shell=True).wait()
        if p1 != 0 and p2 != 0:
            raise RuntimeError('please install ntpd or chrony to synchronize '
                               'system time')


def check_for_update():
    """Check for script update."""
    script_path = os.path.abspath(sys.argv[0])
    global sha1sum

    if sha1sum is None:
        with open(script_path, 'r') as f:
            sha1sum = hashlib.sha1(f.read().encode('utf-8')).hexdigest()

    with urllib.request.urlopen(_SCRIPT_SRC + '.sha1',
                                timeout=_REQUEST_TIMEOUT) as f:
        if f.getcode() != 200:
            raise RuntimeError('unable to get upgrade metadata')
        update_sha1sum = f.read().strip().decode('utf-8')

    if sha1sum != update_sha1sum:
        print('Script upgrade found, performing upgrade ...')
    else:
        return

    with urllib.request.urlopen(_SCRIPT_SRC, timeout=_REQUEST_TIMEOUT) as f:
        script_data = f.read()
        new_sha1sum = hashlib.sha1(script_data).hexdigest()

    if new_sha1sum != update_sha1sum:
        raise RuntimeError('failed to verify upgrade payload, aborted')

    with open(script_path, 'w') as f:
        f.write(script_data.decode('utf-8'))

    os.execve(script_path,
              [script_path] + sys.argv[1:] + ['--skip-env-check'], os.environ)


def start(args, force=False):
    """Start the docker container."""
    container_name = get_container_name(args.testnet)

    p = subprocess.Popen(['docker', 'inspect', '-f', '{{.State.Running}}',
                          container_name],
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()

    if stdout.strip() == b'true':
        if force:
            print('Stopping old container ...')
            subprocess.Popen(['docker', 'rm', '-f', container_name]).wait()
        else:
            print('Container already running.')
            return
    elif stdout.strip() == b'false':
        print('Stopping old container ...')
        subprocess.Popen(['docker', 'rm', '-f', container_name]).wait()

    tangerine_image = get_tangerine_image(args)

    port = 30303 if not args.testnet else 40303

    wd = os.getcwd()
    cmd = ['docker', 'run',
           '-d', '--name=%s' % container_name,
           '-v', '%s:/mnt' % wd,
           '-p', '%d:%d/tcp' % (port, port),
           '-p', '%d:%d/udp' % (port, port),
           '--restart', 'always',
           '-t', tangerine_image,
           '--identity=%s' % args.identity,
           '--bp',
           '--nodekey=/mnt/%s' % args.nodekey,
           '--datadir=/mnt/datadir',
           '--syncmode=fast',
           '--cache=1024',
           '--verbosity=%s' % args.verbosity,
           '--gcmode=archive',
           '--port=%d' % port]

    if args.testnet:
        cmd.append('--testnet')

    print('Starting container ...')
    update_image(tangerine_image)
    subprocess.Popen(cmd, stdout=subprocess.PIPE).wait()

    print('\nContainer running, check logs with `docker logs -f %s\'\n' %
          container_name)


def monitor(args):
    """Monitor if there are newer image."""
    tangerine_image = get_tangerine_image(args)
    old_image = get_image_version(tangerine_image)

    while True:
        update_image(tangerine_image)
        new_image = get_image_version(tangerine_image)

        if new_image != old_image:
            sleep_time = random.randint(0, _SLEEP_RAND_RANGE)
            print('New image found, sleeping for %s seconds before '
                  'updating ...' % sleep_time)
            time.sleep(sleep_time)

            # Check update again.
            update_image(tangerine_image)
            new_image = get_image_version(tangerine_image)
            start(args, True)

            old_image = new_image

        time.sleep(_POLLING_INTERVAL)

        try:
            check_for_update()
        except Exception as e:
            print('Error: %s' % e)


def main():
    """Main."""
    parser = argparse.ArgumentParser(
        description='Script for launching a Tangerine Node')
    parser.add_argument('--nodekey', default='node.key', dest='nodekey',
                        help='Path to nodekey, default to `node.key\'')
    parser.add_argument('--identity', default=None, dest='identity',
                        help='Name of the node, e.g. ByzantineLab')
    parser.add_argument('--testnet', default=False, action='store_true',
                        dest='testnet',
                        help='Whether or not to run a testnet node')
    parser.add_argument('--force', default=False, action='store_true',
                        dest='force',
                        help='Force restart a node')
    parser.add_argument('--verbosity', default=3, dest='verbosity',
                        help='Verbosity level')
    parser.add_argument('--skip-env-check', default=False,
                        dest='skip_env_check', action='store_true',
                        help='Skip environment check, should only be '
                             'used for AU mechanism')

    args = parser.parse_args()

    if not args.skip_env_check:
        check_environment()

    if args.identity is None:
        raise RuntimeError("please specify identity (node name)")

    if not os.path.exists(args.nodekey):
        res = input('No node key found, generate a new one? [y/N] ')
        if res == 'y':
            generate_node_key(args.nodekey)
        else:
            print('Abort.')
            sys.exit(1)

    start(args, args.force)

    pid = os.fork()
    if pid == 0:
        monitor(args)
    else:
        print('Daemon process running ...')


if __name__ == '__main__':
    try:
        main()
    except RuntimeError as e:
        print('Error: %s' % str(e))
    except KeyboardInterrupt:
        print('Got interrupt, quitting ...')
