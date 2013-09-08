package main

import (
  "fmt"
  "net/http"
  "io/ioutil"
  "regexp"
  "strconv"
  "strings"
  "labix.org/v2/mgo"
  "os"
  "runtime"
)

type Team struct {
  state, city, teamName, teamID, teamNumber string
}

type PageRequest struct {
  numPages int
  country string
  err error
}

type Award struct {
  team Team
  year, eventID, event, award string
}

type WLT struct {
  teamNumber, w, l, t string
}

func getPageContent(url string) (response string, err error) {

  resp, err := http.Get(url)
  if err != nil {
    return "", err
  }
  defer resp.Body.Close()
  contents, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return "", err
  }
  return string(contents), nil
}

func getAwards(team Team, returnChannel chan<- []Award) {

  url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/team/FRC/%s", team.teamID)
  response, err := getPageContent(url)
  if err != nil {
    returnChannel<- nil
    return
  }

  awardArray := make([]Award, 0)

  re, _ := regexp.Compile(`<tr(?s)(.*?)</tr>`)
  res := re.FindAllStringSubmatch(response, -1)
  for _, awardRow := range res[1:] {
    re, _ = regexp.Compile(`<tr class="(?:odd|even)"><td>([\d]{4})</td><td><a href="/whats-going-on/event/(\d+)">(?:[\d]{4}) - (.*?)</a></td><td>(.*?)</td>`)
    res = re.FindAllStringSubmatch(awardRow[0], -1)
    awards := strings.Split(res[0][4], ", ")
    for _, award := range awards {
      a := Award{team, res[0][1], res[0][2], res[0][3], award}
      awardArray = append(awardArray, a)
    }
  }
  returnChannel<- awardArray
}

func getNumberOfPages(country string, returnChannel chan<- *PageRequest) {

  url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=0&ProgramCode=FRC&Season=2013&Country=%s&sort=asc&order=Team%%20Number", country)
  contents, err := getPageContent(url)
  if err != nil {
    returnChannel<- &PageRequest{0, "", err}
    return
  }

  re, _ := regexp.Compile(`<a title="Go to last page" href="/whats-going-on/teams\?page=([\d]+?)&amp`)
  res := re.FindStringSubmatch(contents)
  num := 0
  if len(res) > 0 {
    num, _ = strconv.Atoi(res[1])
  }
  returnChannel<- &PageRequest{num, country, nil}
}

func getTeams(url string, c chan<- []Team) {

  contents, err := getPageContent(url)
  if err != nil {
    // Handle error
  }
  re, _ := regexp.Compile(`<tr class="(even|odd)"><td>US</td><td>([\w\s]+?)</td><td>([\w\s]+?)</td><td>([\w\s]+?)</td><td><a href="/whats-going-on/team/FRC/([\d]+?)\">([\d]+?)</a>`)
  res := re.FindAllStringSubmatch(string(contents), -1)
  teams := make([]Team, 0)
  for _, teamMatch := range res {
    t := Team{teamMatch[2], teamMatch[3], teamMatch[4], teamMatch[5], teamMatch[6]}
    teams = append(teams, t)
  }
  c <-teams
}

func getOverallWLT(teamNumber string, c chan<- WLT) {

  url := fmt.Sprintf("http://www.thebluealliance.com/team/%s/2013", teamNumber)
  contents, err := getPageContent(url)
  if err != nil {

  }
  re, _ := regexp.Compile(`<strong>([\d]{1,2})-([\d]{1,2})-([\d]{1,2})</strong>`)
  res := re.FindAllStringSubmatch(string(contents), -1)
  if len(res) > 0 {
    wlt := WLT{teamNumber, res[0][1], res[0][2], res[0][3]}
    c <-wlt
  } else {
    c <-WLT{teamNumber, "", "", ""}
  }
}

func getCountries() (countryArray []string, err error) {

  contents, err := getPageContent("http://www.usfirst.org/whats-going-on")
  if err != nil {
    return
  }
  reOuter, _ := regexp.Compile(`<select id="edit-country--2"(.*)`)
  reInner, _ := regexp.Compile(`<option value="(.*?)"`)
  resOuter := reOuter.FindString(contents)
  resInner := reInner.FindAllStringSubmatch(resOuter, -1)
  countryArray = make([]string, 0)
  for _, country := range resInner {
    c := strings.Replace(country[1], " ", "+", -1)
    countryArray = append(countryArray, c)
  }

  return countryArray, nil
}

func main() {

  team2337 := Team{"Grand Blanc", "MI", "EngiNERDs", "84793", "2337"}

  awardChannel := make(chan []Award, 10)  

  go getAwards(team2337, awardChannel)
  fmt.Println(<-awardChannel)
  return

  uri := os.Getenv("FF_MONGO_URL")
  if uri == "" {
    fmt.Println("no connection string provided")
    os.Exit(1)
  }

  sess, err := mgo.Dial(uri)
  if err != nil {
    fmt.Printf("Can't connect to mongo, go error %v\n", err)
    os.Exit(1)
  }
  defer sess.Close()

  // Do a call to get a list of all of the teams (2013)
  teamChannel := make(chan []Team, 10)
  n1 := 0

  countries, err := getCountries()
  if err != nil {
    return
  }

  pageRequestChannel := make(chan *PageRequest, 10)

  for _, country := range countries {
    go getNumberOfPages(country, pageRequestChannel)
  }
  
  for i := len(countries); i > 0; i-- {
    pageReq := <-pageRequestChannel
    if err := pageReq.err; err != nil {
      fmt.Println("Error!")
      fmt.Println(err)
      continue
    }
    for i := 0; i <= pageReq.numPages; i++ {
      url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=%d&ProgramCode=FRC&Season=2013&Country=%s&sort=asc&order=Team%%20Number", i, pageReq.country)
      go getTeams(url, teamChannel)
      n1++
    }
  }

  wltChannel := make(chan WLT, 10)
  n2 := 0
  // urlArray := make(map[string] string)
  for i := n1; i > 0; i-- {
    fmt.Println(i)
    tt := <-teamChannel
    for _, team := range tt {
      go getOverallWLT(team.teamNumber, wltChannel)
      n2++
    }
  }
  for i := n2; i > 0; i-- {
    fmt.Println(<-wltChannel)
  }
}