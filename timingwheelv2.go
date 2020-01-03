package timingwheelv2

import (
	"errors"
	"sync"
	"time"
)

var (
	errDurationTooSmall = errors.New("duration too small")
	errInvalidDuration  = errors.New("invalid duration")
	errInvalidSlotCount = errors.New("invalid slot count")
)

// Releaser is a interface decribes objects which can release.
type Releaser interface {
	Release()
}

type itemCon map[Releaser]bool
type slotCon []itemCon

// TimingWheel is a data structure that manages some items which may be out of time soon.
type TimingWheel struct {
	stepTime time.Duration // Every step time, the wheel rolls one index forward.

	mux      sync.Mutex // protect slots, curIndex and item2Idx
	slots    slotCon
	curIndex int
	item2Idx map[Releaser]int
}

// NewTimingWheel returns a TimingWheel instance.
// Parameter max is the cycle of every item get checked.
// Parameter slotCnt is the slot count of the wheel. Every slot can contain a few items.
// max and slotCnt MUST be positive.
// max MUST >= slotCnt. It is the best way that max%slotCnt == 0.
func NewTimingWheel(max time.Duration, slotCnt int) (*TimingWheel, error) {
	if max <= 0 {
		return nil, errInvalidDuration
	}
	if slotCnt <= 0 {
		return nil, errInvalidSlotCount
	}

	n := max.Nanoseconds()
	c := int64(slotCnt)
	if n < c {
		return nil, errDurationTooSmall
	}
	s := n / c
	if n%c > 0 {
		s++
	}

	slots := make(slotCon, slotCnt)
	for i := range slots {
		slots[i] = make(itemCon)
	}

	tw := &TimingWheel{
		stepTime: time.Duration(s),
		slots:    slots,
		item2Idx: make(map[Releaser]int),
	}
	return tw, nil
}

// AddItem adds an item to this wheel.
func (tw *TimingWheel) AddItem(item Releaser) {
	var toAdd bool
	tw.mux.Lock()
	i, ok := tw.item2Idx[item]
	if ok {
		if i != tw.curIndex {
			delete(tw.slots[i], item)
			toAdd = true
		}
	} else {
		toAdd = true
	}
	if toAdd {
		tw.slots[tw.curIndex][item] = true
		tw.item2Idx[item] = tw.curIndex
	}
	tw.mux.Unlock()
}

// DelItem deletes a item from this wheel. But it does not release this item.
func (tw *TimingWheel) DelItem(item Releaser) {
	tw.mux.Lock()
	if i, ok := tw.item2Idx[item]; ok {
		delete(tw.slots[i], item)
		delete(tw.item2Idx, item)
	}
	tw.mux.Unlock()
}

// Run rolls this wheel
func (tw *TimingWheel) Run(quitCh func() chan bool, deferFunc func()) {
	defer deferFunc()
	timer := time.NewTimer(tw.stepTime)
	defer timer.Stop()
	for {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(tw.stepTime)

		select {
		case <-quitCh():
			return
		case <-timer.C:
			tw.stepForward()
		}
	}
}

// move one index forward and check items in new index.
func (tw *TimingWheel) stepForward() {
	tw.mux.Lock()
	idx := tw.curIndex + 1
	if idx >= len(tw.slots) {
		idx = 0
	}
	tw.curIndex = idx
	curSlot := tw.slots[idx]
	for k := range curSlot {
		delete(tw.item2Idx, k)
		k.Release()
	}
	tw.slots[idx] = make(itemCon)
	tw.mux.Unlock()
}

type Observer interface {
	beforeStep()
	afterStep()
	afterRelease()
	afterMove()
}

// runWithStepObserver needs two step observers.
// This function is for ease of unit test.
func (tw *TimingWheel) runWithStepObserver(quitCh func() chan bool, deferFunc func(), ob Observer) {
	defer deferFunc()
	timer := time.NewTimer(tw.stepTime)
	defer timer.Stop()
	for {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(tw.stepTime)

		select {
		case <-quitCh():
			return
		case <-timer.C:
			ob.beforeStep()
			tw.stepForwardWithObserver(ob)
			ob.afterStep()
		}
	}
}

func (tw *TimingWheel) stepForwardWithObserver(ob Observer) {
	tw.mux.Lock()
	idx := tw.curIndex + 1
	if idx >= len(tw.slots) {
		idx = 0
	}
	tw.curIndex = idx
	curSlot := tw.slots[idx]
	for k := range curSlot {
		delete(tw.item2Idx, k)
		k.Release()
		ob.afterRelease()
	}
	tw.slots[idx] = make(itemCon)
	tw.mux.Unlock()
}

// itemCount returns the item count in this wheel.
// This function is just for unit test.
func (tw *TimingWheel) itemCount() int {
	var cnt int
	tw.mux.Lock()
	cnt = len(tw.item2Idx)
	tw.mux.Unlock()
	return cnt
}
