package main

import (
  "fmt"
  "net/http"
  "io/ioutil"
  "regexp"
  "strconv"
)

type Team struct {
  state, city, teamName, teamID, teamNumber string
}

type WLT struct {
  teamNumber, w, l, t string
}

func getNumberOfPages() (num int, err error) {
  resp, err := http.Get("http://www.usfirst.org/whats-going-on/teams?page=0&ProgramCode=FRC&Season=2013&Country=USA&sort=asc&order=Team%%20Number")
  if err != nil {
    return -1, err
  }
  defer resp.Body.Close()
  contents, err := ioutil.ReadAll(resp.Body)
  re, _ := regexp.Compile(`<a title="Go to last page" href="/whats-going-on/teams\?page=([\d]+?)&amp`)
  res := re.FindStringSubmatch(string(contents))
  num, _ = strconv.Atoi(res[1])
  return num, nil
}

func getTeams(url string, c chan<- []Team) {
  resp, err := http.Get(url)
  if err != nil {
    return
  }
  defer resp.Body.Close()
  contents, err := ioutil.ReadAll(resp.Body)
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

func getWLT(teamNumber string, c chan<- WLT) {
  url := fmt.Sprintf("http://www.thebluealliance.com/team/%s/2013", teamNumber)
  resp, err := http.Get(url)
  if err != nil {
    return
  }
  defer resp.Body.Close()
  contents, err := ioutil.ReadAll(resp.Body)
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

func main() {
  // Do a call to get a list of all of the teams (2013)
  c1 := make(chan []Team)
  c2 := make(chan WLT)
  n1 := 0
  // Check how many teams to get
  numPages, err := getNumberOfPages()
  if err != nil {
    return
  }
  fmt.Println(numPages)
  for i := 0; i <= numPages; i++ {
    url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=%d&ProgramCode=FRC&Season=2013&Country=USA&sort=asc&order=Team%%20Number", i)
    go getTeams(url, c1)
    n1++
  }
  n2 := 0
  // urlArray := make(map[string] string)
  for i := n1; i > 0; i-- {
    tt := <-c1
    for _, team := range tt {
      go getWLT(team.teamNumber, c2)
      n2++
    }
    fmt.Println(i)
  }
  for i := n2; i > 0; i-- {
    fmt.Println(<-c2)
  }
}