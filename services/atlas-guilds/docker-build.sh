if [[ "$1" = "NO-CACHE" ]]
then
   docker build --no-cache -f Dockerfile.dev --tag ${PWD##*/}:latest .
else
   docker build -f Dockerfile.dev --tag ${PWD##*/}:latest .
fi
