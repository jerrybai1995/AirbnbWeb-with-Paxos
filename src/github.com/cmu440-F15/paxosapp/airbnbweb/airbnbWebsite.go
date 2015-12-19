package main

import (
	"flag"
	"fmt"
	"github.com/cmu440-F15/paxosapp/rpc/airbnbrpc"
	"html/template"
	"log"
	"net/http"
	"net/rpc"
	"regexp"
	"strconv"
	"time"
)

type Page struct {
	Title   string
	Body    []byte
	Hidden1 string // A hidden part prepared for hidden fields in html.
	Hidden2 string
	State   string // Potential alert message
}

var airbnbCli *rpc.Client

// A flag for user to specify the port of the website.
var (
	portFlag = flag.String("port", "8080", "port for website")
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

/* This function takes a username and password and tries to register this new user.
 * It returns true if the registration succeeds, and false otherwise.
 */
func registerUser(username, password string) bool {
	registerArgs := &airbnbrpc.RegisterUserArgs{username, password}
	var registerReply airbnbrpc.RegisterUserReply
	// Make the RPC call
	err := airbnbCli.Call("AirbnbNode.RegisterUser", registerArgs, &registerReply)
	if err != nil {
		fmt.Println("registerUser rpc call error...")
	}
	if registerReply.Status == airbnbrpc.OK {
		return true
	} else {
		return false
	}
}

/* Input a username and password, check whether this combination is a valid (correct)
 * one. Return a boolean, representing validity, and a string, which is a unique id
 * assigned to this use. We add this id to form a new url eventually.
 */
func checkValidity(username, password string) (bool, string) {
	checkArgs := &airbnbrpc.RegisterUserArgs{username, password}
	var checkReply airbnbrpc.CheckValidityReply
	err := airbnbCli.Call("AirbnbNode.CheckValidity", checkArgs, &checkReply)
	if err != nil {
		fmt.Println("checkValidity rpc call error...")
	}
	return checkReply.Valid, checkReply.HUsername
}

/* This function, give the information about the new house we want to add, uses the
 * RPC call to register this new housing information in the paxos nodes system. Once
 * the addition is a success, the function returns true, and false otherwise.
 */
func addNewHouse(owner, addr, city, zip, rating string, downtown bool, price string) bool {
	addHouseArgs := &airbnbrpc.AddNewHouseArgs{owner, addr, city, zip, rating, downtown, price}
	var addHouseReply airbnbrpc.RegisterUserReply
	err := airbnbCli.Call("AirbnbNode.AddNewHouse", addHouseArgs, &addHouseReply)
	if err != nil {
		fmt.Println("addNewHouse rpc call error...")
	}
	if addHouseReply.Status == airbnbrpc.OK {
		return true
	} else {
		return false
	}
}

/* Given a userid and a houseid as the input, this function represents the user's
 * action that this user wants to select this house. Whether the operation is
 * successful will be determined by the paxos algorithm. The function returns a
 * booleans value denoting the success of the op.
 */
func selectHousing(userid string, houseid int) bool {
	selectHousingArgs := &airbnbrpc.SelectHousingArgs{userid, houseid}
	var selectHousingReply airbnbrpc.RegisterUserReply
	err := airbnbCli.Call("AirbnbNode.SelectHousing", selectHousingArgs, &selectHousingReply)
	if err != nil {
		fmt.Println("selectHousing rpc call error...")
	}
	if selectHousingReply.Status == airbnbrpc.OK {
		return true
	} else {
		return false
	}
}

/* Given a userid, fetch the current reservatoins that he/she has made so far. This
 * function returns the slice of airbnbrpc.HousingInfo that the user has reserved.
 */
func fetchCurrentReservations(userid string) []airbnbrpc.HousingInfo {
	fetchArgs := &airbnbrpc.FetchArgs{userid}
	var fetchReply airbnbrpc.FetchReply
	err := airbnbCli.Call("AirbnbNode.FetchCurrentReservations", fetchArgs, &fetchReply)
	if err != nil {
		fmt.Println("fetchCurrentReservations rpc call error...")
	}
	return fetchReply.Rst
}

/* Input a metric, this function returns the list of all potential locations matching
 * it. For instance, with metric "New York", it should return a slice of all
 * unreserved locations in New York in our system.
 */
func searchHousingBy(metric string) []airbnbrpc.HousingInfo {
	searchArgs := &airbnbrpc.SearchArgs{metric}
	var searchReply airbnbrpc.FetchReply
	err := airbnbCli.Call("AirbnbNode.SearchHousingBy", searchArgs, &searchReply)
	if err != nil {
		fmt.Println("searchHousingBy rpc call error...")
	}
	return searchReply.Rst
}

/* This function handles the login url and behavior of the users. It will call the
 * functions such as checkValidity() so as to ascertain whether the user trying to
 * log in is a legitimate one. It renders the html directly.
 */
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the relevant information from the html form submission.
		username := r.FormValue("user")
		password := r.FormValue("pass")
		valid, id := checkValidity(username, password)
		if valid {
			// Then we enter the system.
			http.Redirect(w, r, "/reservation/"+id, http.StatusFound)
			return
		} else {
			/* We send out an alert if the combination is not correct. */
			p := &Page{Title: "Login Airbnb", State: "Incorrect combination. Please try again."}
			if username == "" {
				p = &Page{Title: "Login Airbnb"}
			}
			renderTemplate(w, "login", p)
		}
	} else {
		/* Usual GET method call on the login page */
		p := &Page{Title: "Login Airbnb"}
		renderTemplate(w, "login", p)
	}

}

/* This handler is used when the user of the system is trying to register a new user.
 * It has the same view as the login page, but it sends out registratoin alert (State)
 * when there is an error during registration.
 */
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		/* Get the username and password that the user wants to register, and then
		 * use RPC to register.
		 */
		username := r.FormValue("user")
		password := r.FormValue("pass")
		result := registerUser(username, password)
		if result {
			/* If valid, we then go back to the normal login stage, where the user has
			 * to log in again using his/her new username & password.
			 */
			http.Redirect(w, r, "/login/", http.StatusFound)
		} else {
			p := &Page{Title: "Login Airbnb", State: "Registration not successful."}
			renderTemplate(w, "login", p)
		}
	}
}

/* This function (handler) is designed mainly to handle the automatic ajax call from
 * javascript that requires the server (paxos) to provide the reservations that the
 * user currently owns. When we finish the RPC call, we simply write to w, which
 * will be reflected in the ajax success function.
 */
func getReservationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("userid")
		curReservations := fetchCurrentReservations(id)
		htmlReservations := packReservedLocations(curReservations)
		w.Write([]byte(htmlReservations))
	}
}

/* The reservation page handler. Simply render the template. */
func reservationHandler(w http.ResponseWriter, r *http.Request, id string) {
	p := &Page{
		Title:   "Your Reservation",
		Hidden1: id,
	}
	renderTemplate(w, "reservation", p)
}

/* The html templates in the project directory. */
var templates = template.Must(template.ParseFiles("login.html", "reservation.html"))

/* Use the template html code and execute this html file */
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/* A regexp checking that the url is correct. */
var validPath = regexp.MustCompile("^/(login|reservation)/([a-zA-Z0-9]+)$")

/* Wrap a "higher-level" handler (which may have a title) to a standard handler (which
 * only has http.ReponseWriter and *http.Request as the arguments) so that we can
 * match the urls to the handlers we want to assign to them (see main function).
 */
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

/* This function is called as a result of the javascript ajax call. In particular,
 * when the ajax call submits its form to add a new housing, this function parses
 * the json request and passes the arguments to the RPC. Then, when it hears back
 * the result, it flushes the result back to the ajax in webview.
 */
func addHouseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		/* Parse the json passed from the javascript ajax call */
		owner := r.FormValue("owner")
		addr := r.FormValue("addr")
		city := r.FormValue("city")
		zip := r.FormValue("zip")
		rating := r.FormValue("rating")
		downtown := r.FormValue("downtown") == "Yes"
		price := r.FormValue("price")
		// Make RPC call
		result := addNewHouse(owner, addr, city, zip, rating, downtown, price)
		if result == true {
			w.Write([]byte("Success!"))
		} else {
			w.Write([]byte("Failure!"))
		}
	}
}

/* This function is called by the javascript as a result of user interaction. We
 * search the housing using the metric and then pack these locations into html code
 * that will be appended to reservation.html.
 */
func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		metric := r.FormValue("metric")
		time.Sleep(1 * time.Second)
		alt := searchHousingBy(metric)
		w.Write([]byte(packLocations(alt)))
	}
}

/* When the user selects a specific housing location he/she wishes to have, this
 * handler is responsible for passing the message between the web view and the
 * RPC call. The returned boolean represents whether the selction is a success or not.
 */
func selectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		userId := r.FormValue("user")
		housingId, _ := strconv.Atoi(r.FormValue("houseid"))
		result := selectHousing(userId, housingId)
		if result == true {
			w.Write([]byte("true"))
		} else {
			w.Write([]byte("false"))
		}
	}
}

/* Return a whole bunch of html codes for the slice of locations passed in. */
func packLocations(allLocations []airbnbrpc.HousingInfo) string {
	ans := ""
	for _, loc := range allLocations {
		ans += packSingleLocation(loc)
	}
	return ans
}

/* Return a whole bunch of html codes for the slice of reserved locations passed in. */
func packReservedLocations(allLocations []airbnbrpc.HousingInfo) string {
	ans := ""
	for _, loc := range allLocations {
		ans += packSingleReservedLocation(loc)
	}
	return ans
}

/* We pack each location into a html panel. This html code is then returned as a
 * string.
 */
func packSingleLocation(info airbnbrpc.HousingInfo) string {
	ans := "<div class='bs-callout bs-callout-success'>"
	ans += fmt.Sprintf("<span id='housingId' style='display: none'>%d</span>", info.Id)
	ans += fmt.Sprintf("<h4>%s, %s %s</h4>", info.Address, info.City, info.Zip)
	ans += fmt.Sprintf("<p>Host: <span id='hostInfo'>%s</span></p>", info.Owner)
	ans += fmt.Sprintf("<p>Rating: <span id='ratingInfo'>%s</span></p>", info.Rating)
	tag := `<div style='text-align: right'>
<span class='label label-warning' id='downtownInfo'>Downtown</span>
  </div>`
	if info.Downtown != true {
		tag = `<div style='text-align: right'>
<span class='label label-warning' id='downtownInfo'></span>
  </div>`
	}
	selection := `<button class="btn btn-default" id="moreInfo" style="margin-right: 10px" data-toggle="modal" data-target="#gmapModal">More info</button>
<button class="btn btn-danger" id="selectThis">Select this</button>`
	ans += fmt.Sprintf("<div style='text-align: center'>%s %s</div>", selection, tag)
	ans += "</div>"
	return ans
}

/* For reserved locations, we have a different panel (css) for it. Html code returned */
func packSingleReservedLocation(info airbnbrpc.HousingInfo) string {
	ans := fmt.Sprintf("<div class='panel panel-warning' id=%d> <div class='panel-heading'>", info.Id)
	ans += fmt.Sprintf("<h3 class='panel-title'>%s, %s %s</h3> </div>", info.Address, info.City, info.Zip)
	ans += fmt.Sprintf("<div class='panel-body'>Your host: <strong id='yourHost'>%s</strong>", info.Owner)
	if info.Downtown {
		ans += "<br /><br />"
		ans += "<div style='text-align: right'><span class='label label-warning' id='downtownInfo'>Downtown</span></div></div> </div>"
	}
	return ans
}

func main() {
	var err error
	flag.Parse()
	myPort := *portFlag
	airbnbCli, err = rpc.DialHTTP("tcp", "localhost:9999")
	if err != nil {
		fmt.Println("Fatal error: CANNOT CONNECT TO arunner.")
		return
	}

	/* For demo purpose, we register one user and 4 housing locations */
	registerUser("shaojieb", "123123")
	addNewHouse("Mark Hans", "5032 Forbes Ave.", "Pittsburgh", "15213", "4.7", true, "2000")
	addNewHouse("John Doe", "944 Tiverton Ave # 11", "Los Angeles", "90024", "4.9", false, "1000")
	addNewHouse("Ellie Mankins", "901 S Broadway", "Los Angeles", "90015", "4.8", true, "5000")
	addNewHouse("Ellen Nowry", "Room 661 102 Broad St.", "New ork", "10004", "4.6", true, "800")

	// Register the handlers that will be called when certain POST/GET methods are used
	http.HandleFunc("/login/", loginHandler) // GET
	http.HandleFunc("/login", loginHandler)  // POST
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/select", selectHandler)
	http.HandleFunc("/addHouse", addHouseHandler)
	http.HandleFunc("/reservation/", makeHandler(reservationHandler))
	http.HandleFunc("/getReservation", getReservationHandler)

	// Listen at the port specified.
	http.ListenAndServe(fmt.Sprintf(":%s", myPort), nil)
}
