package websocket

func isControlFrame(OpCode byte) bool {
	return OpCode != PayloadTypeText && OpCode != PayloadTypeBinary
}

func getFragmentStatus(pos, size, length int) int {
	if pos+size >= length {
		return FragmentEnd
	}
	if pos <= 0 {
		return FragmentBegin
	}
	return FragmentMiddle
}
