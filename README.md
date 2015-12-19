# Airbnb Server with Paxos

AirbnbServer is a simple housing search & reservation system that uses Paxos algorithm as its core replication functionality. In particular, each registered user will have an id, and, by inputting on the command line, they will be
able to make reservations in the system. 

###### AUTHORS (alphabetical order):
  - Shaojie (Jerry) Bai
  - Zirui (Edward) Wang
  
This project also server as the application layer of CMU 15-440: Distributed System p3.  To see more: [CMU 15-440 p3][p3]

> The goal of this project is to apply the idea
> of paxos algorithm to real life problems. In
> particular, paxos algorithm will be of great use
> in our Airbnb Server because it can ensure
> consistency across the several data storage nodes.
> More importantly, it guarantees a high level 
> fault tolerance (wihout considering Byzantine 
> failure) so that it will be very easy to recover.


### Version
1.0.0

### Installation

First, you need to put the <kbd>/airbnb</kbd> folder to your local Go source folder. Otherwise, it will not compile.

You will need to run the following commands in order to really get output:

```sh
$ cd /airbnb/src
$ go install trial
```
Then you can run the program by running the main function:
```sh
$ cd /trial
$ go run main.go
```

### Todos

 - UI (preferably on web platform)
 - Testing
 - Lots of other things :)


[//]: # (These are reference links used in the body of this note and get stripped out when the markdown processor does its job. There is no need to format nicely because it shouldn't be seen. Thanks SO - http://stackoverflow.com/questions/4823468/store-comments-in-markdown-syntax)


   [p3]: <https://github.com/cmu440-F15/p3>
  


