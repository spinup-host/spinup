#!/bin/bash

usage() {
    echo $0 pg_hba_file_path

    printf 'To replace trust method with md5 for replication connections'
}

if [ "$1" = "-h" ] || [ "$1" = "--help" ]
then 
    usage
    exit 1
else
    filepath="$1"  # the first arg
fi

sed -i 's/^host    replication     all             127.0.0.1\/32.*/host    replication     all             all            md5/g' $filepath
sed -i 's/^host    replication     all             ::1\/128.*/host    replication     all             all                 md5/g' $filepath
