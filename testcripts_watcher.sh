#!/bin/bash

# I use "run tests on save" in my editor.
# Unfortantly, changes to text files does not trigger this. Hence this workaround.
while true; do find testscripts internal -type f -name "*.txt" -o -type f -name "*.toml" | entr -pd touch main_test.go; sleep 1; done