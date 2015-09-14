#!/bin/bash# Train word2vec

source settings.cfg

cd ./word2vec

make

./word2vec \
	-train $concatenated_data \
	-output $vectors \
	-cbow 1 \
	-size 200 \
	-window 10 \
	-negative 10 \
	-hs 0 \
	-sample 1e-5 \
	-threads 40 \
	-binary 1 \
	-iter 3 \
	-min \
	-count 50