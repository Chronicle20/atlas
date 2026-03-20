SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

if [[ "$1" = "NO-CACHE" ]]
then
   docker build --no-cache --tag atlas-login:latest -f "$SCRIPT_DIR/Dockerfile" "$REPO_ROOT"
else
   docker build --tag atlas-login:latest -f "$SCRIPT_DIR/Dockerfile" "$REPO_ROOT"
fi
