# Lab1
## To server only: 
### For GET: (Dont forget to change port and to send to /files/.)
`curl -v -o cat.jpg http://localhost:PORT/files/cat.jpg`
### For POST:
`curl -v -X POST -F "file=@cat2.jpg" http://localhost:PORT/files`

## To Proxy:
###  For Get: (Dont forget to change ports and to send to /files/. First port is for the server and the second one for the proxy.)
` curl -X GET http://localhost:PORT/files/cat.jpg -x http://localhost:PORT/ -o cat.jpg`
`

## On the cloud:
### Setup:
` 2 Ubuntu based machines (instances) on AWS.
` Download Golang on both machines. 
` Get the zip with: wget https://go.dev/dl/go1.21.4.linux-amd64.tar.gz
` Unzip the folder: sudo tar -xzvf go1.21.4.linux-amd64.tar.gz
` Environment variable: export PATH=$PATH:/usr/local/go/bin
` Git clone our local repository to the machines: git clone https://github.com/PavlosStampoulis/Group7Lab1.git 
` Go to the directory of main.go and respectively proxy.go on the two assigned machines.
` Then in order for the code to run without needing to have the terminal opened, we type this command in the terminal:
`` nohup go run main.go > output.log 2>&1 &
`` and respectively: nohup go run proxy.go > output.log 2>&1 &
` Now both the server and proxy are up and running and any output can be checked with: cat output.log

## Others:
### Test.bh 
Sends 11 get requests to the proxy to test concurrent requests. 
