# run_bp.py

This script is used for running Tangerine Network BP node. The script itself
supports automatic upgrade (AU), as well as automatically upgrading the node
docker image.

## Updating

The commit updating the script must be signed whenever `run_bp.py` is changed. Check the [GitHub Help](https://help.github.com/en/articles/signing-commits) to set it up.

After committing the file, several approvers(check `_SCRIPT_APPROVER` and `_SCRIPT_APPROVE_THRESHOLD` in the script) then update their `run_bp.py.APPROVER_GITHUB_ID` with the commit hash of the `run_bp.py`. Note that the approver's commit must be signed to be considered a valid approval.
