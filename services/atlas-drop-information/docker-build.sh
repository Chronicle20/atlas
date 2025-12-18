if [[ "$1" = "NO-CACHE" ]]
then
   docker build --no-cache --tag atlas-dis:latest .
else
   docker build --tag atlas-dis:latest .
fi
