// Code generated by "stringer -type=Status"; DO NOT EDIT.

package status

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Waiting-0]
	_ = x[Running-1]
	_ = x[Done-2]
	_ = x[Timeout-3]
	_ = x[Canceled-4]
	_ = x[Error-5]
}

const _Status_name = "WaitingRunningDoneTimeoutCanceledError"

var _Status_index = [...]uint8{0, 7, 14, 18, 25, 33, 38}

func (i Status) String() string {
	if i < 0 || i >= Status(len(_Status_index)-1) {
		return "Status(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Status_name[_Status_index[i]:_Status_index[i+1]]
}
