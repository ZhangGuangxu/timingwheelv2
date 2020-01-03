package timingwheelv2

import (
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"
)

const maxTime = 1000 * time.Millisecond

type Item struct {
	createTime time.Time
}

func newItem() *Item {
	return &Item{
		createTime: time.Now(),
	}
}

func (i *Item) Release() {

}

func TestWheel(test *testing.T) {
	slotCnt := 100
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		test.Errorf("NewTimingWheel return error[%v]", err)
	}

	quit := make(chan bool)
	quitCh := func() chan bool {
		return quit
	}

	wg := &sync.WaitGroup{}
	deferFunc := func() {
		log.Println("before wg.Done()")
		wg.Done()
	}

	wg.Add(1)
	go w.Run(quitCh, deferFunc)

	now := time.Now()

	wg.Add(1)
	go func() {
		defer deferFunc()
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		t := now
		endT := now.Add(1 * time.Second)
		cache := make([]*Item, 0, 100)
		for t.Before(endT) {
			select {
			case <-ticker.C:
				item := newItem()

				r := rand.Intn(10)
				if r%2 != 0 {
					cache = append(cache, item)
				}

				r2 := -1
				cacheLen := len(cache)
				if rand.Intn(2) > 0 && cacheLen > 0 {
					r2 = rand.Intn(cacheLen)
					w.AddItem(cache[r2])
				} else {
					w.AddItem(item)
				}
				t = time.Now()
			}
		}
	}()

	t1 := now.Add(1 * time.Second)
	tm := now.Add(1300 * time.Millisecond)
	t2 := now.Add(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var itemCount int

	for t := now; t.Before(t2); {
		select {
		case <-ticker.C:
			c := w.itemCount()
			if t.Before(t1) {
				if !(c > itemCount) {
					test.Errorf("not more than before, c=%v, itemCount=%v", c, itemCount)
				}
			} else if t.After(tm) {
				if !(c < itemCount) {
					test.Errorf("not less than before, c=%v, itemCount=%v", c, itemCount)
				}
			}
			itemCount = c
			t = time.Now()
		}
	}

	time.Sleep(3500 * time.Millisecond)
	if w.itemCount() != 0 {
		test.Errorf("wheel still has some item, count=%d", w.itemCount())
	}
	close(quit)
	log.Println("before wg.Wait()")
	wg.Wait()
}

func TestTimingwheel2(test *testing.T) {
	if _, err := NewTimingWheel(-1, 2); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(0, 2); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(2, -1); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(2, 0); err == nil {
		test.Error("should return error")
	}

	if _, err := NewTimingWheel(1, 2); err == nil {
		test.Error("should return error")
	}

	w, _ := NewTimingWheel(3*time.Nanosecond, 2)
	if w.stepTime != 2 {
		test.Error("w.stepTime should be 2")
	}

	slotCnt := 5
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		test.Errorf("NewTimingWheel return error[%v]", err)
	}

	quit := make(chan bool, 1)
	quitCh := func() chan bool {
		return quit
	}

	var wg sync.WaitGroup
	deferFunc := func() {
		wg.Done()
	}

	wg.Add(1)
	go w.Run(quitCh, deferFunc)

	wg.Add(1)
	go func() {
		defer deferFunc()

		t1 := time.NewTicker(50 * time.Millisecond)
		t2 := time.NewTicker(110 * time.Millisecond)
		t3 := time.NewTicker(235 * time.Millisecond)
		t4 := time.NewTicker(80 * time.Millisecond)
		endT := time.Now().Add(2 * time.Second)
		m := newItem()

		for time.Now().Before(endT) {
			select {
			case <-t1.C:
				w.AddItem(m)
				w.AddItem(m)
			case <-t2.C:
				w.AddItem(m)
			case <-t3.C:
				m = newItem()
				w.AddItem(m)
			case <-t4.C:
				w.DelItem(m)
			}
		}
	}()

	time.Sleep(2100 * time.Millisecond)
	quit <- true
	wg.Wait()
}
