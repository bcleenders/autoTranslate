# Call with
# ./spark-submit \
#   --master yarn-cluster \
#   --num-executors 50 \
#   --driver-memory 4g \
#   --executor-memory 1g \
#   --executor-cores 4 \
#   <path to this file>.py
import json
import re
import pickle

from pyspark import SparkContext
from pyspark.mllib.feature import Word2Vec

sc = SparkContext(appName="Word2Vec")

corpus = (sc
        .textFile('hdfs:///word2vec/reddit/200*/RC_*')
        .map(lambda s: re.sub(r'[^A-Za-z0-9\ ]+', '', json.loads(s)['body'].lower()).split(" "))
        )
word2vec = Word2Vec()
model = word2vec.fit(corpus)

save_file = 'hdfs:///word2vec/output_2007_2009.txt'

model.save(sc, save_file)

#with open(save_file, 'w') as f:
#    pickle.dump(model, save_file)