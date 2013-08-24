package main

import (
  "fmt"
  "net/http"
  "io/ioutil"
  "regexp"
)

type Team struct {
  teamNumber, teamID string
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
  re, _ := regexp.Compile(`<a href="/whats-going-on/team/FRC/([\d]+?)\">([\d]+?)</a>`)
  res := re.FindAllStringSubmatch(string(contents), -1)
  teams := make([]Team, 1)
  for _, teamMatch := range res {
    t := Team{teamMatch[2], teamMatch[1]}
    teams = append(teams, t)
  }
  c <-teams
}

func main() {
  // Do a call to get a list of all of the teams (2013)
  c := make(chan []Team)
  n := 0
  for i := 0; i < 93; i++ {
    url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=%d&ProgramCode=FRC&Season=2013&Country=USA&StateProv=&ZipCode=&Radius=&op=Search&form_build_id=form-YX7Qw3xg7FJMXiUh4BDpghbrZPIgnDubFk9bU9jM9S8&form_id=first_search_teams_form&sort=asc&order=Team%%20Number", i)
    go getTeams(url, c)
    n++
  }
  for i := n; i > 0; i-- {
    n := <-c
    fmt.Println(n)
  }
}