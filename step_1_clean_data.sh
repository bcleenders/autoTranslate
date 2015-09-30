#!/bin/bash

source settings.cfg

cd gocode

time go run *.go \
	-unzipped=$unzipped \
	-zipped=$zipped \
	-out=$cleaned_data \
	-start=2007 \
	-last=2009
