if [[ "$1" = "NO-CACHE" ]]
then
   docker build --no-cache --tag atlas-login:latest .
else
   docker build --tag atlas-login:latest .
fi
