#!/bin/bash

# this script generate a git repository that generate commits of given size

test_updown() {
    mkdir $1
    cd $1
    git init
    # create a file with $2 LOC and commit it
    for i in $(seq 1 $2); do
        echo "line $i" >>file.txt
        git add file.txt
        git commit -m "commit $i"
    done
    # remove the file
    rm file.txt
    git add file.txt
    git commit -m "back to zero"
}

test_updown "up" 100
