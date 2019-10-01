# run_bp.py

This script is used for running Tangerine Network BP node. The script itself
supports automatic upgrade (AU), as well as automatically upgrading the node
docker image.

## Updating

Whenever `run_bp.py` is changed, `run_bp.py.sha1` needs to be updated
correspondingly with:

    sha1sum run_bp.py | awk '{ print $1 }' > run_bp.py.sha1
