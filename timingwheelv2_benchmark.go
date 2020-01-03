package timingwheelv2

import (
	"sync"
	"testing"
)

type Dog struct {
}

func (o *Dog) Release() {

}

func BenchmarkTimingWheel_AddItem(b *testing.B) {
	slotCnt := 100
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		b.Errorf("NewTimingWheel return error[%v]", err)
	}

	quit := make(chan bool)
	quitCh := func() chan bool {
		return quit
	}

	wg := &sync.WaitGroup{}
	deferFunc := func() {
		//log.Println("before wg.Done()")
		wg.Done()
	}

	wg.Add(1)
	go w.Run(quitCh, deferFunc)

	for i := 0; i < b.N; i++ {
		w.AddItem(&Dog{})
	}

	close(quit)
	wg.Wait()
}
