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
)

type Team struct {
  State       string  `bson:"state"`
  City        string  `bson:"city"`
  TeamName    string  `bson:"teamName"`
  TeamID      string  `bson:"teamID"`
  TeamNumber  string  `bson:"teamNumber"`
}

type PageRequest struct {
  numPages int
  country string
  err error
}

type Award struct {
  Team    string  `bson:"team"`
  Year    string  `bson:"year"`
  EventID string  `bson:"eventID"`
  Event   string  `bson:"event"`
  Award   string  `bson:"award"`
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

  url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/team/FRC/%s", team.TeamID)
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
      a := Award{team.TeamNumber, res[0][1], res[0][2], res[0][3], award}
      awardArray = append(awardArray, a)
    }
  }
  returnChannel<- awardArray
}

func getNumberOfPages(country, year string, returnChannel chan<- *PageRequest) {

  url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=0&ProgramCode=FRC&Season=%s&Country=%s&sort=asc&order=Team%%20Number", year, country)
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

  sess.SetSafe(&mgo.Safe{})

  for _, year := range []string{"2013", "2012", "2011", "2010", "2009"} {
    name := fmt.Sprintf("ff-%s", year)
    collection := sess.DB("ff").C(name)

    teamChannel := make(chan []Team)
    // awardChannel := make(chan []Award)  
    pageRequestChannel := make(chan *PageRequest)

    teamCount := 0

    countries, err := getCountries()
    if err != nil {
      return
    }

    for _, country := range countries {
      go getNumberOfPages(country, year, pageRequestChannel)
    }
    
    for i := len(countries); i > 0; i-- {
      pageReq := <-pageRequestChannel
      if pageReq.err != nil {
        fmt.Println("Error!")
        fmt.Println(err)
        continue

      }
      for i := 0; i <= pageReq.numPages; i++ {
        url := fmt.Sprintf("http://www.usfirst.org/whats-going-on/teams?page=%d&ProgramCode=FRC&Season=%s&Country=%s&sort=asc&order=Team%%20Number", i, year, pageReq.country)
        go getTeams(url, teamChannel)
        teamCount++
      }
    }

    for i := teamCount; i > 0; i-- {
      for _, team := range <-teamChannel {
        err = collection.Insert(team)
        if err != nil {
          fmt.Printf("Can't insert document: %v\n", err)
        } else {
          fmt.Println("Inserted ", team.TeamNumber)
        }
      }
    }

  }
}