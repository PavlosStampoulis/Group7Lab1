# Lab1
## To server only: 
### For GET: (Dont forget to change port and to send to /files/.)
`curl -v -o cat.jpg http://localhost:PORT/files/cat.jpg`
### For POST:
`curl -v -X POST -F "file=@cat2.jpg" http://localhost:PORT/files`

## To Proxy:
###  For Get: (Dont forget to change ports and to send to /files/.)
` curl -X GET http://localhost:PORT/files/cat.jpg -x http://localhost:PORT/ -o cat.jpg`
`

## Others:
### Test.bh 
Sends 11 get requests to the proxy to test concurrent requests. 
