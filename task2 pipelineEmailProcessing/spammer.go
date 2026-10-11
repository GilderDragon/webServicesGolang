package main

import (
	"fmt"
	"slices"
	"sync"
)

func RunPipeline(cmds ...cmd) {
	in := make(chan interface{})
	close(in)

	wg := &sync.WaitGroup{}

	for _, c := range cmds {
		out := make(chan interface{})

		wg.Add(1)

		go func(c cmd, in, out chan interface{}) {
			defer wg.Done()
			c(in, out)
			close(out)
		}(c, in, out)

		in = out
	}

	wg.Wait()
}

func SelectUsers(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	var users = make(map[User]struct{})
	for v := range in {
		wg.Add(1)

		go func(v interface{}) {
			defer wg.Done()

			user := GetUser(v.(string))

			mu.Lock()

			_, ok := users[user]
			if ok {
				mu.Unlock()
				return
			}

			users[user] = struct{}{}
			mu.Unlock()

			out <- user
		}(v)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	batch := make([]User, 0, GetMessagesMaxUsersBatch)
	for v := range in {
		batch = append(batch, v.(User))
		if len(batch) == GetMessagesMaxUsersBatch {
			wg.Add(1)

			go func(v []User) {
				defer wg.Done()

				msgID, _ := GetMessages(v...)

				for _, val := range msgID {
					out <- val
				}
			}(append([]User{}, batch...))
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		wg.Add(1)

		go func(v []User) {
			defer wg.Done()

			msgID, _ := GetMessages(v...)

			for _, val := range msgID {
				out <- val
			}
		}(batch)
	}

	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	sem := make(chan struct{}, 5)

	for v := range in {
		wg.Add(1)

		sem <- struct{}{}

		go func(v interface{}) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			isSpam, _ := HasSpam(v.(MsgID))

			out <- MsgData{
				ID:      v.(MsgID),
				HasSpam: isSpam,
			}
		}(v)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	msgDatas := make([]MsgData, 0)
	for v := range in {
		msgDatas = append(msgDatas, v.(MsgData))
	}

	slices.SortFunc(msgDatas, func(a, b MsgData) int {
		if a.HasSpam == b.HasSpam {
			if a.ID == b.ID {
				return 0
			}

			if a.ID < b.ID {
				return -1
			}

			return 1
		}

		if a.HasSpam && !b.HasSpam {
			return -1
		}

		return 1
	})

	for _, msgData := range msgDatas {
		out <- fmt.Sprintf("%v %v", msgData.HasSpam, msgData.ID)
	}
}
