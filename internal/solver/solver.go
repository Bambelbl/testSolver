package solver

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
)

type TestSolver struct {
	id              int
	addrServer      string
	log             *zap.SugaredLogger
	countOfQuestion int
	testResult      string
}

func NewTestSolver(id int, addrServer string, log *zap.SugaredLogger) *TestSolver {
	return &TestSolver{
		id:              id,
		addrServer:      addrServer,
		log:             log,
		countOfQuestion: 1,
	}
}

func (ts *TestSolver) Solve() (res int) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		ts.log.Errorf("%d: CLIENT SETTING ERROR: %s", ts.id, err.Error())
		return
	}
	client := &http.Client{
		Transport: &http.Transport{},
		Jar:       jar,
	}

	fmt.Printf("%d: CLIENT SETTING WAS COMPLETED\n", ts.id)

	var resp *http.Response
	for resp, err = client.Get(ts.addrServer); !(err == nil && resp.StatusCode == 200); {
		if err != nil {
			ts.log.Errorf("%d: REQUEST ERROR: %s", ts.id, err.Error())
		}
		resp, err = client.Get(ts.addrServer)
	}

	pathToStart, err := findStartTest(resp.Body)
	if err != nil {
		ts.log.Errorf("%d: START PATH ERROR: %s", ts.id, err.Error())
	}
	err = resp.Body.Close()
	if err != nil {
		ts.log.Errorf("%d: RESPONSE BODY ERROR: %s", ts.id, err.Error())
		return
	}

	for resp, err = client.Get(ts.addrServer + pathToStart); !(err == nil && resp.StatusCode == 200); {
		if err != nil {
			ts.log.Errorf("%d: REQUEST ERROR: %s", ts.id, err.Error())
		}
		resp, err = client.Get(ts.addrServer + pathToStart)
	}

	fmt.Printf("%d: START TO SOLVE\n", ts.id)
	for i := 1; i <= ts.getCountOfQuestion(); i++ {
		payload, err := ts.solveQuestion(resp.Body)
		if err != nil {
			ts.log.Errorf("%d: QUESTION №%d PROCESSING ERROR: %s", ts.id, i, err.Error())
			return
		}
		err = resp.Body.Close()
		if err != nil {
			ts.log.Errorf("%d: RESPONSE BODY ERROR: %s", ts.id, err.Error())
			return
		}

		if err != nil {
			ts.log.Errorf("%d: LIMITER ERROR: %s", ts.id, err.Error())
		}
		for resp, err = client.PostForm(ts.addrServer+resp.Request.URL.Path, payload); !(err == nil && resp.StatusCode == 200); {
			if err != nil {
				ts.log.Errorf("%d: REQUEST ERROR: %s", ts.id, err.Error())
			}
			resp, err = client.PostForm(ts.addrServer+resp.Request.URL.Path, payload)
		}
		fmt.Printf("%d: QUESTION №%d WAS SOLVED\n", ts.id, i)
	}
	ts.testResult, err = parseResult(resp.Body)
	if err != nil {
		ts.log.Errorf("%d: RESULT PROCESSING ERROR: %s", ts.id, err.Error())
	}
	err = resp.Body.Close()
	if err != nil {
		ts.log.Errorf("%d: RESPONSE BODY ERROR: %s", ts.id, err.Error())
		return
	}
	if ts.testResult != "Passed" {
		return
	}
	fmt.Printf("%d: TEST PASSED\n", ts.id)
	return 1
}

func findStartTest(reader io.Reader) (path string, err error) {
	tkn := html.NewTokenizer(reader)
	for {
		tokenType := tkn.Next()
		tag, _ := tkn.TagName()
		switch {
		case tokenType == html.ErrorToken:
			return "", errors.New("start path can't be found on page")
		case tokenType == html.StartTagToken && string(tag) == "a":
			for {
				attrKey, attrValue, moreAttr := tkn.TagAttr()
				if string(attrKey) == "href" {
					return string(attrValue), nil
				}
				if !moreAttr {
					break
				}
			}
		}
	}
}

func (ts *TestSolver) getCountOfQuestion() int {
	return ts.countOfQuestion
}
func (ts *TestSolver) solveQuestion(reader io.Reader) (values url.Values, err error) {
	values = make(url.Values, 0)
	tkn := html.NewTokenizer(reader)
	for {
		tokenType := tkn.Next()
		tag, _ := tkn.TagName()
		switch {
		case tokenType == html.ErrorToken:
			return url.Values{}, errors.New("there are't sub questions on page")
		case tokenType == html.StartTagToken && string(tag) == "h1":
			tokenType = tkn.Next()
			title := tkn.Token().Data
			ts.countOfQuestion, err = strconv.Atoi(string(title[len(title)-1]))
		case tokenType == html.StartTagToken && string(tag) == "form":
			for {
				tokenType = tkn.Next()
				tag, _ = tkn.TagName()
				switch {
				case string(tag) == "input":
					var inputType, name string
					var radioFirstVal string
					for {
						attrKey, attrValue, moreAttr := tkn.TagAttr()
						switch string(attrKey) {
						case "type":
							inputType = string(attrValue)
						case "name":
							name = string(attrValue)
						case "value":
							radioFirstVal = string(attrValue)
						}
						if !moreAttr {
							break
						}
					}
					if inputType == "text" {
						values[name] = []string{"test"}
					} else {
						values[name] = []string{findLongestValue(tkn, radioFirstVal)}
					}
				case string(tag) == "select":
					var name string
					for {
						attrKey, attrValue, moreAttr := tkn.TagAttr()
						if string(attrKey) == "name" {
							name = string(attrValue)
							break
						}
						if !moreAttr {
							break
						}
					}
					values[name] = []string{findLongestValue(tkn, "")}
				case tokenType == html.EndTagToken && string(tag) == "form":
					return
				}
			}
		}
	}
}

func findLongestValue(tkn *html.Tokenizer, curAns string) string {
	maxLen := len(curAns)
	for {
		tokenType := tkn.Next()
		tag, _ := tkn.TagName()
		switch {
		case string(tag) == "input" || string(tag) == "option":
			for {
				attrKey, attrValue, moreAttr := tkn.TagAttr()
				if string(attrKey) == "value" {
					val := string(attrValue)
					if len(val) > maxLen {
						curAns = val
						maxLen = len(val)
					}
				}
				if !moreAttr {
					break
				}
			}
		case tokenType == html.EndTagToken && string(tag) == "p":
			return curAns
		}
	}
}

func parseResult(reader io.Reader) (res string, err error) {
	tkn := html.NewTokenizer(reader)
	for {
		tokenType := tkn.Next()
		tag, _ := tkn.TagName()
		switch {
		case tokenType == html.ErrorToken:
			return "", errors.New("test result can't be found on page")
		case tokenType == html.StartTagToken && string(tag) == "h1":
			tokenType = tkn.Next()
			return tkn.Token().Data, nil
		}
	}
}
