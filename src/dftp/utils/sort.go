package utils

import (
	"reflect"
	"sort"
)

// Штуки, упрощающие сортировку
//

// Sort a slice in-place given the provided less comparator function.
//
// For example:
//
//    // (FWIW, sort.Ints(...) will not work with a []NotAnInt)
//    type NotAnInt int
//    s := []NotAnInt{3, 4, 8, 5, 2}
//
//    slicesorter.SortSlice(s, func(l, r interface{}) bool {
//      return l.(NotAnInt) > r.(NotAnInt)
//    })
//
//    // Prints "[8, 5, 4, 3, 2]":
//    fmt.Println(s)
//
// Утащено с http://blog.bensigelman.org/post/101709831324/sort-slices-in-go-without-cursing-under-your
func SortSlice(inputSlice interface{}, lessFunc func(l, r interface{}) bool) {
	sortable := sortableSlice{inputSlice, lessFunc}
	sort.Sort(sortable)
}

// The internal type that binds everything together.
type sortableSlice struct {
	inputSlice interface{}
	lessFunc   func(l interface{}, r interface{}) bool
}

// A stupid helper.
func (s sortableSlice) reflectElems(i, j int) (elemI, elemJ reflect.Value) {
	sliceVal := reflect.ValueOf(s.inputSlice)
	elemI = sliceVal.Index(i)
	elemJ = sliceVal.Index(j)
	return
}

///////////////////////////////
// The sort.Interface contract:
///////////////////////////////

func (s sortableSlice) Len() int { return reflect.ValueOf(s.inputSlice).Len() }

func (s sortableSlice) Swap(i, j int) {
	// Cue the preposterous reflection dance!
	elemI, elemJ := s.reflectElems(i, j)
	iInt := elemI.Interface()
	jInt := elemJ.Interface()
	// (Thankfully, Set() knows about slice indices)
	elemI.Set(reflect.ValueOf(jInt))
	elemJ.Set(reflect.ValueOf(iInt))
}

func (s sortableSlice) Less(i, j int) bool {
	elemI, elemJ := s.reflectElems(i, j)
	return s.lessFunc(elemI.Interface(), elemJ.Interface())
}
