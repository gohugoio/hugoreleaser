#!/bin/bash

trap exit SIGINT

# I use "run tests on save" in my editor.
# Unfortantly, changes to text files does not trigger this. Hence this workaround.
while true; do find . -type f -name "*.txt" -o -type f -name "*.yaml" | entr -pd touch main_test.go; done