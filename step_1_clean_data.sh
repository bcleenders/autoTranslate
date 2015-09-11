#!/bin/bash

source settings.cfg

time go run *.go \
	-unzipped=$unzipped \
	-zipped=$unzipped \
	-out=$cleaned_data \
	-start=2007 \
	-last=2009
