package types

import (
	"reflect"
	"strconv"
	"testing"
)

func TestSparseArrayFromSlice(t *testing.T) {
	a := []string{"a", "b", "c", "d"}
	expectedElems := map[int]string{
		0: "a",
		1: "b",
		2: "c",
		3: "d",
	}

	sa := NewSparseArrayWithElems(a)

	if !reflect.DeepEqual(expectedElems, sa.elems) {
		t.Errorf("Creating new SparseArray with elems failed. expected %#v, got %#v", expectedElems, sa.elems)
	}
}

func TestSparseArrayLen(t *testing.T) {
	sa := NewSparseArray()

	sa.Put(1, "")
	if sa.Len() != 1 {
		t.Errorf("SparseArray.Len() should be 1, got %d", sa.Len())
	}

	sa.Put(2, "")
	sa.Put(3, "")
	sa.Put(4, "")
	if sa.Len() != 4 {
		t.Errorf("SparseArray.Len() should be 4, got %d", sa.Len())
	}

	sa.Put(4, "")
	sa.Put(4, "")
	if sa.Len() != 4 {
		t.Errorf("SparseArray.Len() should be 4, got %d", sa.Len())
	}

	sa.Delete(1)
	if sa.Len() != 3 {
		t.Errorf("SparseArray.Len() should be 3, got %d", sa.Len())
	}

	// Check that repeated deletes dont affect other elements.
	sa.Delete(1)
	if sa.Len() != 3 {
		t.Errorf("SparseArray.Len() should be 3, got %d", sa.Len())
	}
}

func TestSparseArrayPutGet(t *testing.T) {
	sa := NewSparseArray()

	sa.Put(1, "a")
	v := sa.Get(1)
	if v != "a" {
		t.Errorf("Put Get expected 'a', got '%s'", v)
	}

	sa.Put(1, "b")
	v = sa.Get(1)
	if v != "b" {
		t.Errorf("Put Get expected 'b', got '%s'", v)
	}
}

func TestSparseArrayDelete(t *testing.T) {
	sa := NewSparseArray()
	sa.Put(1, "1")
	sa.Put(2, "2")
	sa.Put(3, "3")
	sa.Put(4, "4")

	sa.Delete(2)
	v := sa.Get(2)
	if v != "" {
		t.Errorf("Deleted element 2 but get returned a non empty result '%s'", v)
	}

	sa.Delete(4)
	v = sa.Get(4)
	if v != "" {
		t.Errorf("Deleted element 4 but get returned a non empty result '%s'", v)
	}
}

func TestSparseArraySort(t *testing.T) {
	sa := NewSparseArray()
	unsorted := []int{5, 3, 1, 2, 4}
	sorted := []int{1, 2, 3, 4, 5}
	for _, e := range unsorted {
		sa.Put(e, "")
	}

	if !reflect.DeepEqual(sa.filledElems, unsorted) {
		t.Errorf("Elements dont have insert order before sort. Expected %#v, got %#v", unsorted, sa.filledElems)
	}

	sa.sort()
	if !reflect.DeepEqual(sa.filledElems, sorted) {
		t.Errorf("Elements not sorted after sort. Expected %#v, got %#v", sorted, sa.filledElems)
	}
}

func TestSparseArraySliceFrom(t *testing.T) {
	sa := NewSparseArray()
	for i := 0; i < 10; i++ {
		sa.Put(i, strconv.Itoa(i))
	}
	expectedElems := map[int]string{
		5: "5",
		6: "6",
		7: "7",
		8: "8",
		9: "9",
	}

	sliced := sa.SliceFrom(5)
	if sliced.Len() != 5 {
		t.Errorf("expected sliced array to contain 5 elems, has %d", sliced.Len())
	}

	if !reflect.DeepEqual(expectedElems, sliced.elems) {
		t.Errorf("sliced array does not contain the correct elements. expected %#v, got %#v", expectedElems, sliced.elems)
	}
}

func TestSparseArraySliceFromSparse(t *testing.T) {
	sa := NewSparseArray()
	for i := 0; i < 10; i++ {
		sa.Put(i, strconv.Itoa(i))
	}
	sa.Delete(6)
	sa.Delete(8)
	expectedElems := map[int]string{
		5: "5",
		7: "7",
		9: "9",
	}

	sliced := sa.SliceFrom(5)
	if sliced.Len() != 3 {
		t.Errorf("expected sliced array to contain 3 elems, has %d", sliced.Len())
	}

	if !reflect.DeepEqual(expectedElems, sliced.elems) {
		t.Errorf("sliced array does not contain the correct elements. expected %#v, got %#v", expectedElems, sliced.elems)
	}
}

func TestSparseArraySliceFromLen(t *testing.T) {
	sa := NewSparseArray()
	for i := 0; i < 10; i++ {
		sa.Put(i, strconv.Itoa(i))
	}
	expectedElems := map[int]string{
		5: "5",
		6: "6",
		7: "7",
	}

	sliced := sa.SliceFromLen(5, 3)
	if sliced.Len() != 3 {
		t.Errorf("expected sliced array to contain 3 elems, has %d", sliced.Len())
	}

	if !reflect.DeepEqual(expectedElems, sliced.elems) {
		t.Errorf("sliced array does not contain the correct elements. expected %#v, got %#v", expectedElems, sliced.elems)
	}
}

func TestSparseArrayShift(t *testing.T) {
	sa := NewSparseArray()
	for i := 0; i < 5; i++ {
		sa.Put(i, strconv.Itoa(i))
	}

	sa.Shift(0)
	if sa.Len() != 5 {
		t.Errorf("shift(0) should be a noop but length of array has changed")
	}

	sa.Shift(1)
	if sa.Len() != 4 {
		t.Errorf("shift(1) should make the sa.Len() == 4, got %d", sa.Len())
	}

	expectedElems := map[int]string{
		0: "1",
		1: "2",
		2: "3",
		3: "4",
	}
	if !reflect.DeepEqual(expectedElems, sa.elems) {
		t.Errorf("after shift(1), expected %#v, got %#v", expectedElems, sa.elems)
	}

	sa.Shift(2)
	expectedElems = map[int]string{
		0: "3",
		1: "4",
	}
	if !reflect.DeepEqual(expectedElems, sa.elems) {
		t.Errorf("after shift(1) and shift(2), expected %#v, got %#v", expectedElems, sa.elems)
	}

	sa.Shift(3)
	if sa.Len() != 0 {
		t.Errorf("after shift(1), shift(2) and shift(3) array should be empty. contains %#v", sa.elems)
	}
}

func TestSparseArrayMap(t *testing.T) {
	sa := NewSparseArray()
	d := "done"

	replaced := []int{0, 4, 15, 19}
	for _, e := range replaced {
		sa.Put(e, "replace")
	}
	unaffected := []int{2, 6, 7, 17}
	for _, e := range unaffected {
		sa.Put(e, strconv.Itoa(e))
	}
	expectedElems := map[int]string{
		0:  d,
		4:  d,
		15: d,
		19: d,

		2:  "2",
		6:  "6",
		7:  "7",
		17: "17",
	}

	sa.Map(func(s string) string {
		if s == "replace" {
			return d
		}
		return s
	})

	if !reflect.DeepEqual(expectedElems, sa.elems) {
		t.Errorf("elems not as expected after map. expected\n %#v\n got\n %#v", expectedElems, sa.elems)
	}
}
