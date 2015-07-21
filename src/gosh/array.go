package main

import "sort"

// XXX: Seperate out into a package? Adds stutter to types. Array.SparseArray
// XXX: Consider using this as both array and map type. Base methods would
// accept string keys, Array methods would wrap and call Itoa or similar wiht
// error checking?

type SparseArray struct {
	elems map[int]string
	// XXX: Run benchmarks on memory / speed of using
	// Filled instead of just Elems.
	// As far a I can see it will only really be slicing that benifits
	// and even then it will need to sort first.
	filledElems []int

	// Stops repeated sorts on the FilledElems array,
	// becomes false when Put is called with a new element.
	// Consider using sorted binary tree for FilledElems.
	// Worse case on more common activities might not be worth it.
	sorted bool
}

func NewSparseArray() SparseArray {
	sa := SparseArray{}
	sa.elems = map[int]string{}
	sa.filledElems = []int{}
	return sa
}

func (sa SparseArray) Len() int {
	return len(sa.filledElems)
}

func (sa *SparseArray) Map(f func(string) string) {
	for _, e := range sa.filledElems {
		sa.elems[e] = f(sa.elems[e])
	}
}

func (sa *SparseArray) Put(key int, val string) {
	_, found := sa.elems[key]
	if !found {
		sa.filledElems = append(sa.filledElems, key)
		sa.sorted = false
	}
	sa.elems[key] = val
}

func (sa *SparseArray) Get(key int) string {
	return sa.elems[key]
}

func (sa *SparseArray) Delete(key int) {
	_, found := sa.elems[key]
	if found {
		// delete is a noop by itself but we dont want to require O(n)
		// when not even deleting something.
		delete(sa.elems, key)

		// remove the key from filledElems.
		// because of the b slice uses the same memory
		// as filledElems there is no need to allocate.
		b := sa.filledElems[:0]
		for i, e := range sa.filledElems {
			if e == key {
				b = append(b, sa.filledElems[i+1:]...)
				break
			}
			b = append(b, e)
		}
		sa.filledElems = b
	}
}

// SliceFrom creates a new SparseArray containing all the elems after
// index 'from'. The retrned SparseArray will be in a sorted state.
func (sa *SparseArray) SliceFrom(from int) SparseArray {
	sa.sort()

	newSa := NewSparseArray()
	// Since we are adding elements in order from a sorted array
	// the new SpareseArray will also be sorted.
	newSa.sorted = true
	for _, e := range sa.filledElems {
		if e >= from {
			newSa.Put(e, sa.elems[e])
		}
	}
	return newSa
}

// SliceFormLen create a new SparseArray containing elems starting at
// index 'from' and ending at index 'from + length'.
// The retrned SparseArray will be in a sorted state
func (sa *SparseArray) SliceFromLen(from, length int) SparseArray {
	sa.sort()

	newSa := NewSparseArray()
	newSa.sorted = true
	for _, e := range sa.filledElems {
		if e >= from && e < from+length {
			newSa.Put(e, sa.elems[e])
		}
	}
	return newSa
}

// Shift removes the element at index 0 and moves all other
// elements down 1 index.

// The variables in bash would be
//   $1=1 $2=2 $3=3
//   $@=(1 2 3)
// after a shift of 1 (this is the default for a bash shift) they will become
//   $1=2 $2=3
//   $@=(2 3)
// shifts of n > 1 are just repeated applications of the shift(1) case
//
// This is only used in argv for a scripts and functions.
// because of this the array will only be empty or contain contiguous
// elements.
func (sa *SparseArray) Shift(n int) {
	if n <= 0 || sa.Len() == 0 {
		return
	}
	if n > sa.Len() {
		sa.elems = map[int]string{}
		sa.filledElems = []int{}
		return
	}
	sa.sort()

	for i := 0; i < n; i++ {
		sa.Delete(0)
		for j := 0; j < sa.Len()-1; j++ {
			sa.Put(j, sa.Get(j+1))
		}
		sa.Delete(sa.Len() - 1)
	}
	// Put clears the sorted state but since we sorted before the shifts
	// the Array will still be sorted.
	sa.sorted = true
}

func (sa SparseArray) String() string { return "" }

func (sa *SparseArray) sort() {
	if !sa.sorted {
		sort.Ints(sa.filledElems)
		sa.sorted = true
	}
}
