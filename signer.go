package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type structOfMultiHash struct {
	number int
	hash   string
}

func ExecutePipeline(signJobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, jobItem := range signJobs {
		wg.Add(1)
		out := make(chan interface{})
		go func(jobFunction job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			jobFunction(in, out)
		}(jobItem, in, out, wg)
		in = out
	}

	defer wg.Wait()
}
func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	chanStruct := make(chan struct{}, 1)

	for currentData := range in {
		currentData := fmt.Sprintf("%v", currentData)
		uniChan := make(chan string, 1)

		wg.Add(1)
		go func(data string, out chan<- string, chanStruct chan struct{}) {
			defer wg.Done()

			chanStruct <- struct{}{}
			dataHash := DataSignerMd5(data)
			<-chanStruct
			out <- DataSignerCrc32(dataHash)
		}(currentData, uniChan, chanStruct)

		wg.Add(1)
		go func(data string, in <-chan string, out chan<- interface{}) {
			defer wg.Done()

			out <- DataSignerCrc32(data) + "~" + <-in
		}(currentData, uniChan, out)
	}

	wg.Wait()
}
func CombineResults(in, out chan interface{}) {
	var results []string
	var result string
	for hashResult := range in {
		results = append(results, (hashResult).(string))
	}
	sort.Strings(results)
	result = strings.Join(results, "_")
	out <- result
}
func MultiHash(in chan interface{}, out chan interface{}) {
	wg := &sync.WaitGroup{}
	outTemp := make(chan string)
	for input := range in {
		wg.Add(1)

		wgTemp := &sync.WaitGroup{}
		data := input.(string)
		inCh := make(chan structOfMultiHash)

		wgTemp.Add(6)
		for i := 0; i < 6; i++ {
			go func(hashResultChan chan structOfMultiHash, wg *sync.WaitGroup, singleHash interface{}, i int) {
				defer wg.Done()
				hashResultChan <- structOfMultiHash{number: i, hash: DataSignerCrc32(fmt.Sprintf("%v%v", i, singleHash))}
			}(inCh, wgTemp, data, i)
		}
		go func(wgInner *sync.WaitGroup, c chan structOfMultiHash) {
			defer close(c)
			wgInner.Wait()
		}(wgTemp, inCh)

		go func(hashResults chan structOfMultiHash, out chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			result := map[int]string{}
			var data []int

			for hashResult := range hashResults {
				result[hashResult.number] = hashResult.hash
				data = append(data, hashResult.number)
			}
			sort.Ints(data)

			var results []string
			for i := range data {
				results = append(results, result[i])
			}

			out <- strings.Join(results, "")
		}(inCh, outTemp, wg)

	}

	go func(wgOut *sync.WaitGroup, c chan string) {
		defer close(c)
		wgOut.Wait()
	}(wg, outTemp)

	for hash := range outTemp {
		out <- hash
	}

}
