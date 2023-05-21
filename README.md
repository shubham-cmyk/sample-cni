# CNI

It is a sample CNI plugin which helps to manage the networking of kubernetes

Aim - Create a CNI plugin that could help 2 or more container to talk to eachother with the bridge driver. 

Steps to create : 

1. Create a veth eth pair peer 
2. Start vethHost and veth i.e. ip link up 
3. Create a bridge i.e. subnet mask and gateway to be assigned
4. Connect the vethHost to the bridge 
5. Make a route entry to the container veth to forward all the default traffic to the gateway
