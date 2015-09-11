#!/bin/bash# Train word2vec

source settings.cfg

cd ./word2vec

make

./word2vec \
	-train $cleaned_data/* \
	-output vectors.bin \
	-cbow 1 \
	-size 500 \
	-window 10 \
	-negative 10 \
	-hs 0 \
	-sample 1e-5 \
	-threads 40 \
	-binary 1 \
	-iter 3 \
	-min \
	-count 10