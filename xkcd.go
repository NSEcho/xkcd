// Package xkcd provides interface to search xcde comics locally. It
// does so by first indexing data offline locally and then search those.
package xkcd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type XKCD struct {
	Month      string
	Num        int
	Link       string
	Year       string
	News       string
	SafeTitle  string `json:"safe_title"`
	Transcript string
	Alt        string
	Img        string
	Title      string
	Day        string
}

func save(comics []XKCD) {
	file, _ := json.MarshalIndent(comics, "", "    ")
	_ = ioutil.WriteFile("output.json", file, 0644)
}

func fetchLast() int {
	url := "https://xkcd.com"
	r := regexp.MustCompile(`Permanent link to this comic: https://xkcd.com/(\d+)`)
	resp, err := http.Get(url)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1
	}

	match := r.FindStringSubmatch(string(body))
	last, _ := strconv.Atoi(match[1])
	return last
}

func fillToLast(last int, ch chan int) {
	for i := 1; i <= last; i++ {
		ch <- i
	}
	close(ch)
}

func workers(last int, chCom chan *XKCD) {
	chNum := make(chan int, 100)
	go fillToLast(last, chNum)
	var wg sync.WaitGroup
	for i := 1; i < 100; i++ {
		wg.Add(1)
		go fetchOne(chNum, chCom, last, &wg)
	}
	wg.Wait()
	close(chCom)
}

func fetchOne(chNum chan int, chCom chan *XKCD, last int, wg *sync.WaitGroup) {
	for number := range chNum {
		var result XKCD
		url := fmt.Sprintf("https://xkcd.com/%d/info.0.json", number)
		resp, err := http.Get(url)
		if err != nil {
			chCom <- nil
		}
		if resp.StatusCode != http.StatusOK {
			if number > last {
				chCom <- nil
			}
			chCom <- &XKCD{Month: "66"}
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			chCom <- nil
		}
		chCom <- &result
		resp.Body.Close()
	}
	wg.Done()
}

func Index() (int, bool) {
	fmt.Println("[*] Starting indexing")
	start := time.Now()
	last := fetchLast()
	if last == -1 {
		return -1, false
	}
	var comics []XKCD
	chComics := make(chan *XKCD, 100)
	go workers(last, chComics)

	for comic := range chComics {
		if comic != nil && comic.Month != "66" {
			comics = append(comics, *comic)
		}
	}
	save(comics)
	fmt.Printf("[*] Indexing finished. Elapsed: %.2f\n", time.Since(start).Seconds())
	return 1, true
}
