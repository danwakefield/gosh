package main

import "sort"

type SparseArray struct {
	Elems map[int]string
	// XXX: Run benchmarks on memory / speed of using
	// Filled instead of just Elems.
	// As far a I can see it will only really be slicing that benifits
	// and even then it will need to sort first.
	FilledElems []int

	// Stops repeated sorts on the FilledElems array,
	// becomes false when Put is called with a new element.
	// Consider using sorted binary tree for FilledElems.
	// Worse case on more common activities might not be worth it.
	sorted bool
}

func (sa SparseArray) Len() int {
	return len(sa.FilledElems)
}

func (sa *SparseArray) Map(f func(string) string) {
	for _, e := range sa.FilledElems {
		sa.Elems[e] = f(sa.Elems[e])
	}
}

func (sa *SparseArray) Put(key int, val string) {
	_, ok := sa.Elems[key]
	if !ok {
		sa.FilledElems = append(sa.FilledElems, key)
		sa.sorted = false
	}
	sa.Elems[key] = val
}

func (sa *SparseArray) Get(key int) string {
	return sa.Elems[key]
}

func (sa *SparseArray) Delete(key int) {
	delete(sa.Elems(key))
}

func (sa *SparseArray) SliceFrom(from int) SparseArray {
	if !sa.sorted {
		sort.Ints(sa.FilledElems)
		sa.sorted = true
	}
	newSa = SparseArray{}
	// Since we are adding elements in order from a sorted array
	// the new SpareseArray will also be sorted.
	newSa.sorted = true
	for _, e := sa.FilledElems {
		if e >= from {
			newSa.Put(e, sa.Elems[e])
		}
	}
      return newSa
}

func (sa *SparseArray) SliceFromLen(from, length int) SparseArray {
	if !sa.sorted {
		sort.Ints(sa.FilledElems)
		sa.sorted = true
	}
	newSa = SparseArray{}
	newSa.sorted = true
	for _, e := sa.FilledElems {
		if e >= from && e < from+length{
			newSa.Put(e, sa.Elems[e])
		}
	}
      return newSa
}

func (sa *SparseArray) Shift(n int) {

}

func (sa *SparseArray) RShift(n int) {}
