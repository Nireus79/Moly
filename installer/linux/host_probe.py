#!/usr/bin/env python3
"""Plays the part of the browser: starts the installed Moly program as a native messaging host and asks for the backend.
Usage: host_probe.py <moly binary> <extension origin>   Prints one JSON line per reply; exit 0 if start succeeded."""
import json, struct, subprocess, sys

exe, origin = sys.argv[1], sys.argv[2]
p = subprocess.Popen([exe, origin], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

def send(m):
    d = json.dumps(m).encode()
    p.stdin.write(struct.pack('<I', len(d)) + d)
    p.stdin.flush()

def recv():
    h = p.stdout.read(4)
    if len(h) < 4:
        return None
    return json.loads(p.stdout.read(struct.unpack('<I', h)[0]))

ok = False
for action in ('ping', 'start', 'status'):
    send({'action': action, 'id': action})
    r = recv()
    if r and 'token' in r:
        r['token'] = r['token'][:6] + '...'
    print(json.dumps(r), flush=True)
    if action == 'start':
        ok = bool(r and r.get('success'))
p.stdin.close()
p.wait(timeout=15)
sys.exit(0 if ok else 1)
