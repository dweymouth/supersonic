package util

type ListSelectionManager struct {
	lastSelectedRow int
	numSelected     int
	selected        BitSet
	len             func() int
}

func NewListSelectionManager(lenFn func() int) ListSelectionManager {
	return ListSelectionManager{lastSelectedRow: -1, len: lenFn}
}

// If the given row is not selected, reset the selection
// and select only the given row.
func (l *ListSelectionManager) Select(row int) {
	if row < 0 || l.selected.IsSet(uint(row)) {
		return
	}
	l.UnselectAll()
	l.selectAdd(row)
}

// If it is not selected, add the given row to the selection.
// If it is selected, unselet it.
func (l *ListSelectionManager) SelectAddOrRemove(row int) {
	if row < 0 {
		return
	}
	if l.selected.IsSet(uint(row)) {
		// find new last selected row
		if row == l.lastSelectedRow {
			for i := row - 1; i >= -1; i++ {
				if i == -1 {
					l.lastSelectedRow = i
				} else if l.selected.IsSet(uint(i)) {
					l.lastSelectedRow = i
					break
				}
			}
		}
		l.selected.Unset(uint(row))
		l.numSelected -= 1
		return
	}

	l.selectAdd(row)
}

func (l *ListSelectionManager) selectAdd(row int) {
	l.numSelected += 1
	l.selected.Set(uint(row))
	if row > l.lastSelectedRow {
		l.lastSelectedRow = row
	}
}

// Select a range between the given row and the furthest-down
// row that is currently selected (which may be above the given row)
// Note: this is modeled after what, as far as I can tell, is Gmail's selection behavior
func (l *ListSelectionManager) SelectRange(row int) {
	if row < 0 || l.selected.IsSet(uint(row)) {
		return
	}
	if l.numSelected == 0 {
		l.selectAdd(row)
		return
	}
	m := maxInt(row, l.lastSelectedRow)
	for i := minInt(l.lastSelectedRow, row); i <= m; i++ {
		l.selectAdd(i)
	}
}

func (l *ListSelectionManager) SelectAll() {
	for i := 0; i < l.len(); i++ {
		l.selectAdd(i)
	}
}

func (l *ListSelectionManager) UnselectAll() {
	l.selected = nil
	l.numSelected = 0
	l.lastSelectedRow = -1
}

func (l *ListSelectionManager) IsSelected(row int) bool {
	return row >= 0 && l.selected.IsSet(uint(row))
}

func (l *ListSelectionManager) GetSelection() []int {
	var sel []int
	for i := 0; i < l.len(); i++ {
		if l.selected.IsSet(uint(i)) {
			sel = append(sel, i)
		}
		if len(sel) == l.numSelected {
			break
		}
	}
	return sel
}

func (l *ListSelectionManager) AreAllSelected() bool {
	return l.numSelected == l.len()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BitSet implementation from
// https://stackoverflow.com/questions/2311373/how-to-implement-bitset-with-go

const uint64size = 64

// BitSet is a set of bits that can be set, cleared and queried.
type BitSet []uint64

// Set ensures that the given bit is set in the BitSet.
func (s *BitSet) Set(i uint) {
	if len(*s) < int(i/uint64size+1) {
		r := make([]uint64, i/uint64size+1)
		copy(r, *s)
		*s = r
	}
	(*s)[i/uint64size] |= 1 << (i % uint64size)
}

// Unset ensures that the given bit is cleared (not set) in the BitSet.
func (s *BitSet) Unset(i uint) {
	if len(*s) >= int(i/uint64size+1) {
		(*s)[i/uint64size] &^= 1 << (i % uint64size)
	}
}

// IsSet returns true if the given bit is set, false if it is cleared.
func (s *BitSet) IsSet(i uint) bool {
	idx := i / uint64size
	if idx >= uint(len(*s)) {
		return false
	}
	return (*s)[i/uint64size]&(1<<(i%uint64size)) != 0
}
