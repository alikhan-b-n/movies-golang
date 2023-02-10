package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, jobs := range jobs {
		wg.Add(1)

		out := make(chan interface{})

		go func(job job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)

			job(in, out)

		}(jobs, in, out, wg)

		in = out
	}

	wg.Wait()

}

func SingleHash(in chan interface{}, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for i := range in {
		wg.Add(1)

		go func(in interface{}, out chan interface{}, wg *sync.WaitGroup, mu *sync.Mutex) {
			defer wg.Done()
			data := strconv.Itoa(in.(int))

			mu.Lock()
			md5Data := DataSignerMd5(data)
			mu.Unlock()

			crc32Chan := make(chan string)

			go func(data string, out chan string) {
				out <- DataSignerCrc32(data)
			}(data, crc32Chan)

			crc32Data := <-crc32Chan
			crc32Md5Data := DataSignerCrc32(md5Data)

			out <- crc32Data + "~" + crc32Md5Data
		}(i, out, wg, mu)
	}
	wg.Wait()

}

func MultiHash(in chan interface{}, out chan interface{}) {
	const THREAD int = 6
	wg := &sync.WaitGroup{}

	for i := range in {
		wg.Add(1)

		go func(in string, out chan interface{}, th int, wg *sync.WaitGroup) {
			defer wg.Done()

			mu := &sync.Mutex{}
			jobWg := &sync.WaitGroup{}
			comb := make([]string, th)

			for i := 0; i < th; i++ {
				jobWg.Add(1)
				data := strconv.Itoa(i) + in

				go func(acc []string, index int, data string, jobWg *sync.WaitGroup, mu *sync.Mutex) {
					defer jobWg.Done()
					data = DataSignerCrc32(data)

					mu.Lock()
					acc[index] = data
					mu.Unlock()
				}(comb, i, data, jobWg, mu)
			}

			jobWg.Wait()
			out <- strings.Join(comb, "")
		}(i.(string), out, THREAD, wg)
	}

	wg.Wait()

}

func CombineResults(in, out chan interface{}) {
	var results []string

	for i := range in {
		results = append(results, i.(string))
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")

}
