package main

import (
  "fmt"
  "net/http"
  "io/ioutil"
  "regexp"
  "strconv"
  "strings"
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
  year, event, award string
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

func getAwards(team string) (a string, err error) {
  url := fmt.Sprintf("http://www.thebluealliance.com/team/%s/history", team)
  response, err := getPageContent(url)
  if err != nil {
    return "", err
  }

  re, _ := regexp.Compile(`<table(.*?)</table>`)
  res := re.FindAllStringSubmatch(response, -1)
  fmt.Println(res)
  return "", nil
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
    // Handle error
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
  a, _ := getAwards("2337")
  if a != "" {
    fmt.Println("No")
  }
  return

  // Do a call to get a list of all of the teams (2013)
  teamChannel := make(chan []Team)
  n1 := 0

  countries, err := getCountries()
  if err != nil {
    return
  }

  pageRequestChannel := make(chan *PageRequest)

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

  wltChannel := make(chan WLT)
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