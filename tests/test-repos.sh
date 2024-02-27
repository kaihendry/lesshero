#!/bin/bash

# this script generate a git repository that generate commits of given size

test_updown() {
    pushd .
    mkdir $1
    cd $1
    git init
    # create a file with $2 LOC and coammit it
    for i in $(seq 1 $2); do
        echo "line $i" >>file.txt
        git add file.txt
        git commit -m "commit $i"
    done
    # remove the file
    rm file.txt
    git add file.txt
    git commit -m "back to zero"
    popd
}

test_upmovedown() {
    pushd .
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
    # create a file with $2 LOC and commit it
    for i in $(seq 1 $2); do
        echo "line $i" >>file.txt
        git add file.txt
        git commit -m "commit $i"
    done
    git mv file.txt file2.txt
    git add file2.txt
    git commit -m "moved"
    # remove the file2.txt
    rm file2.txt
    git add file2.txt
    git commit -m "back to zero"
    popd
}

test_updown "up" 100
test_upmovedown "m" 100
