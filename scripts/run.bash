#!/bin/bash

SIMPLEM2M=$(command -v simplem2m)

log()
{
    echo "$(date +"%F %T"): $*"
}

usage()
{
    echo -e "$(basename "$0") [-h] [-c <config.toml>] [-d <dbaddr>] [-p <port>]-- program to start the simplem2m service

    where:
        -h show this help text
        -c configuration file location; default is '${HOME}/.simpleam2m/config.toml'
        -d set authentication database address; default is 'mongodb://127.0.0.1:27017'
        -p set listening port of the HTTP service; default is '8080'"
        exit
}

# START #

conf="${HOME}/.simplem2m/config.toml"
port="8080"
db_addr="mongodb://127.0.0.1:27017"
encryption_key="${HOME}/.simplem2m/secret.key"
key="${HOME}/.simplem2m/key.pem"
cert="${HOME}/.simplem2m/cert.pem"

while getopts "hc:p:d:" opt; do
    case "$opt" in
    [h?]) usage
        ;;
    c) conf="${OPTARG}"
        ;;
    d) db_addr="${OPTARG}"
        ;;
    p) port="${OPTARG}"
        ;;
    esac
done

conf_dir=$(dirname ${conf})
if [ ! -d ${conf_dir} ]; then
    log $(mkdir -vp ${conf_dir})
fi

cat <<EOF > ${conf}
    host="0.0.0.0"
    port=${port}
    read_timeout=30
    write_timeout=30

    database_addr="${db_addr}"
    encryption_key="${encryption_key}"
    private_key="${key}"
    certificate="${cert}"
EOF
log "created '${conf}':
$(cat ${conf} | sed 's/^/\t/')"

$SIMPLEM2M --config-file="${conf}"
