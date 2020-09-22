 package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"errors"
	"flag"
	"os/signal"
	"syscall"
)

type paramCheck struct {
	url   string
	param string
}


type arrayFlags []string

func (i *arrayFlags) String() string {
    return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: time.Second,
		DualStack: true,
	}).DialContext,
}

var httpClient = &http.Client{
	Transport: transport,
}


var headers arrayFlags
var lastUrlChecked string

func main() {
	var resultFile *os.File;
	var delayTime = flag.Int("d", 0, "Duration of the delay between urls scans (in milliseconds)")
	var paramMin = flag.Int("s", 0, "Saves urls with [s] unflitered chars to result.txt")
	flag.Var(&headers, "h", "Add Header. Usage: \"[HeaderName]: [HeaderContent]\"")
	flag.Parse()

	// SetupCloseHandler()

	if *paramMin > 0 {
		// Create result.txt
		resultFile, err := os.OpenFile("result.txt", os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
		    panic(err)
		}

		defer resultFile.Close()
	}

	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	sc := bufio.NewScanner(os.Stdin)

	initialChecks := make(chan paramCheck, 40)

	appendChecks := makePool(initialChecks, func(c paramCheck, output chan paramCheck) {
		reflected, err := checkReflected(c.url)
		if err != nil {
			//fmt.Fprintf(os.Stderr, "error from checkReflected: %s\n", err)
			return
		}

		if len(reflected) == 0 {
			// TODO: wrap in verbose mode
			//fmt.Printf("no params were reflected in %s\n", c.url)
			return
		}

		for _, param := range reflected {
			output <- paramCheck{c.url, param}
		}
	})

	charChecks := makePool(appendChecks, func(c paramCheck, output chan paramCheck) {
		wasReflected, err := checkAppend(c.url, c.param, "iy3j4h234hjb23234")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error from checkAppend for url %s with param %s: %s", c.url, c.param, err)
			return
		}

		if wasReflected {
			output <- paramCheck{c.url, c.param}
		}
	})

	done := makePool(charChecks, func(c paramCheck, output chan paramCheck) {
		output_of_url := []string{c.url, c.param}
		for _, char := range []string{"\"", "'", "<", ">", "\\"} {
			wasReflected, err := checkAppend(c.url, c.param, "aprefix"+char+"asuffix")
			if err != nil {
				// fmt.Fprintf(os.Stderr, "error from checkAppend for url %s with param %s with %s: %s", c.url, c.param, char, err)
				continue
			}

			if wasReflected {
				output_of_url = append(output_of_url, char)
			}
		}
		if len(output_of_url) > 2 {
			unfliteredChars := output_of_url[2:]
			
			if len(unfliteredChars) > 1 || !Find(unfliteredChars, "\\"){
				fmt.Printf("URL: %s Param: %s Unfiltered: %v \n", output_of_url[0] , output_of_url[1],output_of_url[2:])
			} 
			
			if *paramMin > 0 && len(unfliteredChars) >= *paramMin {
				// Write to the file
				// fmt.Println("Writing to the file")
				fmt.Fprintf(resultFile, fmt.Sprintf("URL: %s Param: %s Unfiltered: %v \n", output_of_url[0] , output_of_url[1],output_of_url[2:])) // need to fix
			}
		}
	})

	for sc.Scan() {
		initialChecks <- paramCheck{url: sc.Text()}
		time.Sleep(time.Duration(*delayTime) * time.Millisecond) // Waits between each url scaning
	}

	close(initialChecks)
	<-done
}

func checkReflected(targetURL string) ([]string, error) {

	out := make([]string, 0)
	
	lastUrlChecked = targetURL 

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return out, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.100 Safari/537.36")
	for _, header := range headers { // Adds the headers to the http packet
		headerName, headerContent, err := splitHeader(header)
		if err != nil {
			// fmt.Fprintf(os.Stderr, "HeaderError: %s\n", err)
			continue
		}
		// Trims the whitespaces
		headerName = strings.TrimSpace(headerName)
		headerContent = strings.TrimSpace(headerContent)
		// fmt.Printf("Name: =>%s<= | Content: =>%s<=\n", headerName, headerContent)
		req.Header.Add(headerName, headerContent)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return out, err
	}
	if resp.Body == nil {
		return out, err
	}
	defer resp.Body.Close()

	// always read the full body so we can re-use the tcp connection
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return out, err
	}

	// nope (:
	if strings.HasPrefix(resp.Status, "3") {
		return out, nil
	}

	// also nope
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "html") {
		return out, nil
	}

	body := string(b)

	u, err := url.Parse(targetURL)
	if err != nil {
		return out, err
	}

	for key, vv := range u.Query() {
		for _, v := range vv {
			if !strings.Contains(body, v) {
				continue
			}

			out = append(out, key)
		}
	}

	return out, nil
}

func checkAppend(targetURL, param, suffix string) (bool, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}

	qs := u.Query()
	val := qs.Get(param)
	//if val == "" {
	//return false, nil
	//return false, fmt.Errorf("can't append to non-existant param %s", param)
	//}

	qs.Set(param, val+suffix)
	u.RawQuery = qs.Encode()

	reflected, err := checkReflected(u.String())
	if err != nil {
		return false, err
	}

	for _, r := range reflected {
		if r == param {
			return true, nil
		}
	}

	return false, nil
}

type workerFunc func(paramCheck, chan paramCheck)

func makePool(input chan paramCheck, fn workerFunc) chan paramCheck {
	var wg sync.WaitGroup

	output := make(chan paramCheck)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			for c := range input {
				fn(c, output)
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}

func splitHeader(header string) (string, string, error) {
	s := strings.SplitN(header, ":", 2) // Splits only the first ':'
	if len(s) != 2 {
		return "", "", errors.New("Invalid header \"" + header + "\"")
	}
	headerName := s[0]
	headerContent := s[1]
	return headerName, headerContent, nil
}

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	quit := make(chan bool)		
	go func() {
		for {
			<-c
			fmt.Println("\r- Ctrl+C pressed in Terminal")
			fmt.Println("Would you like to exit? [y/n]")
			
			go func() { // prints the last checked url
			    for {
			        select {
			        case <-quit:
			            return
			        default:
			            fmt.Printf("\rLast url: %s", lastUrlChecked)
			        }
			    }
			}()
			
			// wait for input
			var userSelection string
			_, err := fmt.Scanln(&userSelection)
			if err != nil {
				
			}
			fmt.Println("userSelection: ", userSelection)
			quit <- true
			os.Exit(0)
		}
	}()
}

// Find takes a slice and looks for an element in it. If found it will
// return true, otherwise false
func Find(slice []string, val string) (bool) {
    for _, item := range slice {
        if item == val {
            return true
        }
    }
    return false
}