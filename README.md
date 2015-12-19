# Airbnb Server with Paxos

AirbnbServer is a simple housing search & reservation system that uses Paxos algorithm as its core replication functionality. Users will be able to register, log in, search for housing and then make the reservations by selecting the house(s) they like. In particular, we used Paxos nodes as a distributed backup as well as a mechanism to reach eventual consensus--- especially where there are lots of participating user nodes. 

###### AUTHORS (alphabetical order):
  - Shaojie (Jerry) Bai
  - Zirui (Edward) Wang
  
This project also server as the application layer of CMU 15-440: Distributed System p3.  To see more: [CMU 15-440 p3][p3]

> The goal of this project is to apply the idea
> of paxos algorithm to real life problems. In
> fact, paxos algorithm turns out to be of great use
> in our Airbnb Server because it can ensure
> consistency across the several data storage nodes.
> More importantly, it guarantees a high level 
> fault tolerance (wihout considering Byzantine 
> failure) so that it will be very easy to recover.


### Version
1.1.0


### Technology

The programming languages involved are mainly Go (Golang), Javascript, HTML and CSS, with Bootstrap framework v3.3.5, which can be found at [here](http://getbootstrap.com/). If offers a lot of components that are really useful, such as modals, cool buttons, etc. The interactive part is achieved mainly using Javascript's jQuery library. A typical ajax call serve as the communication "channel" between webview and handlers.

This project is made up of 4 layers in total:
  - The front-end webview. This is the part that directly interacts with the users. Most of the implementations were written in HTML, CSS and Javascript (jQuery).
  - The web controllers/handlers. These handlers are much like the controllers in the MVC model that directly deal with the view. However, they don't do any processing tasks. When the user perform certain kind of operations, it is then the job of the handlers to parse the request, serve as a proxy, and ask the back-end system for responses. Once it gets the result, it will render the view and display to the users.
  - The AirbnbNode. This is a *para-centralized* part of our system. A cluster of the web views connect to the same AirbnbNode. In addition, each AirbnbNode is connected to all the Paxos nodes. It serves in between the handlers and the Paxos nodes as a layer of indirection--- this offers a better interface usage and masks the Paxos calls from the front-end development. Once the AirbnbNode collects the information from handler(s), they start proposing to Paxos.
  - Paxos algorithm (nodes). Paxos nodes are responsible for the distributed backup. The AirbnbNode, once contacted and ready to Propose, will contact an arbitrary available Paxos node by first calling `GetNextProposalNumber()`. Then it will wrap the (key, value) pair in ar argument and RPC `Propose()` that each Paxos node supports. Once there is a success, Paxos node will pass the result back using RPC to the AirbnbNode (who will forward the reply upward in hierarchy).

### Installation

You don't necessarily need to have Golang installed on your computer (though we recommend it). In **/p3/bin** folder and **/p3/src/github.com/cmu440-F15/paxosapp/airbnbweb** folders, we already had `.exe` file prepared for you. 

You will first need to download our project from GitHub. Then, put it in your local directory where you put your Go, and set your `GOPATH` to this folder by

```sh
$ export GOPATH=$'your_path_of_download\p3'
```
(For example, if you downloaded to your local Windows directory "C:\Go", then do `export GOPATH=$'C:\Go\p3'`.

If you want to compile from scratch, then do the following:

```sh
$ cd $GOPATH/src
$ go install github.com/cmu440-F15/paxosapp/runners/arunner
...
$ cd $GOPATH/bin
$ ./arunner
```

After the steps above, you should have already had a bunch of Paxos nodes and AirbnbNodes running. Then you may want to open another terminal (or you can run `./arunner &` to run in background), reset your `GOPATH` in a similar fashion, and do

```sh
$ cd $GOPATH/src/github.com/cmu440-F15/paxosapp/airbnbweb
$ go build airbnbWebsite.go
$ ./airbnbWebsite -=port=8080
```

In the last step, if you do not specify your own port, it will default to port **":8080"**.

Finally, you can open your web browser and type in [localhost:8080/login/](localhost:8080/login/). From there, you will have access to the system!

### Todos

 - UI (better interactions, more funcitonalities)
 - Testing
 - Lots of other things :)


[//]: # (These are reference links used in the body of this note and get stripped out when the markdown processor does its job. There is no need to format nicely because it shouldn't be seen. Thanks SO - http://stackoverflow.com/questions/4823468/store-comments-in-markdown-syntax)


   [p3]: <https://github.com/cmu440-F15/p3>
  


