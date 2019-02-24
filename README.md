# Bitmark node with nodemap service

Forked from https://github.com/bitmark-inc/bitmark-node/, reference it for more usages.

* This project integrates bitmark-node to nodemap service https://github.com/zodahu/nodemap.
  * depends on bitmarkd to collect peers information and sends to node map server
  * requires [MAP_SERVER_IP_PORT] when docker starts, reference to https://github.com/zodahu/nodemap 

## Installation

```
docker pull zodahu/bitmark-node
```

## Run
```
sudo docker run -d --name bitmarkNode -p 9980:9980 \
-p 2136:2136 -p 2130:2130 \
-e MAP_IP_PORT=[MAP_SERVER_IP_PORT] \
-e PUBLIC_IP=[SERVER_IP] \
-v $HOME/bitmark-node-data/db:/.config/bitmark-node/db \
-v $HOME/bitmark-node-data/data:/.config/bitmark-node/bitmarkd/bitmark/data \
-v $HOME/bitmark-node-data/data-test:/.config/bitmark-node/bitmarkd/testing/data \
zodahu/bitmark-node
```

## Send peers information to nodemap service
Note that start Bitmark Node (bitmarkd) first
![nodemap_button](https://github.com/zodahu/data/blob/master/data/nodemap_button.png)


