package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var areAppointments = regexp.MustCompile(`you will have 10 minutes to complete your payment`)
var loggedOut = regexp.MustCompile(`You have not used this service for 25 minutes so you need to sign in again`)
var noAppointments = regexp.MustCompile(`We do not have any appointments in the next 45 business days at your selected location.`)
var longFormIDs = make(map[string]string)

func main() {
	longFormIDs["PCCR"] = "Croydon (England)"
	// longFormIDs["PEBE"] = "Belfast (Ireland)"
	longFormIDs["PEBI"] = "Birmingham (England)"
	// longFormIDs["PECA"] = "Cardiff (Wales)"
	// longFormIDs["PEGL"] = "Glasgow (Scotland)"
	// longFormIDs["PELI"] = "Liverpool (England)"
	// longFormIDs["PESH"] = "Sheffield (England)"

	doEvery(1*time.Second, run)
}

func doEvery(d time.Duration, f func()) {
	for range time.Tick(d) {
		f()
	}
}

func run() {
	ch := make(chan string)

	for centreID := range longFormIDs {
		go fetch(centreID, ch)
	}
	for range longFormIDs {
		fmt.Println(<-ch)
	}
}

func fetch(centreID string, ch chan<- string) {
	client := &http.Client{}
	form := url.Values{}
	form.Set("csrfToken", "5de593886ff58ac902debf4f533d06251cefa6d0-1531242268971-20cabbb57dda221c32ccfa99")
	form.Set("premiumLoungeOpted", "false")
	form.Set("centreId", centreID)

	formString := form.Encode()
	req, err := http.NewRequest("POST", "https://visas-immigration.service.gov.uk/save/payment.appointmentCentreSelection", strings.NewReader(formString))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "_ga=GA1.4.1738972275.1528314464; _gid=GA1.4.1576818527.1531163141; rxVisitor=1531163158628O0I2DPAIB60JEAU2KTCTDS5O2774DA0H; _gat=1; UKV&I_SESSION=\"3c1a7fcd9a35bfb297868116d1e51c066d6fb6c8-csrfToken=ac98bae1f8911f4497aaa7a25fc5aa35b15725ce-1531169251039-20cabbb57dda221c32ccfa99&applicationId=dbc4e9a9-ba2a-4c9a-914e-19e941dcc66e&path=payment.appointmentCentreSelection\"; dtLatC=2; dtPC=11$42269112_662h-vNIDPAPBAELPINIMOBIDBMOIFNFCGAEFG; rxvt=1531244073165|1531242245299; dtCookie=11$DE0F2C320FE1899E0B73FBE740333572|visas-immigration.service.gov.uk|1; dtSa=true%7CC%7C-1%7CSaving...%7C-%7C1531242285261%7C42269112_662%7Chttps%3A%2F%2Fvisas-immigration.service.gov.uk%2Fedit%2Fpayment.appointmentCentreSelection%7CPay%20-%20Choose%20a%20Premium%20Service%20Centre%7C1531242273166%7C")
	resp, err := client.Do(req)

	if err != nil {
		ch <- fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		// ch <- bodyString
		if areAppointments.MatchString(bodyString) {
			ch <- time.Now().String() + ": Appointments at " + longFormIDs[centreID]
			delete(longFormIDs, centreID)
		} else if loggedOut.MatchString(bodyString) {
			ch <- time.Now().String() + ": USER LOGGED OUT!"
			os.Exit(1)
		} else if noAppointments.MatchString(bodyString) {
			ch <- time.Now().String() + ": No Appointments at " + longFormIDs[centreID]
		} else {
			ch <- time.Now().String() + ": Unknown State"
			ch <- bodyString
		}
	}
}
