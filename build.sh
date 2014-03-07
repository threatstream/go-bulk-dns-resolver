#!/usr/bin/env bash

##
# @author Jay Taylor [@jtaylor]
#
# @date 2014-03-06
#

cd "$(dirname "$0")"

if test "$1" == '-u' || test "$1" == '--update'; then
    forceUpdate=1
    echo 'info: dependency update will be forced'
fi


echo 'info: fetching dependencies'
# This finds all lines between:
# import (
#     ...
# )
# and appropriately filters the list down to the projects dependencies.  It also ignores any lines which start with "//", as those are comments.
dependencies=$(find . -wholename '*.go' -exec awk '{ if ($1 ~ /^import/ && $2 ~ /[(]/) { s=1; next; } if ($1 ~ /[)]/) { s=0; } if (s) print; }' {} \; | grep -v '^[^\.]*$' | tr -d '\t' | tr -d '"' | sed 's/^\. \{1,\}//g' | sort | uniq | grep -v '^[ \t]*\/\/' | sed 's/_ //g')
for dependency in $dependencies; do
    echo "info:     retrieving: ${dependency}"
    if test -n "${forceUpdate}" || ! test -d "${GOPATH}/src/${dependency}"; then
        go get -u $dependency
        rc=$?
        test $rc -ne 0 && echo "error: retrieving dependency ${dependency} exited with non-zero status code ${rc}" && exit $rc
    else
        echo 'info:         -> already exists, skipping'
    fi
done


echo 'info: compiling project'

go build -o bulkdns main.go utils.go answer.go

buildResult=$?

if test $buildResult -eq 0; then
    echo 'info:     build succeeded - the binary is located at ./bulkdns'
else
    echo "error:    build failed, exited with status ${buildResult}" 1>&2
fi

exit $buildResult

